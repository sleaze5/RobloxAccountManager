package appservice

import (
	"cmp"
	"context"
	"crypto/rand"
	"math"
	"slices"

	"github.com/sleaze5/RobloxAccountManager/internal/games"
	"github.com/sleaze5/RobloxAccountManager/internal/integration/rovalra"
	"github.com/sleaze5/RobloxAccountManager/internal/logging"
)

// Each listed server costs one Roblox join check, so a region lists a small
// batch at a time instead of its whole page.
const regionServerBatchSize = 5

func (service *Service) ListServerRegions(ctx context.Context, refresh bool) ([]games.Region, error) {
	ctx = logging.WithOperation(ctx, "rovalra-"+rand.Text())
	regions, err := service.serverRegions(ctx, refresh)
	if err != nil {
		return nil, err
	}
	preferred := service.settings.Integrations().RoValraRegion
	gameRegions := make([]games.Region, len(regions))
	listed := preferred == ""
	for index, region := range regions {
		gameRegions[index] = region.Region
		listed = listed || region.Code == preferred
	}
	if !listed {
		service.logs.Module("integration.rovalra").WarnContext(ctx, "preferred region is no longer listed", "operation", "rovalra-regions", "region", preferred)
	}
	return gameRegions, nil
}

func (service *Service) SetRoValraRegion(ctx context.Context, code string) error {
	ctx = logging.WithOperation(ctx, "rovalra-"+rand.Text())
	regions, err := service.serverRegions(ctx, false)
	if err != nil {
		return err
	}
	if !slices.ContainsFunc(regions, func(region rovalra.Region) bool { return region.Code == code }) {
		return gamesInputError("Choose a region from the list.")
	}
	if err := service.settings.SetRoValraRegion(code); err != nil {
		service.logger.WarnContext(ctx, "could not save integration settings", "operation", "integration-settings", "integration", "rovalra", "error", err)
		return err
	}
	service.logger.InfoContext(ctx, "integration settings saved", "operation", "integration-settings", "integration", "rovalra", "region", code)
	return nil
}

func (service *Service) serverRegions(ctx context.Context, refresh bool) ([]rovalra.Region, error) {
	if !service.settings.Integrations().RoValra {
		return nil, gamesInputError("Turn on the RoValra integration in Settings to use RoValra servers.")
	}
	regions, err := service.rovalra.Regions(ctx, refresh)
	if err != nil {
		return nil, service.rovalraError(ctx, "rovalra-regions", 0, err)
	}
	return regions, nil
}

func (service *Service) ListNearbyServerRegions(ctx context.Context, placeID int64) ([]games.NearbyRegion, error) {
	if err := validGameID(placeID); err != nil {
		return nil, err
	}
	ctx, done, err := service.beginRoValra(ctx)
	if err != nil {
		return nil, err
	}
	defer done()
	regions, err := service.serverRegions(ctx, false)
	if err != nil {
		return nil, err
	}
	preferred := service.settings.Integrations().RoValraRegion
	origin := slices.IndexFunc(regions, func(region rovalra.Region) bool { return region.Code == preferred })
	if origin < 0 {
		return nil, gamesInputError("Choose a preferred region in Settings under Integrations.")
	}
	stats, err := service.rovalra.Stats(ctx, placeID)
	if err != nil {
		return nil, service.rovalraError(ctx, "rovalra-stats", placeID, err)
	}
	return nearbyRegions(regions, regions[origin], stats), nil
}

func (service *Service) ListRegionServers(ctx context.Context, query games.RegionServerQuery) ([]string, error) {
	if err := validGameID(query.PlaceID); err != nil {
		return nil, err
	}
	ctx, done, err := service.beginRoValra(ctx)
	if err != nil {
		return nil, err
	}
	defer done()
	regions, err := service.serverRegions(ctx, false)
	if err != nil {
		return nil, err
	}
	if !slices.ContainsFunc(regions, func(region rovalra.Region) bool { return region.Code == query.Region }) {
		return nil, gamesInputError("Choose a region from the list.")
	}
	page, err := service.rovalra.Servers(ctx, games.RecordQuery{
		PlaceID: query.PlaceID, Region: query.Region, Order: games.RecordOrderNewest, Limit: 100,
	})
	if err != nil {
		return nil, service.rovalraError(ctx, "rovalra-region-servers", query.PlaceID, err)
	}
	jobIDs := make([]string, 0, regionServerBatchSize)
	for _, server := range page.Servers {
		if len(jobIDs) == regionServerBatchSize {
			break
		}
		if !slices.Contains(query.ExcludedJobIDs, server.JobID) {
			jobIDs = append(jobIDs, server.JobID)
		}
	}
	return jobIDs, nil
}

func (service *Service) CheckGameServerJoinable(ctx context.Context, placeID, accountID int64, jobID string) (bool, error) {
	if err := validGameID(placeID); err != nil {
		return false, err
	}
	if !games.ValidJobID(jobID) {
		return false, gamesInputError("The server is invalid. Search again.")
	}
	ctx, done, err := service.beginGames(ctx)
	if err != nil {
		return false, err
	}
	defer done()
	joinable, err := service.games.Joinable(ctx, accountID, placeID, jobID)
	if err != nil {
		return false, service.gamesError(ctx, "game-server-join-check", err)
	}
	return joinable, nil
}

// nearbyRegions lists the regions that have servers of the place, with the
// origin first and the others from nearest to farthest.
func nearbyRegions(regions []rovalra.Region, origin rovalra.Region, stats games.ServerStats) []games.NearbyRegion {
	hasServersByRegionCode := make(map[string]bool, len(stats.Regions))
	for _, statsRegion := range stats.Regions {
		hasServersByRegionCode[statsRegion.Code] = true
	}
	nearby := make([]games.NearbyRegion, 0, len(hasServersByRegionCode))
	for _, region := range regions {
		if !hasServersByRegionCode[region.Code] || region.Code == origin.Code {
			continue
		}
		nearby = append(nearby, games.NearbyRegion{Region: region.Region, DistanceKilometers: regionDistanceKilometers(origin, region)})
	}
	slices.SortStableFunc(nearby, func(left, right games.NearbyRegion) int {
		return cmp.Compare(left.DistanceKilometers, right.DistanceKilometers)
	})
	if hasServersByRegionCode[origin.Code] {
		nearby = slices.Insert(nearby, 0, games.NearbyRegion{Region: origin.Region})
	}
	return nearby
}

func regionDistanceKilometers(from, to rovalra.Region) float64 {
	distance := math.Inf(1)
	for _, fromCoordinates := range from.Coordinates {
		for _, toCoordinates := range to.Coordinates {
			distance = min(distance, distanceKilometers(fromCoordinates, toCoordinates))
		}
	}
	return distance
}

func distanceKilometers(from, to rovalra.Coordinates) float64 {
	const earthRadiusKilometers = 6371
	radians := math.Pi / 180
	latitudeDelta, longitudeDelta := (to.Latitude-from.Latitude)*radians, (to.Longitude-from.Longitude)*radians
	haversine := math.Sin(latitudeDelta/2)*math.Sin(latitudeDelta/2) + math.Cos(from.Latitude*radians)*math.Cos(to.Latitude*radians)*math.Sin(longitudeDelta/2)*math.Sin(longitudeDelta/2)
	haversine = min(1, max(0, haversine))
	return 2 * earthRadiusKilometers * math.Atan2(math.Sqrt(haversine), math.Sqrt(1-haversine))
}

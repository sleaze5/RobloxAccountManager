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

const (
	// Join checks are bounded like RoValra's own server finder, so a place full of closed records cannot stall a search.
	nearestServerChecksPerRegion = 5
	nearestServerChecks          = 15
)

// ListServerRegions returns RoValra's server-browser regions, grouped by country with the US first.
// They hold no account data, so they are available while the vault is locked.
func (service *Service) ListServerRegions(ctx context.Context, refresh bool) ([]games.Region, error) {
	ctx = logging.WithOperation(ctx, "rovalra-"+rand.Text())
	regions, err := service.serverRegions(ctx, refresh)
	if err != nil {
		return nil, err
	}
	preferred := service.settings.Integrations().RoValraRegion
	result := make([]games.Region, len(regions))
	listed := preferred == ""
	for index, region := range regions {
		result[index] = region.Region
		listed = listed || region.Code == preferred
	}
	if !listed {
		service.logs.Module("integration.rovalra").WarnContext(ctx, "preferred region is no longer listed", "operation", "rovalra-regions", "region", preferred)
	}
	return result, nil
}

// SetRoValraRegion saves a region code shared with the server browser.
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

// FindNearestGameServer returns the newest server RoValra recorded for a place in the preferred region, or else in the closest region
// that has one. A positive accountID skips servers that do not accept that account.
func (service *Service) FindNearestGameServer(ctx context.Context, placeID, accountID int64) (games.NearestServer, error) {
	if err := validGameID(placeID); err != nil {
		return games.NearestServer{}, err
	}
	if accountID < 0 {
		return games.NearestServer{}, gamesInputError("The account is invalid.")
	}
	ctx, done, err := service.beginRoValra(ctx)
	if err != nil {
		return games.NearestServer{}, err
	}
	defer done()
	regions, err := service.serverRegions(ctx, false)
	if err != nil {
		return games.NearestServer{}, err
	}
	preferred := service.settings.Integrations().RoValraRegion
	origin := slices.IndexFunc(regions, func(region rovalra.Region) bool { return region.Code == preferred })
	if origin < 0 {
		return games.NearestServer{}, gamesInputError("Choose a preferred region in Settings under Integrations.")
	}
	stats, err := service.rovalra.Stats(ctx, placeID)
	if err != nil {
		return games.NearestServer{}, service.rovalraError(ctx, "rovalra-stats", placeID, err)
	}
	return service.nearestServer(ctx, placeID, accountID, regionsByDistance(regions, regions[origin], stats))
}

// nearestServer searches regions in order and returns the first server found.
func (service *Service) nearestServer(ctx context.Context, placeID, accountID int64, regions []rovalra.Region) (games.NearestServer, error) {
	checksLeft := nearestServerChecks
	for searched, region := range regions {
		page, err := service.rovalra.Servers(ctx, games.RecordQuery{
			PlaceID: placeID, Region: region.Code, Order: games.RecordOrderNewest, Limit: 100,
		})
		if err != nil {
			return games.NearestServer{}, service.rovalraError(ctx, "rovalra-region-servers", placeID, err)
		}
		jobID, err := service.firstJoinable(ctx, placeID, accountID, page.Servers, &checksLeft)
		if err != nil {
			return games.NearestServer{}, service.gamesError(ctx, "game-server-join-check", err)
		}
		if jobID != "" {
			service.logs.Module("integration.rovalra").DebugContext(ctx, "nearest server found", "operation", "rovalra-nearest-server", "place_id", placeID, "region", region.Code, "regions_searched", searched+1, "join_checks", nearestServerChecks-checksLeft)
			return games.NearestServer{JobID: jobID, Region: region.Region}, nil
		}
		if checksLeft == 0 {
			break
		}
	}
	return games.NearestServer{}, gamesInputError("No open server of this place was found. Try again later.")
}

// firstJoinable returns the newest server that accepts the account, or the newest server when no account checks them.
func (service *Service) firstJoinable(ctx context.Context, placeID, accountID int64, servers []games.ServerRecord, checksLeft *int) (string, error) {
	if accountID == 0 {
		if len(servers) == 0 {
			return "", nil
		}
		return servers[0].JobID, nil
	}
	for _, server := range servers[:min(len(servers), nearestServerChecksPerRegion)] {
		if *checksLeft == 0 {
			break
		}
		*checksLeft--
		joinable, err := service.games.Joinable(ctx, accountID, placeID, server.JobID)
		if err != nil || joinable {
			return server.JobID, err
		}
	}
	return "", nil
}

// regionsByDistance prioritizes the preferred region, then the nearest regions with recorded servers.
func regionsByDistance(regions []rovalra.Region, origin rovalra.Region, stats games.ServerStats) []rovalra.Region {
	distances := make(map[string]float64, len(stats.Regions))
	for _, region := range stats.Regions {
		distances[region.Code] = 0
	}
	available := make([]rovalra.Region, 0, len(regions))
	for _, region := range regions {
		if _, exists := distances[region.Code]; exists {
			distances[region.Code] = regionDistanceKm(origin, region)
			available = append(available, region)
		}
	}
	distances[origin.Code] = -1
	slices.SortStableFunc(available, func(a, b rovalra.Region) int {
		return cmp.Compare(distances[a.Code], distances[b.Code])
	})
	return available
}

// Region proximity uses the nearest pair of datacenters across the two regions.
func regionDistanceKm(a, b rovalra.Region) float64 {
	distance := math.Inf(1)
	for _, left := range a.Coordinates {
		for _, right := range b.Coordinates {
			distance = min(distance, distanceKm(left, right))
		}
	}
	return distance
}

// distanceKm returns the great-circle distance between two locations with the haversine formula.
func distanceKm(a, b rovalra.Coordinates) float64 {
	const earthRadiusKm = 6371
	radians := math.Pi / 180
	dLat, dLon := (b.Latitude-a.Latitude)*radians, (b.Longitude-a.Longitude)*radians
	h := math.Sin(dLat/2)*math.Sin(dLat/2) + math.Cos(a.Latitude*radians)*math.Cos(b.Latitude*radians)*math.Sin(dLon/2)*math.Sin(dLon/2)
	h = min(1, max(0, h))
	return 2 * earthRadiusKm * math.Atan2(math.Sqrt(h), math.Sqrt(1-h))
}

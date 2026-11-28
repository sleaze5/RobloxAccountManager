package rovalra

import (
	"cmp"
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/sleaze5/RobloxAccountManager/internal/games"
)

const regionsKeep = 24 * time.Hour

type Region struct {
	games.Region

	Coordinates []Coordinates
}

type Coordinates struct {
	Latitude  float64
	Longitude float64
}

func (client *Client) Regions(ctx context.Context, refresh bool) ([]Region, error) {
	client.mutex.Lock()
	defer client.mutex.Unlock()
	if err := client.loadDatacenters(ctx, refresh); err != nil {
		return nil, err
	}
	return client.regions, nil
}

// Datacenter returns where RoValra places a Roblox datacenter, and false for a
// datacenter that RoValra does not list.
func (client *Client) Datacenter(ctx context.Context, datacenterID int64) (games.Datacenter, bool, error) {
	client.mutex.Lock()
	defer client.mutex.Unlock()
	if err := client.loadDatacenters(ctx, false); err != nil {
		return games.Datacenter{}, false, err
	}
	datacenter, found := client.datacenterByID[datacenterID]
	return datacenter, found, nil
}

// loadDatacenters reads RoValra's list of datacenters into the regions and
// the datacenter locations, unless the kept list is recent. The caller holds
// the mutex.
func (client *Client) loadDatacenters(ctx context.Context, refresh bool) error {
	if !refresh && client.regions != nil && time.Since(client.regionsFetchedAt) < regionsKeep {
		return nil
	}
	const path = "/v1/datacenters/list"
	code, body, err := client.fetch(ctx, path, nil)
	if err != nil {
		return err
	}
	if code != http.StatusOK {
		return fmt.Errorf("%s: HTTP %d", path, code)
	}
	var payload []struct {
		DatacenterIDs []int64 `json:"dataCenterIds"`
		Location      struct {
			City        string   `json:"city"`
			Region      string   `json:"region"`
			Country     string   `json:"country"`
			CountryName string   `json:"country_name"`
			LatLong     []string `json:"latLong"`
		} `json:"location"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return fmt.Errorf("%s: invalid data: %w", path, err)
	}
	regions := make([]Region, 0, len(payload))
	byCode := make(map[string]int, len(payload))
	datacenterByID := make(map[int64]games.Datacenter)
	for _, regionEntry := range payload {
		location := regionEntry.Location
		region, ok := newRegion(location.Region, location.CountryName, location.Country, location.LatLong)
		if !ok {
			continue
		}
		datacenter := games.Datacenter{City: text(location.City), Region: text(location.Region), Country: region.Country, CountryCode: region.CountryCode}
		if datacenter.City == datacenter.Region {
			datacenter.Region = ""
		}
		for _, datacenterID := range regionEntry.DatacenterIDs {
			if datacenterID > 0 {
				datacenterByID[datacenterID] = datacenter
			}
		}
		if index, exists := byCode[region.Code]; exists {
			regions[index].Coordinates = append(regions[index].Coordinates, region.Coordinates...)
		} else {
			byCode[region.Code] = len(regions)
			regions = append(regions, region)
		}
	}
	if len(regions) == 0 {
		return fmt.Errorf("%s: no server regions", path)
	}
	slices.SortFunc(regions, func(a, b Region) int {
		return cmp.Or(cmp.Compare(countryRank(a), countryRank(b)), cmp.Compare(a.Country, b.Country), cmp.Compare(a.Name, b.Name))
	})
	client.regions, client.datacenterByID, client.regionsFetchedAt = regions, datacenterByID, time.Now()
	return nil
}

func newRegion(name, country, countryCode string, coordinates []string) (Region, bool) {
	if len(coordinates) != 2 || len(countryCode) != 2 || !ValidRegionCode(countryCode) {
		return Region{}, false
	}
	latitude, latitudeError := strconv.ParseFloat(coordinates[0], 64)
	longitude, longitudeError := strconv.ParseFloat(coordinates[1], 64)
	if latitudeError != nil || longitudeError != nil || !validCoordinates(latitude, longitude) {
		return Region{}, false
	}
	if country = text(country); country == "" {
		country = countryCode
	}
	code := countryCode
	if countryCode == "US" {
		name = text(name)
		code += "-" + strings.ToUpper(name)
	} else {
		name = country
	}
	if name == "" || !ValidRegionCode(code) {
		return Region{}, false
	}
	return Region{
		Region:      games.Region{Code: code, Name: name, Country: country, CountryCode: countryCode},
		Coordinates: []Coordinates{{Latitude: latitude, Longitude: longitude}},
	}, true
}

func validCoordinates(latitude, longitude float64) bool {
	return !math.IsNaN(latitude) && !math.IsNaN(longitude) && latitude >= -90 && latitude <= 90 && longitude >= -180 && longitude <= 180
}

func countryRank(region Region) int {
	if region.CountryCode == "US" {
		return 0
	}
	return 1
}

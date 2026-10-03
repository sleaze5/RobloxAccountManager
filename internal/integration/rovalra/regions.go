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

// Region groups the datacenters covered by a server-browser region code.
type Region struct {
	games.Region
	Coordinates []Coordinates
}

type Coordinates struct {
	Latitude  float64
	Longitude float64
}

// Regions returns US states first, then other countries, with both groups sorted alphabetically.
// Refresh bypasses the cached list without discarding it if the request fails.
func (client *Client) Regions(ctx context.Context, refresh bool) ([]Region, error) {
	client.mu.Lock()
	defer client.mu.Unlock()
	if !refresh && client.regions != nil && time.Since(client.regionsFetchedAt) < regionsKeep {
		return client.regions, nil
	}
	const path = "/v1/datacenters/list"
	code, body, err := client.fetch(ctx, path, nil)
	if err != nil {
		return nil, err
	}
	if code != http.StatusOK {
		return nil, fmt.Errorf("%s: HTTP %d", path, code)
	}
	var payload []struct {
		Location struct {
			Region      string   `json:"region"`
			Country     string   `json:"country"`
			CountryName string   `json:"country_name"`
			LatLong     []string `json:"latLong"`
		} `json:"location"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("%s: invalid data: %w", path, err)
	}
	regions := make([]Region, 0, len(payload))
	byCode := make(map[string]int, len(payload))
	for _, item := range payload {
		region, ok := newRegion(item.Location.Region, item.Location.CountryName, item.Location.Country, item.Location.LatLong)
		if !ok {
			continue
		}
		if index, exists := byCode[region.Code]; exists {
			regions[index].Coordinates = append(regions[index].Coordinates, region.Coordinates...)
		} else {
			byCode[region.Code] = len(regions)
			regions = append(regions, region)
		}
	}
	if len(regions) == 0 {
		return nil, fmt.Errorf("%s: no server regions", path)
	}
	slices.SortFunc(regions, func(a, b Region) int {
		return cmp.Or(cmp.Compare(countryRank(a), countryRank(b)), cmp.Compare(a.Country, b.Country), cmp.Compare(a.Name, b.Name))
	})
	client.regions, client.regionsFetchedAt = regions, time.Now()
	return regions, nil
}

func newRegion(name, country, countryCode string, latLong []string) (Region, bool) {
	if len(latLong) != 2 || len(countryCode) != 2 || !ValidRegionCode(countryCode) {
		return Region{}, false
	}
	latitude, latErr := strconv.ParseFloat(latLong[0], 64)
	longitude, lonErr := strconv.ParseFloat(latLong[1], 64)
	if latErr != nil || lonErr != nil || math.IsNaN(latitude) || math.IsNaN(longitude) || latitude < -90 || latitude > 90 || longitude < -180 || longitude > 180 {
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

func countryRank(region Region) int {
	if region.CountryCode == "US" {
		return 0
	}
	return 1
}

package rovalra

import (
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/sleaze5/RobloxAccountManager/internal/appmeta"
	"github.com/sleaze5/RobloxAccountManager/internal/games"
)

const (
	maxResponseSize  = 1 << 20
	detailsBatchSize = 50
	maxTextLength    = 100
)

type Client struct {
	http *http.Client

	mu               sync.Mutex
	regions          []Region
	regionsFetchedAt time.Time
}

func New() *Client {
	return &Client{http: &http.Client{
		Timeout: 15 * time.Second,
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}}
}

type record struct {
	ServerID     string `json:"server_id"`
	PlaceVersion int64  `json:"place_version"`
	FirstSeen    string `json:"first_seen"`
	City         string `json:"city"`
	Region       string `json:"region"`
	Country      string `json:"country"`
	RegionCode   string `json:"region_code"`
	DatacenterID int64  `json:"datacenter_id"`
	IPAddress    string `json:"ip_address"`
}

func (client *Client) Records(ctx context.Context, placeID int64, jobIDs []string) (map[string]games.ServerRecord, error) {
	result := make(map[string]games.ServerRecord, len(jobIDs))
	for start := 0; start < len(jobIDs); start += detailsBatchSize {
		parameters := url.Values{
			"place_id":   {strconv.FormatInt(placeID, 10)},
			"server_ids": {strings.Join(jobIDs[start:min(start+detailsBatchSize, len(jobIDs))], ",")},
		}
		var payload struct {
			Servers []record `json:"servers"`
		}
		if err := client.get(ctx, "/v1/servers/details", parameters, &payload); err != nil {
			return result, err
		}
		for _, item := range payload.Servers {
			if converted, ok := item.convert(); ok {
				result[converted.JobID] = converted
			}
		}
	}
	return result, nil
}

func (client *Client) Servers(ctx context.Context, query games.RecordQuery) (games.RecordPage, error) {
	parameters := url.Values{
		"place_id": {strconv.FormatInt(query.PlaceID, 10)},
		"limit":    {strconv.Itoa(query.Limit)},
	}
	if query.Cursor > 0 {
		parameters.Set("cursor", strconv.FormatInt(query.Cursor, 10))
	}
	path := "/v1/servers/" + string(query.Order)
	if query.Region != "" {
		path = "/v1/servers/region"
		parameters.Set("region", query.Region)
	}
	var payload struct {
		Servers    []record `json:"servers"`
		NextCursor *int64   `json:"next_cursor"`
	}
	if err := client.get(ctx, path, parameters, &payload); err != nil {
		return games.RecordPage{}, err
	}
	if len(payload.Servers) > 100 {
		return games.RecordPage{}, errors.New("RoValra returned an oversized server page")
	}
	page := games.RecordPage{Servers: make([]games.ServerRecord, 0, len(payload.Servers))}
	for _, item := range payload.Servers {
		if converted, ok := item.convert(); ok {
			page.Servers = append(page.Servers, converted)
		}
	}
	if payload.NextCursor != nil && *payload.NextCursor > query.Cursor {
		page.NextCursor = *payload.NextCursor
	}
	return page, nil
}

func (client *Client) Stats(ctx context.Context, placeID int64) (games.ServerStats, error) {
	var payload struct {
		Counts struct {
			DetailedRegions map[string]struct {
				Cities map[string]int64 `json:"cities"`
			} `json:"detailed_regions"`
			Regions       map[string]int64 `json:"regions"`
			NewestVersion *int64           `json:"newest_place_version"`
			TotalServers  int64            `json:"total_servers"`
		} `json:"counts"`
	}
	if err := client.get(ctx, "/v1/servers/counts", url.Values{"place_id": {strconv.FormatInt(placeID, 10)}}, &payload); err != nil {
		return games.ServerStats{}, err
	}
	counts := payload.Counts
	stats := games.ServerStats{Regions: make([]games.ServerRegion, 0, len(counts.Regions)), TotalServers: max(counts.TotalServers, 0)}
	if counts.NewestVersion != nil {
		stats.NewestVersion = max(*counts.NewestVersion, 0)
	}
	for code, servers := range counts.Regions {
		if !ValidRegionCode(code) || servers <= 0 {
			continue
		}
		cities := counts.DetailedRegions[code].Cities
		region := games.ServerRegion{Code: code, CountryCode: strings.SplitN(code, "-", 2)[0], Cities: make([]string, 0, len(cities)), Servers: servers}
		for city := range cities {
			if city = text(city); city != "" {
				region.Cities = append(region.Cities, city)
			}
		}
		slices.SortFunc(region.Cities, func(a, b string) int { return cmp.Compare(cities[b], cities[a]) })
		stats.Regions = append(stats.Regions, region)
	}
	slices.SortFunc(stats.Regions, func(a, b games.ServerRegion) int {
		return cmp.Or(cmp.Compare(b.Servers, a.Servers), cmp.Compare(a.Code, b.Code))
	})
	return stats, nil
}

func (client *Client) get(ctx context.Context, path string, parameters url.Values, target any) error {
	code, body, err := client.fetch(ctx, path, parameters)
	if err != nil {
		return err
	}
	var status struct {
		Status  string `json:"status"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(body, &status); err != nil {
		return fmt.Errorf("%s: HTTP %d with invalid JSON: %w", path, code, err)
	}
	if code != http.StatusOK || status.Status != "success" {
		return fmt.Errorf("%s: HTTP %d: %s", path, code, text(status.Message))
	}
	if err := json.Unmarshal(body, target); err != nil {
		return fmt.Errorf("%s: invalid data: %w", path, err)
	}
	return nil
}

func (client *Client) fetch(ctx context.Context, path string, parameters url.Values) (int, []byte, error) {
	endpoint := &url.URL{Scheme: "https", Host: "apis.rovalra.com", Path: path, RawQuery: parameters.Encode()}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return 0, nil, err
	}
	request.Header.Set("Accept", "application/json")
	request.Header.Set("User-Agent", appmeta.UserAgent)
	response, err := client.http.Do(request)
	if err != nil {
		return 0, nil, fmt.Errorf("request %s: %w", path, err)
	}
	defer response.Body.Close()
	body, err := io.ReadAll(io.LimitReader(response.Body, maxResponseSize+1))
	if err != nil {
		return 0, nil, fmt.Errorf("read %s: %w", path, err)
	}
	if len(body) > maxResponseSize {
		return 0, nil, fmt.Errorf("%s: response is too large", path)
	}
	return response.StatusCode, body, nil
}

func (item record) convert() (games.ServerRecord, bool) {
	if !games.ValidJobID(item.ServerID) {
		return games.ServerRecord{}, false
	}
	converted := games.ServerRecord{
		JobID: item.ServerID, PlaceVersion: max(item.PlaceVersion, 0),
		City: text(item.City), Region: text(item.Region), Country: text(item.Country),
		DatacenterID: max(item.DatacenterID, 0),
	}
	if ValidRegionCode(item.RegionCode) {
		converted.CountryCode = item.RegionCode
	}
	if address := net.ParseIP(item.IPAddress); address != nil {
		converted.Address = address.String()
	}
	firstSeen := item.FirstSeen
	if firstSeen != "" && !strings.HasSuffix(firstSeen, "Z") {
		firstSeen += "Z"
	}
	if seen, err := time.Parse(time.RFC3339Nano, firstSeen); err == nil {
		converted.FirstSeenMs = seen.UnixMilli()
	}
	if converted.City == converted.Region {
		converted.Region = ""
	}
	return converted, true
}

func ValidRegionCode(code string) bool {
	if len(code) < 2 || len(code) > 64 {
		return false
	}
	for _, char := range code {
		if (char < 'A' || char > 'Z') && char != '-' && char != ' ' {
			return false
		}
	}
	return true
}

func text(value string) string {
	value = strings.TrimSpace(value)
	if value == "Unknown" || !utf8.ValidString(value) || utf8.RuneCountInString(value) > maxTextLength {
		return ""
	}
	return value
}

package rbxservers

import (
	"context"
	"errors"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/sleaze5/RobloxAccountManager/internal/appmeta"
	"github.com/sleaze5/RobloxAccountManager/internal/games"
)

const (
	maxResponseSize = 2 << 20
	maxServers      = 100
	maxNameLength   = 100
)

var (
	quickLaunchPattern = regexp.MustCompile(`(?is)<a\b[^>]*\bhref\s*=\s*["']/embedded/quicklaunch/([0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12})["'][^>]*>(.*?)</a>`)
	spanPattern        = regexp.MustCompile(`(?is)<span\b[^>]*>(.*?)</span>`)
	tagPattern         = regexp.MustCompile(`(?s)<[^>]*>`)
)

type Client struct {
	http *http.Client
}

func New() *Client {
	return &Client{http: &http.Client{
		Timeout: 15 * time.Second,
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}}
}

func (client *Client) Servers(ctx context.Context, placeID int64) (games.ListedPrivateServerPage, error) {
	endpoint := &url.URL{Scheme: "https", Host: "rbxservers.xyz", Path: "/embedded/game/" + strconv.FormatInt(placeID, 10)}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), http.NoBody)
	if err != nil {
		return games.ListedPrivateServerPage{}, fmt.Errorf("create RBXServers request: %w", err)
	}
	request.Header.Set("Accept", "text/html")
	request.Header.Set("User-Agent", appmeta.UserAgent)
	response, err := client.http.Do(request)
	if err != nil {
		return games.ListedPrivateServerPage{}, fmt.Errorf("request RBXServers servers: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return games.ListedPrivateServerPage{}, fmt.Errorf("RBXServers servers: HTTP %d", response.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, maxResponseSize+1))
	if err != nil {
		return games.ListedPrivateServerPage{}, fmt.Errorf("read RBXServers servers: %w", err)
	}
	if len(body) > maxResponseSize {
		return games.ListedPrivateServerPage{}, errors.New("RBXServers servers: response is too large")
	}
	return games.ListedPrivateServerPage{Servers: parse(string(body))}, nil
}

// The UUID is the RBXServers server ID, not a Roblox code. The quicklaunch page
// gets the Roblox link code after the browser solves the RBXServers challenge.
func parse(page string) []games.ListedPrivateServer {
	servers := []games.ListedPrivateServer{}
	seen := map[string]bool{}
	for _, match := range quickLaunchPattern.FindAllStringSubmatch(page, -1) {
		id := strings.ToLower(match[1])
		if seen[id] {
			continue
		}
		seen[id] = true
		servers = append(servers, games.ListedPrivateServer{ID: id, Name: serverName(match[2]), URL: "https://rbxservers.xyz/embedded/quicklaunch/" + id})
		if len(servers) == maxServers {
			break
		}
	}
	return servers
}

func serverName(anchor string) string {
	span := spanPattern.FindStringSubmatch(anchor)
	if span == nil {
		return ""
	}
	name := strings.Join(strings.Fields(html.UnescapeString(tagPattern.ReplaceAllString(span[1], ""))), " ")
	if !utf8.ValidString(name) || utf8.RuneCountInString(name) > maxNameLength {
		return ""
	}
	return name
}

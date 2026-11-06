package rbxservers

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
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
	maxResponseSize      = 2 << 20
	maxServers           = 100
	maxNameLength        = 100
	maxChallengeSolution = 1_000_000
)

var ErrServerUnavailable = errors.New("RBXServers server is unavailable")

var (
	quickLaunchPattern = regexp.MustCompile(`(?is)<a\b[^>]*\bhref\s*=\s*["']/embedded/quicklaunch/([0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12})["'][^>]*>(.*?)</a>`)
	spanPattern        = regexp.MustCompile(`(?is)<span\b[^>]*>(.*?)</span>`)
	tagPattern         = regexp.MustCompile(`(?s)<[^>]*>`)
	challengePattern   = regexp.MustCompile(`robot_challenge\\?"\s*:\s*\\?"([A-Za-z0-9+/]+={0,2})`)
	linkCodePattern    = regexp.MustCompile(`^\d{1,64}$`)
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
	code, body, err := client.fetch(ctx, http.MethodGet, &url.URL{Scheme: "https", Host: "rbxservers.xyz", Path: "/embedded/game/" + strconv.FormatInt(placeID, 10)}, nil)
	if err != nil {
		return games.ListedPrivateServerPage{}, fmt.Errorf("RBXServers servers: %w", err)
	}
	if code != http.StatusOK {
		return games.ListedPrivateServerPage{}, fmt.Errorf("RBXServers servers: HTTP %d", code)
	}
	return games.ListedPrivateServerPage{Servers: parse(string(body))}, nil
}

// LinkCode returns the Roblox link code of a shared server. RBXServers gives it
// out after the requester solves the proof-of-work challenge on the server page.
func (client *Client) LinkCode(ctx context.Context, serverID string) (string, error) {
	code, page, err := client.fetch(ctx, http.MethodGet, &url.URL{Scheme: "https", Host: "rbxservers.xyz", Path: "/embedded/quicklaunch/" + serverID}, nil)
	if err != nil {
		return "", fmt.Errorf("RBXServers server page: %w", err)
	}
	if code != http.StatusOK {
		return "", fmt.Errorf("RBXServers server page: HTTP %d", code)
	}
	match := challengePattern.FindSubmatch(page)
	if match == nil {
		return "", ErrServerUnavailable
	}
	decoded, err := base64.StdEncoding.DecodeString(string(match[1]))
	if err != nil {
		return "", fmt.Errorf("decode RBXServers challenge: %w", err)
	}
	var challenge struct {
		Hash string `json:"c"`
		Salt string `json:"s"`
		ID   string `json:"i"`
	}
	if err := json.Unmarshal(decoded, &challenge); err != nil {
		return "", fmt.Errorf("parse RBXServers challenge: %w", err)
	}
	solution, err := solve(ctx, challenge.Hash, challenge.Salt)
	if err != nil {
		return "", fmt.Errorf("solve RBXServers challenge: %w", err)
	}
	request, err := json.Marshal(struct {
		ChallengeID       string `json:"challenge_id"`
		ChallengeSolution int    `json:"challenge_solution"`
		Context           string `json:"ctx"`
	}{challenge.ID, solution, "EmbeddedPage"})
	if err != nil {
		return "", fmt.Errorf("encode RBXServers link request: %w", err)
	}
	code, body, err := client.fetch(ctx, http.MethodPost, &url.URL{Scheme: "https", Host: "api.rbxservers.xyz", Path: "/servers/v2/fetch-link/" + serverID}, request)
	if err != nil {
		return "", fmt.Errorf("RBXServers link: %w", err)
	}
	if code == http.StatusNotFound {
		return "", ErrServerUnavailable
	}
	var result struct {
		LinkCode string `json:"server_linkcode"`
		Error    string `json:"error"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("RBXServers link: HTTP %d with invalid JSON: %w", code, err)
	}
	if code != http.StatusOK || !linkCodePattern.MatchString(result.LinkCode) {
		return "", fmt.Errorf("RBXServers link: HTTP %d: %.100q", code, result.Error)
	}
	return result.LinkCode, nil
}

func solve(ctx context.Context, hash, salt string) (int, error) {
	want, err := hex.DecodeString(hash)
	if err != nil || len(want) != sha256.Size {
		return 0, errors.New("the challenge hash is invalid")
	}
	input := []byte(salt)
	for solution := range maxChallengeSolution + 1 {
		if solution%65536 == 0 && ctx.Err() != nil {
			return 0, ctx.Err()
		}
		sum := sha256.Sum256(strconv.AppendInt(input[:len(salt)], int64(solution), 10))
		if bytes.Equal(sum[:], want) {
			return solution, nil
		}
	}
	return 0, errors.New("the challenge has no solution")
}

func (client *Client) fetch(ctx context.Context, method string, endpoint *url.URL, body []byte) (int, []byte, error) {
	request, err := http.NewRequestWithContext(ctx, method, endpoint.String(), bytes.NewReader(body))
	if err != nil {
		return 0, nil, fmt.Errorf("create request: %w", err)
	}
	if body == nil {
		request.Header.Set("Accept", "text/html")
	} else {
		request.Header.Set("Accept", "application/json")
		request.Header.Set("Content-Type", "application/json")
	}
	request.Header.Set("User-Agent", appmeta.UserAgent)
	response, err := client.http.Do(request)
	if err != nil {
		return 0, nil, fmt.Errorf("request: %w", err)
	}
	defer response.Body.Close()
	data, err := io.ReadAll(io.LimitReader(response.Body, maxResponseSize+1))
	if err != nil {
		return 0, nil, fmt.Errorf("read response: %w", err)
	}
	if len(data) > maxResponseSize {
		return 0, nil, errors.New("response is too large")
	}
	return response.StatusCode, data, nil
}

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

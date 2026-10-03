package gamelaunch

import (
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"
)

type Method string

const (
	MethodPlace Method = "place"
	MethodLink  Method = "link"
	MethodUser  Method = "user"
)

type Input struct {
	Method             Method `json:"method"`
	PlaceID            string `json:"placeId"`
	JobID              string `json:"jobId"`
	Link               string `json:"link"`
	User               string `json:"user"`
	ResolveCurrentGame bool   `json:"resolveCurrentGame"`
	Teleport           bool   `json:"teleport"`
	DirectLaunch       bool   `json:"directLaunch"`
	LaunchData         string `json:"launchData"`
}

var (
	jobIDPattern     = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)
	usernamePattern  = regexp.MustCompile(`^[a-zA-Z0-9_]{1,20}$`)
	codePattern      = regexp.MustCompile(`^[a-zA-Z0-9_-]{1,512}$`)
	shareCodePattern = regexp.MustCompile(`^[0-9a-fA-F]{32}$`)
)

func Parse(input Input) (Request, error) {
	switch input.Method {
	case MethodPlace:
		return placeRequest(input.PlaceID, input.JobID)
	case MethodLink:
		return parseLink(input.Link)
	case MethodUser:
		return parseUser(input.User)
	default:
		return Request{}, fmt.Errorf("choose a launch method")
	}
}

func positiveID(value, label string) (int64, error) {
	value = strings.TrimSpace(value)
	if value == "" || strings.IndexFunc(value, func(c rune) bool { return c < '0' || c > '9' }) >= 0 {
		return 0, fmt.Errorf("enter a positive numeric %s", label)
	}
	id, err := strconv.ParseInt(value, 10, 64)
	if err != nil || id <= 0 {
		return 0, fmt.Errorf("enter a valid positive %s", label)
	}
	return id, nil
}

func placeRequest(place, job string) (Request, error) {
	id, err := positiveID(place, "Place ID")
	if err != nil {
		return Request{}, err
	}
	job = strings.TrimSpace(job)
	if job != "" && !jobIDPattern.MatchString(job) {
		return Request{}, fmt.Errorf("enter a valid Job ID (UUID), or leave it empty for a public server")
	}
	return Request{PlaceID: id, JobID: job}, nil
}

func parseUser(value string) (Request, error) {
	value = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(value), "@"))
	if len(value) >= 3 && strings.EqualFold(value[:3], "id=") {
		id, err := positiveID(value[3:], "User ID")
		return Request{UserID: id}, err
	}
	if !usernamePattern.MatchString(value) {
		return Request{}, fmt.Errorf("enter a username, or use id=123 for a User ID")
	}
	return Request{Username: value}, nil
}

func parseLink(value string) (Request, error) {
	value = strings.TrimSpace(value)
	if len(value) > 8192 {
		return Request{}, fmt.Errorf("the Roblox link is too long")
	}
	if shareCodePattern.MatchString(value) {
		return Request{ShareCode: value, ShareType: ShareServer}, nil
	}
	if strings.HasPrefix(strings.ToLower(value), "roblox://") {
		return parseDeepLink(value[len("roblox://"):])
	}
	if !strings.Contains(value, "://") {
		value = "https://" + value
	}
	link, err := url.Parse(value)
	if err != nil || !allowedWebLink(link) {
		return Request{}, fmt.Errorf("enter a Roblox game, private-server, or experience-invite link, or a 32-character server-share code")
	}
	query, err := launchQuery(link.RawQuery)
	if err != nil {
		return Request{}, err
	}
	switch strings.TrimSuffix(link.Path, "/") {
	case "/share", "/share-links":
		return shareRequest(query)
	case "/games/start":
		return queryRequest(query.Get("placeId"), query)
	}
	parts := strings.Split(strings.Trim(link.Path, "/"), "/")
	if len(parts) < 2 || parts[0] != "games" {
		return Request{}, fmt.Errorf("this Roblox link is not a supported game or server link")
	}
	if query.Has("placeId") && query.Get("placeId") != parts[1] {
		return Request{}, fmt.Errorf("the link contains conflicting Place IDs")
	}
	return queryRequest(parts[1], query)
}

func allowedWebLink(link *url.URL) bool {
	if link == nil || link.User != nil || link.Port() != "" || link.Fragment != "" {
		return false
	}
	if link.Scheme != "https" && link.Scheme != "http" {
		return false
	}
	switch strings.ToLower(link.Hostname()) {
	case "roblox.com", "www.roblox.com", "web.roblox.com":
		return true
	default:
		return false
	}
}

func parseDeepLink(body string) (Request, error) {
	if strings.Contains(body, "#") {
		return Request{}, fmt.Errorf("the Roblox deep link must not contain a fragment")
	}
	path, rawQuery, hasQuery := strings.Cut(body, "?")
	if !hasQuery {
		path, rawQuery = "", body
	}
	query, err := launchQuery(rawQuery)
	if err != nil {
		return Request{}, err
	}
	switch strings.TrimSuffix(path, "/") {
	case "navigation/share_links":
		return shareRequest(query)
	case "", "experiences/start":
		return queryRequest(query.Get("placeId"), query)
	default:
		return Request{}, fmt.Errorf("this Roblox deep link is not a supported game or server link")
	}
}

func launchQuery(raw string) (url.Values, error) {
	query, err := url.ParseQuery(raw)
	if err != nil {
		return nil, fmt.Errorf("the Roblox link has invalid URL parameters")
	}
	for _, key := range []string{"placeId", "gameInstanceId", "privateServerLinkCode", "linkCode", "accessCode", "code", "type"} {
		if values, exists := query[key]; exists && (len(values) != 1 || strings.TrimSpace(values[0]) == "") {
			return nil, fmt.Errorf("the Roblox link has empty or repeated launch parameters")
		}
	}
	if len(query["launchData"]) > 1 {
		return nil, fmt.Errorf("the Roblox link has repeated launch data")
	}
	if query.Has("userId") {
		return nil, fmt.Errorf("use the User method to join a user")
	}
	return query, nil
}

func shareRequest(query url.Values) (Request, error) {
	var shareType ShareType
	switch {
	case strings.EqualFold(query.Get("type"), string(ShareServer)):
		shareType = ShareServer
	case strings.EqualFold(query.Get("type"), string(ShareExperienceInvite)):
		shareType = ShareExperienceInvite
	default:
		return Request{}, fmt.Errorf("use a share link with type=Server or type=ExperienceInvite")
	}
	if !codePattern.MatchString(query.Get("code")) {
		return Request{}, fmt.Errorf("the share link contains an invalid code")
	}
	for _, key := range []string{"placeId", "gameInstanceId", "privateServerLinkCode", "linkCode", "accessCode"} {
		if query.Has(key) {
			return Request{}, fmt.Errorf("the share link contains conflicting launch parameters")
		}
	}
	return Request{ShareCode: query.Get("code"), ShareType: shareType, LaunchData: query.Get("launchData")}, nil
}

func queryRequest(place string, query url.Values) (Request, error) {
	if query.Has("code") || query.Has("type") {
		return Request{}, fmt.Errorf("use a full Roblox share link for a share code")
	}
	request, err := placeRequest(place, query.Get("gameInstanceId"))
	if err != nil {
		return Request{}, err
	}
	request.LinkCode = query.Get("privateServerLinkCode")
	if code := query.Get("linkCode"); code != "" {
		if request.LinkCode != "" && request.LinkCode != code {
			return Request{}, fmt.Errorf("the link contains conflicting private-server codes")
		}
		request.LinkCode = code
	}
	request.AccessCode = query.Get("accessCode")
	request.LaunchData = query.Get("launchData")
	if err := validatePrivateCodes(request); err != nil {
		return Request{}, err
	}
	return request, nil
}

func validatePrivateCodes(request Request) error {
	for _, code := range []string{request.LinkCode, request.AccessCode} {
		if code != "" && (!codePattern.MatchString(code) || strings.EqualFold(code, "null") || strings.EqualFold(code, "undefined")) {
			return fmt.Errorf("the private-server link contains an invalid code")
		}
	}
	if request.JobID != "" && (request.LinkCode != "" || request.AccessCode != "") {
		return fmt.Errorf("a link cannot target both a Job ID and a private server")
	}
	return nil
}

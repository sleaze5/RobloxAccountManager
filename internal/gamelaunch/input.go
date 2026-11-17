package gamelaunch

import (
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"slices"
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
	PlayerPath         string `json:"playerPath"`
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
		return Request{}, errors.New("select a join method")
	}
}

func positiveID(text, label string) (int64, error) {
	text = strings.TrimSpace(text)
	if text == "" || strings.IndexFunc(text, func(character rune) bool { return character < '0' || character > '9' }) >= 0 {
		return 0, fmt.Errorf("enter a %s that uses only digits", label)
	}
	id, err := strconv.ParseInt(text, 10, 64)
	if err != nil || id <= 0 {
		return 0, fmt.Errorf("enter a valid %s", label)
	}
	return id, nil
}

func placeRequest(place, job string) (Request, error) {
	placeID, err := positiveID(place, "place ID")
	if err != nil {
		return Request{}, err
	}
	job = strings.TrimSpace(job)
	if job != "" && !jobIDPattern.MatchString(job) {
		return Request{}, errors.New("enter a valid job ID, or leave it empty to join any server")
	}
	return Request{PlaceID: placeID, JobID: job}, nil
}

func parseUser(user string) (Request, error) {
	user = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(user), "@"))
	if len(user) >= 3 && (strings.EqualFold(user[:3], "id:") || strings.EqualFold(user[:3], "id=")) {
		userID, err := positiveID(user[3:], "user ID")
		return Request{UserID: userID}, err
	}
	if !usernamePattern.MatchString(user) {
		return Request{}, errors.New("enter a username, or id: and a user ID")
	}
	return Request{Username: user}, nil
}

func parseLink(linkText string) (Request, error) {
	linkText = strings.TrimSpace(linkText)
	if len(linkText) > 8192 {
		return Request{}, errors.New("the link is too long")
	}
	if shareCodePattern.MatchString(linkText) {
		return Request{ShareCode: linkText, ShareType: ShareServer}, nil
	}
	if strings.HasPrefix(strings.ToLower(linkText), "roblox://") {
		return parseDeepLink(linkText[len("roblox://"):])
	}
	if !strings.Contains(linkText, "://") {
		linkText = "https://" + linkText
	}
	link, err := url.Parse(linkText)
	if err != nil || !allowedWebLink(link) {
		return Request{}, errors.New("enter a game link, private server link, game invite, or server share code")
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
		return Request{}, errors.New("this link is not a supported game or server link")
	}
	if query.Has("placeId") && query.Get("placeId") != parts[1] {
		return Request{}, errors.New("the link has 2 different place IDs")
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
		return Request{}, errors.New("this link is not a supported game or server link")
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
		return Request{}, errors.New("this link is not a supported game or server link")
	}
}

func launchQuery(raw string) (url.Values, error) {
	query, err := url.ParseQuery(raw)
	if err != nil {
		return nil, errors.New("the link is invalid")
	}
	for _, key := range []string{"placeId", "gameInstanceId", "privateServerLinkCode", "linkCode", "accessCode", "code", "type"} {
		if values, exists := query[key]; exists && (len(values) != 1 || strings.TrimSpace(values[0]) == "") {
			return nil, errors.New("the link is invalid")
		}
	}
	if len(query["launchData"]) > 1 {
		return nil, errors.New("the link is invalid")
	}
	if query.Has("userId") {
		return nil, errors.New("use User to join a user")
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
		return Request{}, errors.New("use a server share link or a game invite")
	}
	if !codePattern.MatchString(query.Get("code")) {
		return Request{}, errors.New("the share link has an invalid code")
	}
	if slices.ContainsFunc([]string{"placeId", "gameInstanceId", "privateServerLinkCode", "linkCode", "accessCode"}, query.Has) {
		return Request{}, errors.New("the share link is invalid")
	}
	return Request{ShareCode: query.Get("code"), ShareType: shareType, LaunchData: query.Get("launchData")}, nil
}

func queryRequest(place string, query url.Values) (Request, error) {
	if query.Has("code") || query.Has("type") {
		return Request{}, errors.New("paste the full share link")
	}
	request, err := placeRequest(place, query.Get("gameInstanceId"))
	if err != nil {
		return Request{}, err
	}
	request.LinkCode = query.Get("privateServerLinkCode")
	if code := query.Get("linkCode"); code != "" {
		if request.LinkCode != "" && request.LinkCode != code {
			return Request{}, errors.New("the link has 2 different private server codes")
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
			return errors.New("the private server link has an invalid code")
		}
	}
	if request.JobID != "" && (request.LinkCode != "" || request.AccessCode != "") {
		return errors.New("a link cannot have both a job ID and a private server")
	}
	return nil
}

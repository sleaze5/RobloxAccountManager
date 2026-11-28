package services

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/sleaze5/RobloxAccountManager/internal/gamelaunch"
	"github.com/sleaze5/RobloxAccountManager/internal/games"
	"github.com/sleaze5/RobloxAccountManager/internal/roblox"
)

const gameJoinCheckEndpoint = "game-join-check"

// The status codes of Roblox's join service, as RoSeal lists them in
// JoinServerStatusCode. Roblox has not used 7, 8, and 9.
const (
	joinStatusRetry                         = 0
	joinStatusServerFound                   = 1
	joinStatusServerDataLoaded              = 2
	joinStatusGameDisabled                  = 3
	joinStatusServerUnavailable             = 4
	joinStatusServerUnavailableUnexpectedly = 5
	joinStatusServerFull                    = 6
	joinStatusFollowedUserLeft              = 10
	joinStatusGameRestricted                = 11
	joinStatusNoPermission                  = 12
	joinStatusServerBusy                    = 13
	joinStatusPartyTooLarge                 = 16
	joinStatusChannelMismatch               = 19
	joinStatusSetChannelInternalOnly        = 20
	joinStatusUnauthorizedPrivacySettings   = 21
	joinStatusInQueue                       = 22
	joinStatusUserBanned                    = 23
)

// joinRefusal is the reason that the message of a join response names, as
// RoSeal's parseJoinServerStatusMessage reads it.
type joinRefusal int

const (
	joinRefusalUnknown joinRefusal = iota
	joinRefusalBlockedByParent
	joinRefusalCreatorBan
	joinRefusalPrivateServer
	joinRefusalReservedServer
	joinRefusalDeviceNotSupported
	joinRefusalRegionCompliance
	joinRefusalGamePrivate
	joinRefusalUnverifiedSeventeenPlus
	joinRefusalContentMaturity
	joinRefusalInvalidPlace
	joinRefusalInvalidServer
	joinRefusalInvalidAccessCode
	joinRefusalUnderReview
	joinRefusalFriendsOnly
	joinRefusalGroupOnly
	joinRefusalCannotFollowUser
	joinRefusalUserNotFound
	joinRefusalRequestDenied
	joinRefusalNonRootPlace
	joinRefusalPurchaseRequired
	joinRefusalNotPublished
	joinRefusalPrivacy
	joinRefusalUnrated
	joinRefusalSocialHangout
	joinRefusalInternalError
)

// joinRefusalByMessage holds the messages of the join service that RoSeal
// recognizes, copied exactly. Roblox writes the bracketed ones for clients to
// translate, so they never reach the user as they are.
var joinRefusalByMessage = map[string]joinRefusal{
	"<This text is not reviewed by UX because it is not displayed to users. Each client provides their own translation> This experience was blocked by your parent":                                                   joinRefusalBlockedByParent,
	"<This text is not reviewed by UX because it is not displayed to users. The client renders a message chosen by the creator when they banned the user.> You have been banned from this experience by its creators": joinRefusalCreatorBan,
	"User lacks access to join private server":                            joinRefusalPrivateServer,
	"Can't join private instance through specific joins":                  joinRefusalPrivateServer,
	"User lacks permissions to join private server":                       joinRefusalReservedServer,
	"Your device is not supported to play this game.":                     joinRefusalDeviceNotSupported,
	"This game is unavailable in your region to comply with regulations.": joinRefusalRegionCompliance,
	"Game's root place is not active.":                                    joinRefusalGamePrivate,
	"<This text is not reviewed by UX because it is not displayed to users. Each client provides their own translation> This experience is rated 17+ and user is 17 years or older but it is not verified.": joinRefusalUnverifiedSeventeenPlus,
	"This game is unavailable due to your content controls setting.": joinRefusalContentMaturity,
	"To join this experience, update your content maturity setting.": joinRefusalContentMaturity,
	"Cannot join place without universe":                             joinRefusalInvalidPlace,
	"Place does not have universe":                                   joinRefusalInvalidPlace,
	"Cannot join game without placeId.":                              joinRefusalInvalidPlace,
	"Invalid place id for game join.":                                joinRefusalInvalidPlace,
	"Game instance cannot be joined":                                 joinRefusalInvalidServer,
	"Could not find requested game instance":                         joinRefusalInvalidServer,
	"Access code for private server is not valid.":                   joinRefusalInvalidAccessCode,
	"Access code for reserved server is not valid.":                  joinRefusalInvalidAccessCode,
	// RoSeal does not list this one. Verified against Roblox on 2026-11-23.
	"Link code for private server is not valid.":                           joinRefusalInvalidAccessCode,
	"This game has not been approved, yet.":                                joinRefusalUnderReview,
	"The game is currently unavailable due to restrictions set by Roblox.": joinRefusalUnderReview,
	"You are not allowed to play this game. (Friends Only)":                joinRefusalFriendsOnly,
	"You are not allowed to play this game. (Group Only)":                  joinRefusalGroupOnly,
	"Unauthorized to follow user.":                                         joinRefusalCannotFollowUser,
	"Unable to follow user.":                                               joinRefusalUserNotFound,
	"Cannot join non-root place or root of different universe":             joinRefusalNonRootPlace,
	"Cannot join this non-root place due to join restrictions":             joinRefusalNonRootPlace,
	"You need to purchase access to this game before you can play.":        joinRefusalPurchaseRequired,
	"This game is not published.":                                          joinRefusalNotPublished,
	"User privacy does not allow join.":                                    joinRefusalPrivacy,
	"This experience is not accessible because it is unrated":              joinRefusalUnrated,
	"This experience is unavailable to you as it is unrated.":              joinRefusalUnrated,
	"<This text is not reviewed by UX because it is not displayed to users. Each client provides their own translation> This experience is unavailable to you as it is unrated.":                                                    joinRefusalUnrated,
	"<This text is not reviewed by UX because it is not displayed to users. Each client provides their own translation> This experience is unavailable to you as it is a Social Hangout and/or includes Freeform In-Game Creation.": joinRefusalSocialHangout,
	"Invalid Game.": joinRefusalInternalError,
	"One of the services required to determine game access is currently unavailable": joinRefusalInternalError,
	"Backend Exception": joinRefusalInternalError,
}

// parseJoinRefusal reads the reason in a join response message. Roblox answers
// a request that it denies without a reason with "Unable to join Game", a
// digit for the request type, and 11 or 12.
func parseJoinRefusal(message string) joinRefusal {
	if refusal, found := joinRefusalByMessage[message]; found {
		return refusal
	}
	if code, found := strings.CutPrefix(message, "Unable to join Game "); found && len(code) == 3 && code[0] >= '2' && code[0] <= '8' && (code[1:] == "11" || code[1:] == "12") {
		return joinRefusalRequestDenied
	}
	return joinRefusalUnknown
}

type creatorGameBan struct {
	StartTime string `json:"startTime"`
	// DurationSeconds is null for a ban without an end.
	DurationSeconds *int64 `json:"durationSeconds"`
}

// joinResponse holds the parts of a join response that tell how the join
// would end. The join script also holds tickets and tokens, which stay
// undecoded.
type joinResponse struct {
	Status        int     `json:"status"`
	Message       *string `json:"message"`
	QueuePosition int     `json:"queuePosition"`
	StatusData    *struct {
		// Roblox calls a ban from a game creatorExperienceBan.
		CreatorGameBan *creatorGameBan `json:"creatorExperienceBan"`
	} `json:"statusData"`
	JoinScript *struct {
		// Roblox calls the server ID GameId in the join script.
		ServerID string `json:"GameId"`
		// SessionID is a JSON object in a string, whose JoinType tells how
		// Roblox chose the server.
		SessionID string `json:"SessionId"`
	} `json:"joinScript"`
}

// CheckJoin asks Roblox's join service how a join of the target would end for
// the account, as RoSeal does before it launches. It starts no Roblox client.
// The target joins a user, a private server, a server, or a place, in that
// order of precedence.
func (service *Games) CheckJoin(ctx context.Context, accountID, version int64, target gamelaunch.Request) (games.JoinCheck, error) {
	body, status, err := service.requestJoin(ctx, accountID, version, target)
	if err != nil {
		return games.JoinCheck{}, err
	}
	var payload joinResponse
	if err := json.Unmarshal(body, &payload); err != nil {
		return games.JoinCheck{}, &roblox.Error{Kind: roblox.KindProtocol, Endpoint: gameJoinCheckEndpoint, Status: status, Message: "Roblox returned invalid join details.", Cause: err}
	}
	return classifyJoin(payload, target, time.Now()), nil
}

// requestJoin sends the join request of the target and returns the response
// body, which holds the tickets and tokens of the join script. Decode only the
// fields that are needed, and never log or return the body.
func (service *Games) requestJoin(ctx context.Context, accountID, version int64, target gamelaunch.Request) ([]byte, int, error) {
	path, body := joinCheckRequest(target)
	encodedBody, err := json.Marshal(body)
	if err != nil {
		return nil, 0, fmt.Errorf("encode join check request: %w", err)
	}
	response, err := service.client.Do(ctx, roblox.Request{
		Endpoint: gameJoinCheckEndpoint, AccountID: accountID, ExpectedSecretVersion: version, Authenticated: true,
		Method: http.MethodPost, URL: &url.URL{Scheme: "https", Host: "gamejoin.roblox.com", Path: path},
		Body: encodedBody, ContentType: "application/json", RequiresCSRF: true,
		// A join response holds the whole join script.
		MaxResponseSize: 512 << 10,
		Retry:           roblox.RetryPolicy{MaxAttempts: 3, Idempotent: true, RetryServerErrors: true, RetryRateLimit: true},
	})
	if err != nil {
		return nil, 0, err
	}
	return response.Body, response.Status, nil
}

func joinCheckRequest(target gamelaunch.Request) (string, map[string]any) {
	body := map[string]any{"gameJoinAttemptId": randomUUID()}
	switch {
	case target.UserID > 0:
		body["userIdToFollow"], body["joinOrigin"] = target.UserID, "JoinUser"
		return "/v1/play-with-user", body
	case target.AccessCode != "" || target.LinkCode != "":
		body["placeId"] = target.PlaceID
		if target.AccessCode != "" {
			body["accessCode"] = target.AccessCode
		} else {
			body["linkCode"] = target.LinkCode
		}
		return "/v1/join-private-game", body
	case target.ServerID != "":
		// Roblox calls the server ID gameId in this request.
		body["placeId"], body["gameId"] = target.PlaceID, target.ServerID
		return "/v1/join-game-instance", body
	default:
		body["placeId"] = target.PlaceID
		return "/v1/join-game", body
	}
}

// classifyJoin reads a join response as RoSeal's determineCanJoinUser does. A
// response with HTTP status 200 can still refuse the join, so only its body
// decides.
func classifyJoin(payload joinResponse, target gamelaunch.Request, now time.Time) games.JoinCheck {
	refusal := joinRefusalUnknown
	if payload.Message != nil {
		refusal = parseJoinRefusal(*payload.Message)
	}
	var ban *creatorGameBan
	if payload.StatusData != nil {
		ban = payload.StatusData.CreatorGameBan
	}
	switch {
	case payload.Status == joinStatusUserBanned || ban != nil || refusal == joinRefusalCreatorBan:
		return restricted(creatorBanMessage(ban, now))
	case payload.JoinScript != nil:
		return classifyJoinScript(payload.JoinScript.ServerID, joinType(payload.JoinScript.SessionID), target)
	case refusal == joinRefusalRequestDenied || refusal == joinRefusalInternalError:
		// Roblox gives no reason, so the join can still work.
		return games.JoinCheck{Outcome: games.JoinOutcomeUnknown}
	default:
		return classifyJoinStatus(payload, refusal, target.UserID > 0)
	}
}

// waitingJoinStatuses are the statuses of a join that can work a moment later,
// such as a full server or a server that Roblox is still starting.
var waitingJoinStatuses = []int{
	joinStatusRetry, joinStatusServerFound, joinStatusServerDataLoaded, joinStatusServerBusy,
	joinStatusServerFull, joinStatusServerUnavailableUnexpectedly, joinStatusInQueue,
}

// restrictedMessageByJoinStatus describes the statuses of a join that fails
// whatever the message says.
var restrictedMessageByJoinStatus = map[int]string{
	joinStatusServerUnavailable:      "This server is no longer available.",
	joinStatusFollowedUserLeft:       "This user is not in a game.",
	joinStatusChannelMismatch:        "This server runs a Roblox test build.",
	joinStatusSetChannelInternalOnly: "This server runs a Roblox test build.",
	joinStatusPartyTooLarge:          "The party is too large for this server.",
}

// refusingJoinStatusMessages describes the statuses that refuse access, for a
// message that names no known reason.
var refusingJoinStatusMessages = map[int]string{
	joinStatusGameDisabled:                "This game is unavailable.",
	joinStatusGameRestricted:              "Roblox does not let this account join this game.",
	joinStatusNoPermission:                "This account does not have permission to join.",
	joinStatusUnauthorizedPrivacySettings: "Privacy settings do not allow this join.",
}

func classifyJoinStatus(payload joinResponse, refusal joinRefusal, isFollowingUser bool) games.JoinCheck {
	if payload.QueuePosition > 0 {
		return games.JoinCheck{Outcome: games.JoinOutcomeWaiting}
	}
	// A known reason decides over the status, because Roblox answers a server
	// that no longer exists with the status of an unexpected failure.
	if message, found := refusalMessages[refusal]; found {
		return restricted(refusalMessage(refusal, isFollowingUser, message))
	}
	if slices.Contains(waitingJoinStatuses, payload.Status) {
		return games.JoinCheck{Outcome: games.JoinOutcomeWaiting}
	}
	if message, found := restrictedMessageByJoinStatus[payload.Status]; found {
		return restricted(message)
	}
	if statusMessage, found := refusingJoinStatusMessages[payload.Status]; found {
		return restricted(refusalMessage(refusal, isFollowingUser, statusMessage))
	}
	return games.JoinCheck{Outcome: games.JoinOutcomeUnknown}
}

// classifyJoinScript reads a join that Roblox accepted. Roblox can accept a
// join into another server than the requested one, such as when the followed
// user's server is restricted.
func classifyJoinScript(joinedServerID, sessionJoinType string, target gamelaunch.Request) games.JoinCheck {
	isFollowingUser := target.UserID > 0
	switch {
	case isFollowingUser && sessionJoinType == "MatchMade":
		return restricted("This user is in a restricted server. Roblox would send this account to a different server.")
	case target.ServerID != "" && joinedServerID != "" && !strings.EqualFold(joinedServerID, target.ServerID):
		return restricted("Roblox would send this account to a different server.")
	default:
		return games.JoinCheck{Outcome: games.JoinOutcomeJoinable}
	}
}

// joinType reads how Roblox chose the server, such as MatchMade or
// Specific_PrivateGame, or returns an empty string.
func joinType(sessionID string) string {
	var session struct {
		JoinType string `json:"JoinType"`
	}
	if sessionID == "" || json.Unmarshal([]byte(sessionID), &session) != nil {
		return ""
	}
	return session.JoinType
}

// followedUserRefusalMessages describes the reasons that read differently
// when the account follows a user into their server.
var followedUserRefusalMessages = map[joinRefusal]string{
	joinRefusalPrivateServer:  "This user is in a private server that this account cannot join.",
	joinRefusalReservedServer: "This user is in a reserved server, such as a match.",
}

var refusalMessages = map[joinRefusal]string{
	joinRefusalBlockedByParent:         "Parental controls block this game for this account.",
	joinRefusalPrivateServer:           "This account cannot join this private server.",
	joinRefusalReservedServer:          "This account cannot join this reserved server, such as a match.",
	joinRefusalDeviceNotSupported:      "This game does not support computers.",
	joinRefusalRegionCompliance:        "This game is unavailable in this account's region.",
	joinRefusalGamePrivate:             "This game is private.",
	joinRefusalUnverifiedSeventeenPlus: "This game is 17+, and this account has not verified its age.",
	joinRefusalContentMaturity:         "This account's content maturity setting does not allow this game.",
	joinRefusalUnrated:                 "This game is unrated.",
	joinRefusalSocialHangout:           "This account's settings do not allow social hangout games.",
	joinRefusalInvalidPlace:            "This place does not exist, or it is not part of a game.",
	joinRefusalInvalidServer:           "This server is no longer available.",
	joinRefusalInvalidAccessCode:       "The private server link is invalid or expired.",
	joinRefusalUnderReview:             "Roblox is reviewing this game.",
	joinRefusalFriendsOnly:             "Only friends of the creator can join this game.",
	joinRefusalGroupOnly:               "Only members of the creator's group can join this game.",
	joinRefusalCannotFollowUser:        "This user's privacy settings do not let this account join them.",
	joinRefusalUserNotFound:            "Roblox could not find this user.",
	joinRefusalNonRootPlace:            "This place can only be reached from inside its game.",
	joinRefusalPurchaseRequired:        "This account must buy access to this game first.",
	joinRefusalNotPublished:            "This game is not published.",
	joinRefusalPrivacy:                 "Privacy settings do not allow this join.",
}

// refusalMessage describes the reason, or uses statusMessage for a reason
// that the app does not know.
func refusalMessage(refusal joinRefusal, isFollowingUser bool, statusMessage string) string {
	if message, found := followedUserRefusalMessages[refusal]; found && isFollowingUser {
		return message
	}
	if message, found := refusalMessages[refusal]; found {
		return message
	}
	return statusMessage
}

// creatorBanMessage describes a ban from the game's creator. A ban in a game's
// own scripts is not visible before the client connects.
func creatorBanMessage(ban *creatorGameBan, now time.Time) string {
	if ban == nil {
		return "The game's creator banned this account."
	}
	if ban.DurationSeconds == nil {
		return "The game's creator banned this account permanently."
	}
	start, err := time.Parse(time.RFC3339, ban.StartTime)
	if err != nil {
		return "The game's creator banned this account."
	}
	remaining := start.Add(time.Duration(*ban.DurationSeconds) * time.Second).Sub(now)
	if remaining <= 0 {
		return "The game's creator banned this account."
	}
	return "The game's creator banned this account. The ban ends in about " + approximateDuration(remaining) + "."
}

func approximateDuration(duration time.Duration) string {
	switch {
	case duration < time.Hour:
		return countedUnit(max(int64(duration/time.Minute), 1), "minute")
	case duration < 48*time.Hour:
		return countedUnit(int64(duration/time.Hour), "hour")
	default:
		return countedUnit(int64(duration/(24*time.Hour)), "day")
	}
}

func countedUnit(count int64, unit string) string {
	if count == 1 {
		return "1 " + unit
	}
	return strconv.FormatInt(count, 10) + " " + unit + "s"
}

func restricted(message string) games.JoinCheck {
	return games.JoinCheck{Outcome: games.JoinOutcomeRestricted, Message: message}
}

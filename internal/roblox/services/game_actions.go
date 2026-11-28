package services

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strconv"

	"github.com/sleaze5/RobloxAccountManager/internal/roblox"
)

// The game action shapes below come from the games and develop API
// definitions that Roblox published, checked on 2026-11-27.
const (
	// MaxGameNameLength is the longest game name that Roblox saves.
	MaxGameNameLength = 50
	// MaxGameDescriptionLength is the longest game description that Roblox
	// saves.
	MaxGameDescriptionLength = 1000
)

// GameVote is the vote of an account on a game. Roblox stores a like as an
// up vote, a dislike as a down vote, and no vote as null.
type GameVote string

const (
	GameVoteNone    GameVote = "none"
	GameVoteLike    GameVote = "like"
	GameVoteDislike GameVote = "dislike"
)

func ValidGameVote(vote GameVote) bool {
	return vote == GameVoteNone || vote == GameVoteLike || vote == GameVoteDislike
}

type GameVoteStatus struct {
	Vote    GameVote `json:"vote"`
	CanVote bool     `json:"canVote"`
	// VoteRefusal tells why the account cannot vote. It is empty when the
	// account can vote.
	VoteRefusal string `json:"voteRefusal"`
}

// GameSettings is what the app lets a creator change about a game.
type GameSettings struct {
	Name                     string `json:"name"`
	IsPublic                 bool   `json:"isPublic"`
	ArePrivateServersAllowed bool   `json:"arePrivateServersAllowed"`
}

func gameAddress(universeID int64, suffix string) string {
	return "https://games.roblox.com/v1/games/" + strconv.FormatInt(universeID, 10) + suffix
}

func universeAddress(version string, universeID int64, suffix string) string {
	return "https://develop.roblox.com/" + version + "/universes/" + strconv.FormatInt(universeID, 10) + suffix
}

func invalidGameAction(endpoint string) error {
	return &roblox.Error{Kind: roblox.KindProtocol, Endpoint: endpoint, Message: "Roblox returned invalid game data."}
}

// SetGameFavorite adds the game to the favorites of the account, or removes
// it. Roblox answers only for the account of the session.
func (service *Users) SetGameFavorite(ctx context.Context, accountID, version, universeID int64, isFavorite bool) error {
	// Roblox calls a favorite game a favorited game.
	_, err := service.writeAccount(ctx, accountID, version, "game-favorite-update", http.MethodPost, gameAddress(universeID, "/favorites"), map[string]bool{"isFavorited": isFavorite})
	return err
}

// GameVoteStatus reads the vote of the account on the game, and whether it
// may vote.
func (service *Users) GameVoteStatus(ctx context.Context, accountID, version, universeID int64) (GameVoteStatus, error) {
	const endpoint = "game-vote"
	var payload struct {
		CanVote *bool `json:"canVote"`
		// Roblox calls a like true, a dislike false, and no vote null.
		UserVote *bool `json:"userVote"`
		// Roblox names the reason with a code, such as PlayGame.
		ReasonForNotVoteable string `json:"reasonForNotVoteable"`
	}
	if err := service.readAccountDetails(ctx, accountID, version, endpoint, gameAddress(universeID, "/votes/user"), &payload); err != nil {
		return GameVoteStatus{}, err
	}
	if payload.CanVote == nil {
		return GameVoteStatus{}, invalidGameAction(endpoint)
	}
	status := GameVoteStatus{Vote: GameVoteNone, CanVote: *payload.CanVote}
	if payload.UserVote != nil {
		status.Vote = GameVoteDislike
		if *payload.UserVote {
			status.Vote = GameVoteLike
		}
	}
	if !status.CanVote {
		status.VoteRefusal = voteRefusalsByReason[payload.ReasonForNotVoteable]
		if status.VoteRefusal == "" {
			status.VoteRefusal = "Roblox does not let this account vote on this game."
		}
	}
	return status, nil
}

// voteRefusalsByReason explains the reasons that Roblox gives when an account
// cannot vote. EmailOrPhoneIsVerified was seen on 2026-11-27 for an account
// without a verified email or phone.
var voteRefusalsByReason = map[string]string{
	"PlayGame":               "Play the game with this account to vote.",
	"EmailOrPhoneIsVerified": "Verify the email or phone of this account on Roblox to vote.",
}

func (service *Users) SetGameVote(ctx context.Context, accountID, version, universeID int64, vote GameVote) error {
	var robloxVote *bool
	if vote != GameVoteNone {
		isLike := vote == GameVoteLike
		robloxVote = &isLike
	}
	_, err := service.writeAccount(ctx, accountID, version, "game-vote-update", http.MethodPatch, gameAddress(universeID, "/user-votes"), map[string]*bool{"vote": robloxVote})
	return err
}

// CanManageGame reads whether the account may change the settings of the
// game, as its creator or a member of the creator group with that permission.
func (service *Users) CanManageGame(ctx context.Context, accountID, version, universeID int64) (bool, error) {
	const endpoint = "game-manage-permission"
	var payload struct {
		Data *[]struct {
			UniverseID int64 `json:"universeId"`
			CanManage  bool  `json:"canManage"`
		} `json:"data"`
	}
	address := (&url.URL{Scheme: "https", Host: "develop.roblox.com", Path: "/v1/universes/multiget/permissions", RawQuery: url.Values{"ids": {strconv.FormatInt(universeID, 10)}}.Encode()}).String()
	if err := service.readAccountDetails(ctx, accountID, version, endpoint, address, &payload); err != nil {
		return false, err
	}
	if payload.Data == nil {
		return false, invalidGameAction(endpoint)
	}
	for _, listedPermission := range *payload.Data {
		if listedPermission.UniverseID == universeID {
			return listedPermission.CanManage, nil
		}
	}
	return false, nil
}

func (service *Users) GameSettings(ctx context.Context, accountID, version, universeID int64) (GameSettings, error) {
	const endpoint = "game-settings"
	var payload struct {
		Name                *string `json:"name"`
		AllowPrivateServers bool    `json:"allowPrivateServers"`
		// Roblox names a public game with the privacy type Public.
		PrivacyType string `json:"privacyType"`
	}
	if err := service.readAccountDetails(ctx, accountID, version, endpoint, universeAddress("v1", universeID, "/configuration"), &payload); err != nil {
		return GameSettings{}, err
	}
	if payload.Name == nil {
		return GameSettings{}, invalidGameAction(endpoint)
	}
	return GameSettings{
		Name: boundedText(*payload.Name, MaxGameNameLength), IsPublic: payload.PrivacyType == "Public",
		ArePrivateServersAllowed: payload.AllowPrivateServers,
	}, nil
}

// GameConfigurationChange lists the settings that the configuration API
// changes. A nil field keeps its value.
type GameConfigurationChange struct {
	Name                     *string
	Description              *string
	ArePrivateServersAllowed *bool
}

// UpdateGameSettings changes the name, the description, and the private
// server setting of the game. Roblox leaves out a setting that the request
// does not name.
func (service *Users) UpdateGameSettings(ctx context.Context, accountID, version, universeID int64, change GameConfigurationChange) error {
	request := map[string]any{}
	if change.Name != nil {
		request["name"] = *change.Name
	}
	if change.Description != nil {
		request["description"] = *change.Description
	}
	if change.ArePrivateServersAllowed != nil {
		// Roblox calls this setting allowPrivateServers.
		request["allowPrivateServers"] = *change.ArePrivateServersAllowed
	}
	if len(request) == 0 {
		return nil
	}
	_, err := service.writeAccount(ctx, accountID, version, "game-settings-update", http.MethodPatch, universeAddress("v2", universeID, "/configuration"), request)
	return explainedRefusal(err, gameConfigurationRefusalsByCode)
}

// SetGamePublic makes the game public, so that anyone can find and play it,
// or private, so that only its creator and collaborators can play it. Roblox
// calls these activate and deactivate.
func (service *Users) SetGamePublic(ctx context.Context, accountID, version, universeID int64, isPublic bool) error {
	suffix := "/deactivate"
	if isPublic {
		suffix = "/activate"
	}
	_, err := service.writeAccount(ctx, accountID, version, "game-public-update", http.MethodPost, universeAddress("v1", universeID, suffix), nil)
	return explainedRefusal(err, gameActivationRefusalsByCode)
}

// The develop API names its refusals with a code that is unique to each
// endpoint. A configuration change answers with these codes.
var gameConfigurationRefusalsByCode = map[int]string{
	2:  "This account cannot change the settings of this game.",
	7:  "Roblox rejected the name or the description. Change the text and try again.",
	8:  "Roblox rejected the name because it is too long. Shorten it and try again.",
	25: "This account cannot change the private servers of this game.",
	26: "Roblox rejected the private server price. Set a price on the Roblox website first.",
}

// An activation or a deactivation answers with these codes.
var gameActivationRefusalsByCode = map[int]string{
	3: "This account cannot change the settings of this game.",
	6: "Roblox is reviewing the root place of this game. Try again later.",
	7: "This creator has the most public games that Roblox allows. Make another game private first.",
}

// explainedRefusal replaces the message of a refusal that Roblox names with
// one of the codes.
func explainedRefusal(err error, messagesByCode map[int]string) error {
	remote, ok := errors.AsType[*roblox.Error](err)
	if !ok || (remote.Status != http.StatusBadRequest && remote.Status != http.StatusForbidden) {
		return err
	}
	message, isKnown := messagesByCode[remote.RobloxCode]
	if !isKnown {
		return err
	}
	return &roblox.Error{Kind: remote.Kind, Endpoint: remote.Endpoint, Status: remote.Status, RobloxCode: remote.RobloxCode, Message: message, Cause: err}
}

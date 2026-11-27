package appservice

import (
	"context"
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"

	robloxservices "github.com/sleaze5/RobloxAccountManager/internal/roblox/services"
)

// GameAccountStatus is how the account relates to a game on Roblox.
type GameAccountStatus struct {
	IsFavorite bool                          `json:"isFavorite"`
	Vote       robloxservices.GameVoteStatus `json:"vote"`
	CanManage  bool                          `json:"canManage"`
	// Unavailable lists the parts whose read failed: favorite, vote, or
	// manage.
	Unavailable []string `json:"unavailable"`
}

// GameSettingsChange lists the settings of a game to change on Roblox. A nil
// field keeps its value.
type GameSettingsChange struct {
	Name                     *string `json:"name"`
	Description              *string `json:"description"`
	IsPublic                 *bool   `json:"isPublic"`
	ArePrivateServersAllowed *bool   `json:"arePrivateServersAllowed"`
}

// GetGameAccountStatus reads whether the account marked the game as a
// favorite, how it voted, and whether it can manage the game. A failed part
// is listed as unavailable instead of failing the others.
func (service *Service) GetGameAccountStatus(ctx context.Context, accountID, universeID int64) (GameAccountStatus, error) {
	const operation = "game-account-status"
	ctx, version, done, err := service.beginAccountSession(ctx, accountID, operation)
	if err != nil {
		return GameAccountStatus{}, err
	}
	defer done()
	status := GameAccountStatus{Vote: robloxservices.GameVoteStatus{Vote: robloxservices.GameVoteNone}}
	reads := []accountRead{
		applyRead("favorite", func() (bool, error) {
			return service.users.IsGameFavorite(ctx, accountID, version, universeID)
		}, func(isFavorite bool) { status.IsFavorite = isFavorite }),
		applyRead("vote", func() (robloxservices.GameVoteStatus, error) {
			return service.users.GameVoteStatus(ctx, accountID, version, universeID)
		}, func(vote robloxservices.GameVoteStatus) { status.Vote = vote }),
		applyRead("manage", func() (bool, error) {
			return service.users.CanManageGame(ctx, accountID, version, universeID)
		}, func(canManage bool) { status.CanManage = canManage }),
	}
	logger := service.logs.Module("application.games").With("operation", operation, "account_id", accountID, "universe_id", universeID)
	status.Unavailable, err = service.runAccountReads(ctx, accountID, version, operation, logger, reads)
	if err != nil {
		return GameAccountStatus{}, err
	}
	return status, nil
}

func (service *Service) SetGameFavorite(ctx context.Context, accountID, universeID int64, isFavorite bool) error {
	return service.editAccount(ctx, accountID, "game-favorite-update", "Roblox did not let this account change its favorites.", func(ctx context.Context, version int64) error {
		return service.users.SetGameFavorite(ctx, accountID, version, universeID, isFavorite)
	})
}

func (service *Service) SetGameVote(ctx context.Context, accountID, universeID int64, vote robloxservices.GameVote) error {
	const operation = "game-vote-update"
	if !robloxservices.ValidGameVote(vote) {
		return accountEditError(operation, "Select like, dislike, or no vote.")
	}
	return service.editAccount(ctx, accountID, operation, "Roblox did not accept the vote. Play the game with this account first.", func(ctx context.Context, version int64) error {
		return service.users.SetGameVote(ctx, accountID, version, universeID, vote)
	})
}

func (service *Service) GetGameSettings(ctx context.Context, accountID, universeID int64) (robloxservices.GameSettings, error) {
	const operation = "game-settings"
	ctx, version, done, err := service.beginAccountSession(ctx, accountID, operation)
	if err != nil {
		return robloxservices.GameSettings{}, err
	}
	defer done()
	settings, err := service.users.GameSettings(ctx, accountID, version, universeID)
	if err != nil {
		service.logs.Module("application.games").WarnContext(ctx, "game settings unavailable", "operation", operation, "account_id", accountID, "universe_id", universeID, "error", err)
		return robloxservices.GameSettings{}, err
	}
	return settings, nil
}

// SaveGameSettings changes the settings of the game on Roblox and returns
// them as Roblox saved them.
func (service *Service) SaveGameSettings(ctx context.Context, accountID, universeID int64, change GameSettingsChange) (robloxservices.GameSettings, error) {
	const operation = "game-settings-update"
	configuration, err := validateGameNameAndDescription(change, operation)
	if err != nil {
		return robloxservices.GameSettings{}, err
	}
	var settings robloxservices.GameSettings
	err = service.editAccount(ctx, accountID, operation, "", func(ctx context.Context, version int64) error {
		if err := service.users.UpdateGameSettings(ctx, accountID, version, universeID, configuration); err != nil {
			return err
		}
		if change.IsPublic != nil {
			if err := service.users.SetGamePublic(ctx, accountID, version, universeID, *change.IsPublic); err != nil {
				return err
			}
		}
		var readError error
		settings, readError = service.users.GameSettings(ctx, accountID, version, universeID)
		return readError
	})
	return settings, err
}

func validateGameNameAndDescription(change GameSettingsChange, operation string) (robloxservices.GameConfigurationChange, error) {
	configuration := robloxservices.GameConfigurationChange{ArePrivateServersAllowed: change.ArePrivateServersAllowed}
	if change.Name != nil {
		name := strings.TrimSpace(*change.Name)
		if name == "" || utf8.RuneCountInString(name) > robloxservices.MaxGameNameLength || strings.ContainsFunc(name, unicode.IsControl) {
			return configuration, accountEditError(operation, fmt.Sprintf("Enter a name of 1 to %d characters.", robloxservices.MaxGameNameLength))
		}
		configuration.Name = &name
	}
	if change.Description != nil {
		description := strings.TrimSpace(*change.Description)
		if utf8.RuneCountInString(description) > robloxservices.MaxGameDescriptionLength || !utf8.ValidString(description) {
			return configuration, accountEditError(operation, fmt.Sprintf("Shorten the description to %d characters or fewer.", robloxservices.MaxGameDescriptionLength))
		}
		configuration.Description = &description
	}
	return configuration, nil
}

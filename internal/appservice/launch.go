package appservice

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/sleaze5/RobloxAccountManager/internal/accounts"
	"github.com/sleaze5/RobloxAccountManager/internal/gamelaunch"
	"github.com/sleaze5/RobloxAccountManager/internal/platform/robloxmulti"
	"github.com/sleaze5/RobloxAccountManager/internal/roblox"
	robloxservices "github.com/sleaze5/RobloxAccountManager/internal/roblox/services"
)

var launchOperationID atomic.Uint64

type AccountLaunchFailure struct {
	AccountID int64  `json:"accountId"`
	Message   string `json:"message"`
}

type LaunchResult struct {
	Launched  []int64                `json:"launched"`
	Failures  []AccountLaunchFailure `json:"failures"`
	Cancelled bool                   `json:"cancelled"`
}

func (service *Service) LaunchAccounts(ctx context.Context, accountIDs []int64, input gamelaunch.Input) (LaunchResult, error) {
	result := LaunchResult{Launched: []int64{}, Failures: []AccountLaunchFailure{}}
	if _, err := gamelaunch.Parse(input); err != nil {
		return result, launchPreparationError(err)
	}
	service.launchMu.Lock()
	defer service.launchMu.Unlock()
	if ctx.Err() != nil || !service.vault.Unlocked() {
		return result, launchError(roblox.KindCancelled, "The launch was cancelled or the vault is locked.")
	}
	if len(accountIDs) > 1 {
		approved, err := service.confirmBatchLaunch(ctx, len(accountIDs))
		if err != nil || !approved {
			result.Cancelled = true
			return result, err
		}
	}
	for _, accountID := range accountIDs {
		if ctx.Err() != nil || !service.vault.Unlocked() {
			result.Cancelled = true
			break
		}
		err := service.launchAccount(ctx, accountID, input)
		if errors.Is(err, robloxmulti.ErrLaunchCancelled) {
			result.Cancelled = true
			break
		}
		if err != nil {
			message := "The account could not be launched. Try again."
			var failure *roblox.Error
			if errors.As(err, &failure) {
				message = failure.Message
			}
			result.Failures = append(result.Failures, AccountLaunchFailure{AccountID: accountID, Message: message})
			continue
		}
		result.Launched = append(result.Launched, accountID)
	}
	return result, nil
}

func (service *Service) confirmBatchLaunch(ctx context.Context, count int) (bool, error) {
	state := service.multiInstance.Snapshot()
	if state.Enabled && state.Ready {
		return true, nil
	}
	ctx, cancel := context.WithTimeout(ctx, 90*time.Second)
	defer cancel()
	return service.confirmLaunch(ctx, LaunchConfirmation{
		AccountCount: count, MultiInstanceEnabled: state.Enabled, MultiInstanceMessage: state.Message,
	})
}

func (service *Service) launchAccount(ctx context.Context, accountID int64, input gamelaunch.Input) error {
	return service.runLaunchOperation(ctx, accountID, input, "game-launch", func(ctx context.Context, ticket, browserID string, target gamelaunch.Request) error {
		if err := service.launcher.Launch(ctx, ticket, browserID, target); err != nil {
			message := "Roblox Player could not be started. Check that Roblox is installed and try again."
			if errors.Is(err, gamelaunch.ErrDesktopUnavailable) {
				message = "Roblox could not be launched without administrator permissions. Restart Windows Explorer normally and try again."
			}
			return &roblox.Error{Kind: roblox.KindProtocol, Endpoint: "game-launch", Message: message, Cause: err}
		}
		service.multiInstance.AfterLaunch()
		return nil
	})
}

func (service *Service) CopyLaunchOptions(ctx context.Context, accountIDs []int64, input gamelaunch.Input, copyText func(string) bool) error {
	commands := make([]string, 0, len(accountIDs))
	sessions := make([]roblox.SessionSnapshot, 0, len(accountIDs))
	for _, accountID := range accountIDs {
		session, err := service.sessions.Snapshot(accountID)
		if err != nil {
			return err
		}
		sessions = append(sessions, session)
		command, err := service.accountLaunchOptions(ctx, accountID, input)
		if err != nil {
			return err
		}
		commands = append(commands, command)
	}
	for _, session := range sessions {
		if err := service.checkClipboardAccount(ctx, session.AccountID); err != nil {
			return err
		}
		if session.Context.Err() != nil || !service.sessions.VersionCurrent(session.AccountID, session.SecretVersion) {
			return clipboardError("An account session changed. Try again.")
		}
	}
	if !copyText(strings.Join(commands, "\n")) {
		service.logger.Warn("launch options clipboard write failed", "operation", "copy-launch-options", "account_count", len(accountIDs))
		return clipboardError("Launch options could not be copied. Try again.")
	}
	return nil
}

func (service *Service) accountLaunchOptions(ctx context.Context, accountID int64, input gamelaunch.Input) (string, error) {
	var command string
	err := service.runLaunchOperation(ctx, accountID, input, "copy-launch-options", func(ctx context.Context, ticket, browserID string, target gamelaunch.Request) error {
		var err error
		command, err = service.launcher.CommandLine(ctx, ticket, browserID, target)
		if err != nil {
			return &roblox.Error{Kind: roblox.KindProtocol, Endpoint: "copy-launch-options", Message: "Launch options could not be prepared. Check that Roblox is installed and try again.", Cause: err}
		}
		if ctx.Err() != nil {
			return launchError(roblox.KindCancelled, "The account session changed. Try again.")
		}
		return nil
	})
	return command, err
}

func (service *Service) runLaunchOperation(ctx context.Context, accountID int64, input gamelaunch.Input, operation string, action func(context.Context, string, string, gamelaunch.Request) error) (result error) {
	logger := service.logger.With("operation_id", fmt.Sprintf("launch-%d", launchOperationID.Add(1)), "operation", operation, "account_id", accountID)
	stage := "parse-input"
	cancelled := false
	defer func() {
		if cancelled {
			logger.Info("Roblox launch cancelled")
			return
		}
		if result == nil {
			logger.Info("Roblox launch operation completed")
			return
		}
		var remote *roblox.Error
		kind, status, code := roblox.KindProtocol, 0, 0
		endpoint, message := operation, "The launch failed before a Roblox response was available."
		if errors.As(result, &remote) {
			kind, status, code = remote.Kind, remote.Status, remote.RobloxCode
			endpoint, message = remote.Endpoint, remote.Message
		}
		// Do not log Cause: transport and process errors can include secret URLs.
		logger.Warn("Roblox launch operation failed", "stage", stage, "endpoint", endpoint, "error_kind", kind, "status", status, "roblox_code", code, "error_message", message)
	}()
	target, err := gamelaunch.Parse(input)
	if err != nil {
		message := err.Error()
		return launchError(roblox.KindProtocol, strings.ToUpper(message[:1])+message[1:]+".")
	}
	logger = logger.With("method", input.Method, "share_type", target.ShareType)
	stage = "account-session"
	session, err := service.sessions.Snapshot(accountID)
	if err != nil {
		return err
	}
	if session.State == accounts.StateDisabled {
		return launchError(roblox.KindForbidden, "Network use is disabled for this account.")
	}
	ctx, cancel := context.WithTimeout(ctx, 90*time.Second)
	defer cancel()
	stop := context.AfterFunc(session.Context, cancel)
	defer stop()
	if session.Context.Err() != nil {
		cancel()
	}
	logger.Debug("Preparing Roblox launch")
	if input.Method == gamelaunch.MethodUser {
		stage = "resolve-user"
		target, err = service.resolveLaunchUser(ctx, accountID, target, input.ResolveCurrentGame)
	} else {
		stage = "resolve-link"
		target, err = service.gameLinks.Resolve(ctx, accountID, session.SecretVersion, target)
	}
	if err != nil {
		return err
	}
	target.Teleport = input.Teleport
	target.DirectLaunch = input.DirectLaunch
	if launchData := strings.TrimSpace(input.LaunchData); launchData != "" {
		target.LaunchData = launchData
	}
	var ticketValue string
	loadTicket := func() error {
		stage = "authentication-ticket"
		ticket, err := service.auth.AuthenticationTicket(ctx, accountID)
		if err == nil {
			ticketValue = ticket.Value()
		}
		return err
	}
	if operation == "game-launch" {
		stage = "prepare-launch"
		confirm := func(ctx context.Context, count int, multiInstance bool) (bool, error) {
			approved, err := service.confirmRobloxReplacement(ctx, count, multiInstance)
			if err != nil || !approved {
				return approved, err
			}
			err = loadTicket()
			if err == nil {
				stage = "prepare-launch"
			}
			return true, err
		}
		if err := service.multiInstance.PrepareForLaunch(ctx, confirm); err != nil {
			if errors.Is(err, robloxmulti.ErrLaunchCancelled) {
				cancelled = true
				return err
			}
			var failure *roblox.Error
			if errors.As(err, &failure) {
				return failure
			}
			return launchPreparationError(err)
		}
	}
	if ticketValue == "" {
		if err := loadTicket(); err != nil {
			return err
		}
	}
	stage = "validate-session"
	if ctx.Err() != nil || !service.vault.Unlocked() || !service.sessions.VersionCurrent(accountID, session.SecretVersion) {
		return launchError(roblox.KindCancelled, "The launch ended or the account session changed. Try again.")
	}
	stage = operation
	return action(ctx, ticketValue, session.BrowserID, target)
}

func (service *Service) resolveLaunchUser(ctx context.Context, accountID int64, target gamelaunch.Request, resolveCurrentGame bool) (gamelaunch.Request, error) {
	userID := target.UserID
	if target.Username != "" {
		var err error
		userID, err = service.users.UserIDFromUsername(ctx, target.Username)
		if err != nil {
			return gamelaunch.Request{}, err
		}
	}
	if !resolveCurrentGame {
		return gamelaunch.Request{UserID: userID}, nil
	}
	presences, err := service.presence.AccountPresences(ctx, []robloxservices.PresenceTarget{{AccountID: accountID, RobloxUserID: userID}})
	if err != nil {
		return gamelaunch.Request{}, err
	}
	if len(presences) != 1 {
		return gamelaunch.Request{}, launchError(roblox.KindProtocol, "Roblox did not return this user's current server.")
	}
	presence := presences[0]
	if presence.UserPresenceType != robloxservices.PresenceTypeInGame || presence.PlaceID == nil || *presence.PlaceID <= 0 || presence.GameID == "" {
		return gamelaunch.Request{}, launchError(roblox.KindProtocol, "This user has no visible, joinable server. They may be offline or their privacy settings may hide it.")
	}
	resolved, err := gamelaunch.Parse(gamelaunch.Input{Method: gamelaunch.MethodPlace, PlaceID: strconv.FormatInt(*presence.PlaceID, 10), JobID: presence.GameID})
	if err != nil {
		return gamelaunch.Request{}, launchError(roblox.KindProtocol, "Roblox returned an invalid server for this user. Try again.")
	}
	return resolved, nil
}

func launchError(kind roblox.ErrorKind, message string) error {
	return &roblox.Error{Kind: kind, Endpoint: "game-launch", Message: message}
}

func launchPreparationError(err error) *roblox.Error {
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return &roblox.Error{Kind: roblox.KindCancelled, Endpoint: "game-launch", Message: "The launch was cancelled or timed out. Try joining again."}
	}
	message := err.Error()
	return &roblox.Error{Kind: roblox.KindProtocol, Endpoint: "game-launch", Message: strings.ToUpper(message[:1]) + message[1:] + "."}
}

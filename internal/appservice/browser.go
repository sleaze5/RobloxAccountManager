package appservice

import (
	"context"
	"errors"
	"fmt"

	"github.com/sleaze5/RobloxAccountManager/internal/browser"
	"github.com/sleaze5/RobloxAccountManager/internal/gamelaunch"
	"github.com/sleaze5/RobloxAccountManager/internal/platform/robloxmulti"
	"github.com/sleaze5/RobloxAccountManager/internal/roblox"
)

func (service *Service) GetBrowserState() browser.Snapshot {
	if service.browser == nil {
		return browser.Snapshot{}
	}
	return service.browser.Snapshot()
}

func (service *Service) GetBrowserShutdownEffects() browser.ShutdownEffects {
	if service.browser == nil {
		return browser.ShutdownEffects{}
	}
	return service.browser.Effects()
}

func (service *Service) DownloadBrowserRuntime(ctx context.Context) error {
	if service.browser == nil {
		return errors.New("browser runtime management is unavailable")
	}
	return service.browser.DownloadRuntime(context.WithoutCancel(ctx), false)
}

func (service *Service) CancelBrowserRuntimeDownload() {
	if service.runtime != nil {
		service.runtime.CancelDownload()
	}
}

func (service *Service) RedownloadBrowserRuntime(ctx context.Context) error {
	if service.browser == nil {
		return errors.New("browser runtime management is unavailable")
	}
	return service.browser.DownloadRuntime(context.WithoutCancel(ctx), true)
}

func (service *Service) RemoveBrowserRuntime() error {
	if service.browser == nil {
		return errors.New("browser runtime management is unavailable")
	}
	return service.browser.RemoveRuntime()
}

func (service *Service) StartLoginBrowser(ctx context.Context) (browser.SessionState, error) {
	if !service.vault.Unlocked() {
		return browser.SessionState{}, errors.New("unlock the vault before opening a browser")
	}
	return service.browser.StartLogin(ctx)
}

func (service *Service) OpenAccountBrowser(ctx context.Context, accountID int64) (browser.SessionState, error) {
	if !service.vault.Unlocked() {
		return browser.SessionState{}, errors.New("unlock the vault before opening a browser")
	}
	return service.browser.OpenAccount(ctx, accountID)
}

func (service *Service) FocusBrowserSession(sessionID string) error {
	return service.browser.Focus(sessionID)
}
func (service *Service) CloseBrowserSession(sessionID string) error {
	return service.browser.Close(sessionID)
}
func (service *Service) CloseAllLoginBrowsers() { service.browser.CloseAllLogin() }
func (service *Service) SaveBrowserCandidates(ctx context.Context, selected []string) (browser.SaveResult, error) {
	return service.browser.SaveCandidates(ctx, selected)
}
func (service *Service) DiscardBrowserCandidates() { service.browser.DiscardCandidates() }
func (service *Service) RetryBrowserCleanup()      { service.browser.RetryCleanup() }

func (service *Service) LaunchBrowserGame(ctx context.Context, sessionID, protocolURL string) {
	if ctx.Err() != nil || !service.vault.Unlocked() {
		return
	}
	logger := service.logger.With("operation_id", fmt.Sprintf("launch-%d", launchOperationID.Add(1)), "operation", "browser-game-launch", "session_id", sessionID)
	if err := gamelaunch.ValidateBrowserProtocolURL(protocolURL); err != nil {
		logger.Warn("invalid browser game launch rejected", "error", err)
		service.events.BrowserLaunchFailed(sessionID, "The browser supplied an invalid game launch. Try joining again.")
		return
	}
	err := service.serializeLaunch(ctx, func() error {
		return service.executeLaunch(ctx, service.confirmRobloxReplacement, func() error {
			if err := service.launcher.LaunchProtocol(ctx, protocolURL); err != nil {
				return &roblox.Error{Kind: roblox.KindProtocol, Endpoint: "browser-game-launch", Message: userLaunchMessage(err, "Roblox could not be started. Check that Roblox is installed and try joining again."), Cause: err}
			}
			return nil
		})
	})
	if err != nil {
		var failure *roblox.Error
		if errors.Is(err, robloxmulti.ErrLaunchCancelled) || ctx.Err() != nil || errors.As(err, &failure) && failure.Kind == roblox.KindCancelled {
			logger.Info("browser game launch cancelled")
			return
		}
		message := "Roblox could not be started. Try joining again."
		if errors.As(err, &failure) {
			message = failure.Message
		}
		logger.Warn("browser game launch failed", "error_message", message)
		service.events.BrowserLaunchFailed(sessionID, message)
		return
	}
	logger.Info("browser game launched")
}

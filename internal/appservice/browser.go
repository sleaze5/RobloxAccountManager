package appservice

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/sleaze5/RobloxAccountManager/internal/browser"
	"github.com/sleaze5/RobloxAccountManager/internal/gamelaunch"
	"github.com/sleaze5/RobloxAccountManager/internal/platform/robloxmulti"
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
	if service.runtime == nil {
		return errors.New("browser runtime management is unavailable")
	}
	return service.runtime.Download(context.WithoutCancel(ctx), false)
}

func (service *Service) CancelBrowserRuntimeDownload() {
	if service.runtime != nil {
		service.runtime.CancelDownload()
	}
}

func (service *Service) RedownloadBrowserRuntime(ctx context.Context) error {
	if service.browser == nil || service.runtime == nil {
		return errors.New("browser runtime management is unavailable")
	}
	service.browser.ForceShutdown()
	if err := service.runtime.StopDownload(30 * time.Second); err != nil {
		return err
	}
	if service.vault.Unlocked() {
		service.browser.AllowLaunches()
	}
	return service.runtime.Download(context.WithoutCancel(ctx), true)
}

func (service *Service) RemoveBrowserRuntime() error {
	if service.browser == nil || service.runtime == nil {
		return errors.New("browser runtime management is unavailable")
	}
	service.browser.ForceShutdown()
	err := service.runtime.Remove()
	if service.vault.Unlocked() {
		service.browser.AllowLaunches()
	}
	return err
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
	service.launchMu.Lock()
	defer service.launchMu.Unlock()
	if err := service.multiInstance.PrepareForLaunch(ctx, service.confirmRobloxReplacement); err != nil {
		if errors.Is(err, robloxmulti.ErrLaunchCancelled) || ctx.Err() != nil {
			logger.Info("browser game launch cancelled")
			return
		}
		failure := launchPreparationError(err)
		logger.Warn("browser game launch preparation failed", "error_message", failure.Message)
		service.events.BrowserLaunchFailed(sessionID, failure.Message)
		return
	}
	if err := service.launcher.LaunchProtocol(ctx, protocolURL); err != nil {
		logger.Warn("browser game launch failed", "error", err)
		if ctx.Err() != nil {
			return
		}
		service.events.BrowserLaunchFailed(sessionID, userLaunchMessage(err, "Roblox could not be started. Check that Roblox is installed and try joining again."))
		return
	}
	service.multiInstance.AfterLaunch()
	logger.Info("browser game launched through desktop Explorer")
}

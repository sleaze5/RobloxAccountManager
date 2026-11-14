package bindings

import (
	"context"

	"github.com/sleaze5/RobloxAccountManager/internal/browser"
)

func (service *Service) GetBrowserState() browser.Snapshot { return service.core.GetBrowserState() }

func (service *Service) GetBrowserShutdownEffects() browser.ShutdownEffects {
	return service.core.GetBrowserShutdownEffects()
}

func (service *Service) DownloadBrowserRuntime(ctx context.Context) error {
	return service.core.DownloadBrowserRuntime(ctx)
}

func (service *Service) CancelBrowserRuntimeDownload() { service.core.CancelBrowserRuntimeDownload() }

func (service *Service) RedownloadBrowserRuntime(ctx context.Context) error {
	return service.core.RedownloadBrowserRuntime(ctx)
}

func (service *Service) RemoveBrowserRuntime() error { return service.core.RemoveBrowserRuntime() }

func (service *Service) StartLoginBrowser(ctx context.Context) (browser.SessionState, error) {
	return service.core.StartLoginBrowser(ctx)
}

func (service *Service) OpenAccountBrowser(ctx context.Context, accountID int64, pageURL string) (browser.SessionState, error) {
	if err := positiveID(accountID, "browser-open"); err != nil {
		return browser.SessionState{}, err
	}
	return service.core.OpenAccountBrowser(ctx, accountID, pageURL)
}

func (service *Service) FocusBrowserSession(sessionID string) error {
	return service.core.FocusBrowserSession(sessionID)
}

func (service *Service) CloseBrowserSession(sessionID string) error {
	return service.core.CloseBrowserSession(sessionID)
}

func (service *Service) CloseAllLoginBrowsers() { service.core.CloseAllLoginBrowsers() }

func (service *Service) SaveBrowserCandidates(ctx context.Context, selected []string) (browser.SaveResult, error) {
	return service.core.SaveBrowserCandidates(ctx, selected)
}

func (service *Service) DiscardBrowserCandidates() { service.core.DiscardBrowserCandidates() }

func (service *Service) RetryBrowserCleanup() { service.core.RetryBrowserCleanup() }

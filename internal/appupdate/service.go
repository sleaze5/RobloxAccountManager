package appupdate

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/sleaze5/RobloxAccountManager/internal/appmeta"
	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/updater"
	"github.com/wailsapp/wails/v3/pkg/updater/providers/endpoint"
)

const manifestURL = "https://github.com/sleaze5/RobloxAccountManager/releases/latest/download/manifest.json"

const checkTimeout = 30 * time.Second

type Status string

const (
	StatusIdle        Status = "idle"
	StatusChecking    Status = "checking"
	StatusUpToDate    Status = "up-to-date"
	StatusAvailable   Status = "available"
	StatusDownloading Status = "downloading"
	StatusReady       Status = "ready"
	StatusRestarting  Status = "restarting"
)

type State struct {
	Status           Status `json:"status"`
	CurrentVersion   string `json:"currentVersion"`
	AvailableVersion string `json:"availableVersion,omitempty"`
	DownloadedBytes  int64  `json:"downloadedBytes"`
	TotalBytes       int64  `json:"totalBytes"`
	Error            string `json:"error,omitempty"`
}

var errUnavailable = errors.New("updates are unavailable")

type Service struct {
	logger  *slog.Logger
	changed func()

	mu       sync.Mutex
	updater  *updater.Updater
	state    State
	cancel   context.CancelFunc
	progress func()
}

func New(logger *slog.Logger, changed func()) *Service {
	return &Service{logger: logger, changed: changed, state: State{Status: StatusIdle, CurrentVersion: appmeta.Version}}
}

func (service *Service) Start(ctx context.Context, app *application.App) {
	provider, err := endpoint.New(endpoint.Config{
		URL: manifestURL,
		HTTPClient: &http.Client{Transport: &http.Transport{
			Proxy:                 http.ProxyFromEnvironment,
			TLSHandshakeTimeout:   10 * time.Second,
			ResponseHeaderTimeout: checkTimeout,
		}},
	})
	if err == nil {
		err = app.Updater.Init(updater.Config{
			CurrentVersion: appmeta.Version,
			Providers:      []updater.Provider{provider},
			PublicKey:      appmeta.UpdaterPublicKey,
			Window:         updater.WindowNone,
		})
	}
	if err != nil {
		service.logger.Error("updater could not be configured", "operation", "update-startup", "error", err)
		return
	}
	service.mu.Lock()
	service.updater = app.Updater
	service.progress = app.Event.On(updater.EventDownloadProgress, service.recordProgress)
	service.mu.Unlock()
	go service.removeLeftovers()
	go func() { _ = service.Check(ctx) }()
}

func (service *Service) Shutdown() {
	service.mu.Lock()
	cancel, removeProgress := service.cancel, service.progress
	staged := ""
	if service.updater != nil && service.state.Status == StatusReady {
		staged = service.updater.DownloadedPath()
	}
	service.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	if removeProgress != nil {
		removeProgress()
	}
	if dir := filepath.Dir(staged); staged != "" && strings.HasPrefix(filepath.Base(dir), "wails-update-") {
		if err := os.RemoveAll(dir); err != nil {
			service.logger.Warn("downloaded update could not be removed", "operation", "update-cleanup", "path", dir, "error", err)
		}
	}
}

func (service *Service) State() State {
	service.mu.Lock()
	defer service.mu.Unlock()
	return service.state
}

func (service *Service) Check(ctx context.Context) error {
	service.mu.Lock()
	if service.updater == nil {
		service.mu.Unlock()
		return errUnavailable
	}
	switch service.state.Status {
	case StatusChecking, StatusDownloading, StatusReady, StatusRestarting:
		service.mu.Unlock()
		return nil
	}
	service.state = State{Status: StatusChecking, CurrentVersion: appmeta.Version}
	current := service.updater
	service.mu.Unlock()
	service.changed()

	checkContext, cancel := context.WithTimeout(ctx, checkTimeout)
	release, err := current.Check(checkContext)
	cancel()
	if err == nil && release != nil {
		err = requireSignature(release)
	}
	service.finishCheck(release, err)
	return nil
}

// requireSignature rejects unsigned releases. The Wails updater verifies a
// signature only when the manifest provides one, so a manifest without
// signatures would otherwise install an unverified file.
func requireSignature(release *updater.Release) error {
	if release.Verification == nil || release.Verification.SignatureAlgo == "" || len(release.Verification.Signature) == 0 {
		return errors.New("release " + release.Version + " is not signed")
	}
	return nil
}

func (service *Service) finishCheck(release *updater.Release, err error) {
	service.mu.Lock()
	switch {
	case err != nil:
		service.logger.Warn("update check failed", "operation", "update-check", "error", err)
		service.state = State{Status: StatusIdle, CurrentVersion: appmeta.Version, Error: "Updates could not be checked. Check your connection and try again."}
	case release == nil:
		service.logger.Info("application is up to date", "operation", "update-check", "version", appmeta.Version)
		service.state = State{Status: StatusUpToDate, CurrentVersion: appmeta.Version}
	default:
		service.logger.Info("update available", "operation", "update-check", "version", release.Version)
		service.state = State{Status: StatusAvailable, CurrentVersion: appmeta.Version, AvailableVersion: release.Version, TotalBytes: release.Artifact.Size}
	}
	service.mu.Unlock()
	service.changed()
}

func (service *Service) Install(ctx context.Context) error {
	service.mu.Lock()
	if service.updater == nil {
		service.mu.Unlock()
		return errUnavailable
	}
	if service.state.Status != StatusAvailable {
		service.mu.Unlock()
		return errors.New("no update is available to download")
	}
	downloadContext, cancel := context.WithCancel(context.WithoutCancel(ctx))
	service.cancel = cancel
	service.state.Status = StatusDownloading
	service.state.DownloadedBytes = 0
	service.state.Error = ""
	current := service.updater
	service.mu.Unlock()
	service.changed()

	go service.download(downloadContext, current)
	return nil
}

func (service *Service) download(ctx context.Context, current *updater.Updater) {
	err := current.DownloadAndInstall(ctx)
	service.mu.Lock()
	service.cancel()
	service.cancel = nil
	switch {
	case err == nil:
		service.logger.Info("update downloaded and verified", "operation", "update-install", "version", service.state.AvailableVersion)
		service.state.Status = StatusReady
	case ctx.Err() != nil:
		service.logger.Info("update download canceled", "operation", "update-install", "version", service.state.AvailableVersion)
		service.state.Status = StatusAvailable
	default:
		service.logger.Warn("update download failed", "operation", "update-install", "version", service.state.AvailableVersion, "error", err)
		service.state.Status = StatusAvailable
		service.state.Error = "The update could not be downloaded or verified. Try again."
	}
	service.mu.Unlock()
	service.changed()
}

func (service *Service) CancelDownload() {
	service.mu.Lock()
	cancel := service.cancel
	service.mu.Unlock()
	if cancel != nil {
		cancel()
	}
}

func (service *Service) Restart(ctx context.Context) error {
	service.mu.Lock()
	if service.updater == nil {
		service.mu.Unlock()
		return errUnavailable
	}
	if service.state.Status != StatusReady {
		service.mu.Unlock()
		return errors.New("no update is ready to install")
	}
	service.state.Status = StatusRestarting
	service.state.Error = ""
	current := service.updater
	service.mu.Unlock()
	service.changed()

	service.logger.Info("restarting to install update", "operation", "update-install", "version", service.State().AvailableVersion)
	_ = os.Setenv(helperLogVariable, helperLogPath())
	if err := current.Restart(ctx); err != nil {
		_ = os.Unsetenv(helperLogVariable)
		service.logger.Warn("update restart failed", "operation", "update-install", "error", err)
		service.mu.Lock()
		service.state.Status = StatusReady
		service.state.Error = "The application could not restart to install the update. Try again."
		service.mu.Unlock()
		service.changed()
	}
	return nil
}

func (service *Service) recordProgress(event *application.CustomEvent) {
	progress, ok := event.Data.(updater.Progress)
	if !ok {
		return
	}
	service.mu.Lock()
	if service.state.Status != StatusDownloading {
		service.mu.Unlock()
		return
	}
	service.state.DownloadedBytes = progress.Written
	if progress.Total > 0 {
		service.state.TotalBytes = progress.Total
	}
	service.mu.Unlock()
	service.changed()
}

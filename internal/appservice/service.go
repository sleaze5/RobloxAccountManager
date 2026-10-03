package appservice

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"time"

	"github.com/sleaze5/RobloxAccountManager/internal/appsettings"
	"github.com/sleaze5/RobloxAccountManager/internal/browser"
	"github.com/sleaze5/RobloxAccountManager/internal/gamelaunch"
	"github.com/sleaze5/RobloxAccountManager/internal/integration/rovalra"
	"github.com/sleaze5/RobloxAccountManager/internal/logging"
	"github.com/sleaze5/RobloxAccountManager/internal/logsexplorer"
	"github.com/sleaze5/RobloxAccountManager/internal/platform/robloxmulti"
	"github.com/sleaze5/RobloxAccountManager/internal/roblox"
	robloxservices "github.com/sleaze5/RobloxAccountManager/internal/roblox/services"
	gamestore "github.com/sleaze5/RobloxAccountManager/internal/storage/games"
	"github.com/sleaze5/RobloxAccountManager/internal/storage/vault"
)

// Service coordinates account, vault, Roblox, and launch workflows.
type Service struct {
	vault           Vault
	repo            AccountRepository
	sessions        *roblox.SessionManager
	users           *robloxservices.Users
	accountSettings *robloxservices.AccountSettings
	chat            *robloxservices.Chat
	auth            *robloxservices.Auth
	avatars         *robloxservices.Thumbnails
	presence        *robloxservices.Presence
	gameLinks       *robloxservices.GameLinks
	games           *robloxservices.Games
	rovalra         *rovalra.Client
	favoritePlaces  *gamestore.Repository
	logsExplorer    *logsexplorer.Reader
	launcher        gamelaunch.Launcher
	multiInstance   *robloxmulti.Manager
	settings        *appsettings.Store
	logs            *logging.System
	events          *Events
	logger          *slog.Logger
	browser         *browser.Coordinator
	runtime         *browser.RuntimeManager

	importMu             sync.Mutex
	launchMu             sync.Mutex
	pendingImports       map[string]pendingImportBatch
	validationMu         sync.Mutex
	validationNext       uint64
	validationCancels    map[uint64]context.CancelFunc
	accountSettingsMu    sync.Mutex
	accountSettingsLocks map[int64]*accountSettingsLock
	cookieRenewals       sync.Map

	mu                     sync.Mutex
	lifecycle              context.Context
	gamesContext           context.Context
	gamesCancel            context.CancelFunc
	backgroundCancel       context.CancelFunc
	startupError           string
	reminderDue            bool
	launchConfirmation     *pendingLaunchConfirmation
	launchConfirmationNext uint64
}

func New(
	vaultManager Vault,
	repository AccountRepository,
	sessions *roblox.SessionManager,
	users *robloxservices.Users,
	accountSettings *robloxservices.AccountSettings,
	chat *robloxservices.Chat,
	auth *robloxservices.Auth,
	avatars *robloxservices.Thumbnails,
	presence *robloxservices.Presence,
	gameLinks *robloxservices.GameLinks,
	games *robloxservices.Games,
	rovalraClient *rovalra.Client,
	favoritePlaces *gamestore.Repository,
	logsExplorerReader *logsexplorer.Reader,
	launcher gamelaunch.Launcher,
	multiInstance *robloxmulti.Manager,
	settings *appsettings.Store,
	logs *logging.System,
	events *Events,
	loggers ...*slog.Logger,
) *Service {
	logger := slog.New(slog.DiscardHandler)
	if len(loggers) > 0 && loggers[0] != nil {
		logger = loggers[0]
	}
	return &Service{
		vault:                vaultManager,
		repo:                 repository,
		sessions:             sessions,
		users:                users,
		accountSettings:      accountSettings,
		accountSettingsLocks: make(map[int64]*accountSettingsLock),
		chat:                 chat,
		auth:                 auth,
		avatars:              avatars,
		presence:             presence,
		gameLinks:            gameLinks,
		games:                games,
		rovalra:              rovalraClient,
		favoritePlaces:       favoritePlaces,
		logsExplorer:         logsExplorerReader,
		launcher:             launcher,
		multiInstance:        multiInstance,
		settings:             settings,
		logs:                 logs,
		events:               events,
		logger:               logger,
		lifecycle:            context.Background(),
		pendingImports:       make(map[string]pendingImportBatch),
		validationCancels:    make(map[uint64]context.CancelFunc),
	}
}

func (service *Service) Start(ctx context.Context) error {
	started := time.Now()
	service.logger.Info("vault startup started", "operation", "vault-startup")
	service.mu.Lock()
	service.lifecycle = ctx
	service.mu.Unlock()

	if err := service.vault.AutoUnlock(ctx); err == nil {
		if service.vault.FileState() == vault.FileStateIncomplete {
			service.setStartupError("The portable vault key is missing or damaged. Create a new master password to repair it.")
		} else {
			service.finishUnlock(true)
		}
	} else if !errors.Is(err, vault.ErrNotInitialized) && !errors.Is(err, vault.ErrLocked) {
		message := "Automatic unlock failed. Enter the master password to recover the vault."
		switch {
		case errors.Is(err, vault.ErrIncomplete):
			message = "The vault files are incomplete. Restore a backup or reset the vault."
		case errors.Is(err, vault.ErrUnsupportedVersion), errors.Is(err, vault.ErrUnsupportedSchema):
			message = "The vault version is unsupported. Restore a compatible backup or reset the vault."
		case errors.Is(err, vault.ErrVaultMismatch):
			message = "The database and security files belong to different vaults. Restore a backup or reset the vault."
		}
		service.setStartupError(message)
		service.logger.Warn("automatic vault unlock failed", "error", err)
	} else {
		service.logger.Debug("automatic vault unlock unavailable", "reason", err)
	}

	service.logger.Info("vault startup completed", "operation", "vault-startup", "duration_ms", float64(time.Since(started))/float64(time.Millisecond))
	return nil
}

func (service *Service) Shutdown() error {
	service.logger.Info("application service shutting down")
	service.stopGames()
	service.cancelLaunchConfirmation()
	if service.browser != nil {
		service.browser.Shutdown()
	}
	service.multiInstance.Close()
	service.cancelPendingValidations()
	service.stopBackground()
	service.clearPendingImports()
	service.sessions.Lock()
	return service.vault.Close()
}

func (service *Service) beginValidation(parent context.Context) (context.Context, func()) {
	ctx, cancel := context.WithCancel(parent)
	service.validationMu.Lock()
	service.validationNext++
	id := service.validationNext
	service.validationCancels[id] = cancel
	service.validationMu.Unlock()
	return ctx, func() {
		cancel()
		service.validationMu.Lock()
		delete(service.validationCancels, id)
		service.validationMu.Unlock()
	}
}

func (service *Service) cancelPendingValidations() {
	service.validationMu.Lock()
	for id, cancel := range service.validationCancels {
		cancel()
		delete(service.validationCancels, id)
	}
	service.validationMu.Unlock()
}

func (service *Service) AttachBrowser(coordinator *browser.Coordinator, runtimeManager *browser.RuntimeManager) {
	service.browser, service.runtime = coordinator, runtimeManager
}

func (service *Service) setStartupError(message string) {
	service.mu.Lock()
	service.startupError = message
	service.mu.Unlock()
}

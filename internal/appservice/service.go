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
	"github.com/sleaze5/RobloxAccountManager/internal/integration/rbxservers"
	"github.com/sleaze5/RobloxAccountManager/internal/integration/rovalra"
	"github.com/sleaze5/RobloxAccountManager/internal/logging"
	"github.com/sleaze5/RobloxAccountManager/internal/logsexplorer"
	"github.com/sleaze5/RobloxAccountManager/internal/platform/robloxmulti"
	"github.com/sleaze5/RobloxAccountManager/internal/roblox"
	robloxservices "github.com/sleaze5/RobloxAccountManager/internal/roblox/services"
	gamestore "github.com/sleaze5/RobloxAccountManager/internal/storage/games"
	transferstore "github.com/sleaze5/RobloxAccountManager/internal/storage/transfer"
	"github.com/sleaze5/RobloxAccountManager/internal/storage/vault"
)

type Service struct {
	vault              Vault
	accountRepository  AccountRepository
	sessions           *roblox.SessionManager
	users              *robloxservices.Users
	accountSettings    *robloxservices.AccountSettings
	chat               *robloxservices.Chat
	notificationStream *robloxservices.NotificationStream
	authentication     *robloxservices.Authentication
	thumbnails         *robloxservices.Thumbnails
	presence           *robloxservices.Presence
	gameLinks          *robloxservices.GameLinks
	games              *robloxservices.Games
	privateServers     *robloxservices.PrivateServers
	badges             *robloxservices.Badges
	rovalra            *rovalra.Client
	rbxservers         *rbxservers.Client
	favoritePlaces     *gamestore.Repository
	transfers          *transferstore.Repository
	logsExplorer       *logsexplorer.Reader
	launcher           gamelaunch.Launcher
	multiInstance      *robloxmulti.Manager
	settings           *appsettings.Store
	logs               *logging.System
	events             *Events
	logger             *slog.Logger
	browser            *browser.Coordinator
	runtime            *browser.RuntimeManager

	importMutex          sync.Mutex
	transferMutex        sync.Mutex
	pendingTransfer      *pendingTransfer
	launchMutex          sync.Mutex
	pendingImports       map[string]pendingImportBatch
	validationMutex      sync.Mutex
	validationNext       uint64
	validationCancels    map[uint64]context.CancelFunc
	accountSettingsMutex sync.Mutex
	accountSettingsLocks map[int64]*accountSettingsLock
	cookieRenewals       sync.Map
	metadataRefresh      sync.Map
	metadataSlots        chan struct{}

	mutex                  sync.Mutex
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
	notificationStream *robloxservices.NotificationStream,
	authentication *robloxservices.Authentication,
	thumbnails *robloxservices.Thumbnails,
	presence *robloxservices.Presence,
	gameLinks *robloxservices.GameLinks,
	games *robloxservices.Games,
	privateServers *robloxservices.PrivateServers,
	badges *robloxservices.Badges,
	rovalraClient *rovalra.Client,
	rbxserversClient *rbxservers.Client,
	favoritePlaces *gamestore.Repository,
	transfers *transferstore.Repository,
	logsExplorerReader *logsexplorer.Reader,
	launcher gamelaunch.Launcher,
	multiInstance *robloxmulti.Manager,
	settings *appsettings.Store,
	logs *logging.System,
	events *Events,
	logger *slog.Logger,
) *Service {
	return &Service{
		vault:                vaultManager,
		metadataSlots:        make(chan struct{}, 3),
		accountRepository:    repository,
		sessions:             sessions,
		users:                users,
		accountSettings:      accountSettings,
		accountSettingsLocks: make(map[int64]*accountSettingsLock),
		chat:                 chat,
		notificationStream:   notificationStream,
		authentication:       authentication,
		thumbnails:           thumbnails,
		presence:             presence,
		gameLinks:            gameLinks,
		games:                games,
		privateServers:       privateServers,
		badges:               badges,
		rovalra:              rovalraClient,
		rbxservers:           rbxserversClient,
		favoritePlaces:       favoritePlaces,
		transfers:            transfers,
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
	service.mutex.Lock()
	service.lifecycle = ctx
	service.mutex.Unlock()

	if err := service.vault.AutoUnlock(ctx); err == nil {
		if service.vault.FileState() == vault.FileStateIncomplete {
			service.setStartupError("The portable vault key is missing or damaged. Create a new master password to repair it.")
		} else {
			service.finishUnlock(true)
		}
	} else if !errors.Is(err, vault.ErrNotInitialized) && !errors.Is(err, vault.ErrLocked) && !errors.Is(err, vault.ErrLegacyVault) {
		service.setStartupError(vaultFailureMessage(err, "Automatic unlock failed. Enter the master password to recover the vault."))
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
	service.browser.Shutdown()
	service.multiInstance.Close()
	service.cancelPendingValidations()
	service.stopBackground()
	service.clearPendingImports()
	service.discardPendingTransfer()
	service.sessions.Lock()
	return service.vault.Close()
}

func (service *Service) beginValidation(parent context.Context) (context.Context, func()) {
	ctx, cancel := context.WithCancel(parent)
	service.validationMutex.Lock()
	service.validationNext++
	validationID := service.validationNext
	service.validationCancels[validationID] = cancel
	service.validationMutex.Unlock()
	return ctx, func() {
		cancel()
		service.validationMutex.Lock()
		delete(service.validationCancels, validationID)
		service.validationMutex.Unlock()
	}
}

func (service *Service) cancelPendingValidations() {
	service.validationMutex.Lock()
	for validationID, cancel := range service.validationCancels {
		cancel()
		delete(service.validationCancels, validationID)
	}
	service.validationMutex.Unlock()
}

func (service *Service) AttachBrowser(coordinator *browser.Coordinator, runtimeManager *browser.RuntimeManager) {
	service.browser, service.runtime = coordinator, runtimeManager
}

func (service *Service) setStartupError(message string) {
	service.mutex.Lock()
	service.startupError = message
	service.mutex.Unlock()
}

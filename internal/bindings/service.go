package bindings

import (
	"context"
	"sync"

	"github.com/sleaze5/RobloxAccountManager/internal/appmeta"
	"github.com/sleaze5/RobloxAccountManager/internal/appservice"
	"github.com/sleaze5/RobloxAccountManager/internal/appupdate"
	"github.com/sleaze5/RobloxAccountManager/internal/logging"
	"github.com/sleaze5/RobloxAccountManager/internal/roblox"
	"github.com/wailsapp/wails/v3/pkg/application"
)

type Service struct {
	core     *appservice.Service
	location *appservice.Location
	updates  *appupdate.Service
	events   *appservice.Events
	launch   *logging.Launch

	startMutex   sync.Mutex
	startContext context.Context
	started      bool
}

func NewService(core *appservice.Service, location *appservice.Location, updates *appupdate.Service, events *appservice.Events, launch *logging.Launch) *Service {
	return &Service{core: core, location: location, updates: updates, events: events, launch: launch}
}

func (service *Service) ServiceStartup(ctx context.Context, _ application.ServiceOptions) error {
	service.launch.Begin("service-startup")
	if app := application.Get(); app != nil {
		service.events.AttachWindow(
			func(name string, payload any) { app.Event.Emit(name, payload) },
			func() { showMainWindow(app) },
		)
	}
	service.startMutex.Lock()
	service.startContext = ctx
	service.startMutex.Unlock()
	if err := service.startCore(); err != nil {
		return err
	}
	if app := application.Get(); app != nil {
		service.updates.Start(ctx, app)
	}
	service.launch.WaitForWebview()
	return nil
}

func (service *Service) ServiceShutdown() error {
	service.events.DetachWindow()
	service.updates.Shutdown()
	return service.core.Shutdown()
}

func showMainWindow(app *application.App) {
	windows := app.Window.GetAll()
	if len(windows) == 0 {
		return
	}
	window := windows[0]
	if window.IsMinimised() {
		window.UnMinimise()
	}
	window.Show()
	window.Focus()
}

func (service *Service) GetLaunchReport() logging.LaunchReport { return service.launch.Report() }

func (service *Service) GetBuild() appmeta.Build { return appmeta.CurrentBuild() }

// Automatic unlock and migrations must never touch a vault that the user has
// not chosen.
func (service *Service) startCore() error {
	service.startMutex.Lock()
	defer service.startMutex.Unlock()
	if service.started || service.startContext == nil || !service.location.State().Initialized {
		return nil
	}
	if err := service.core.Start(service.startContext); err != nil {
		return err
	}
	service.started = true
	return nil
}

func accountSelection(accountIDs []int64, endpoint string) error {
	if len(accountIDs) == 0 {
		return &roblox.Error{Kind: roblox.KindProtocol, Endpoint: endpoint, Message: "Select at least one account."}
	}
	seen := make(map[int64]bool, len(accountIDs))
	for _, accountID := range accountIDs {
		if err := positiveID(accountID, endpoint); err != nil {
			return err
		}
		if seen[accountID] {
			return &roblox.Error{Kind: roblox.KindProtocol, Endpoint: endpoint, Message: "The selection has an account more than once."}
		}
		seen[accountID] = true
	}
	return nil
}

func positiveID(id int64, endpoint string) error {
	if id > 0 {
		return nil
	}
	return &roblox.Error{Kind: roblox.KindProtocol, Endpoint: endpoint, Message: "The ID is invalid."}
}

// validateAccountAndTargetIDs checks the account that sends a request and the
// user, item, or outfit that the request is about.
func validateAccountAndTargetIDs(accountID, targetID int64, endpoint string) error {
	if err := positiveID(accountID, endpoint); err != nil {
		return err
	}
	return positiveID(targetID, endpoint)
}

package main

import (
	"embed"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"time"

	"github.com/sleaze5/RobloxAccountManager/internal/appdata"
	"github.com/sleaze5/RobloxAccountManager/internal/appmeta"
	"github.com/sleaze5/RobloxAccountManager/internal/appservice"
	"github.com/sleaze5/RobloxAccountManager/internal/appsettings"
	"github.com/sleaze5/RobloxAccountManager/internal/appupdate"
	"github.com/sleaze5/RobloxAccountManager/internal/bindings"
	"github.com/sleaze5/RobloxAccountManager/internal/browser"
	"github.com/sleaze5/RobloxAccountManager/internal/gamelaunch"
	"github.com/sleaze5/RobloxAccountManager/internal/integration/rovalra"
	"github.com/sleaze5/RobloxAccountManager/internal/logging"
	"github.com/sleaze5/RobloxAccountManager/internal/logsexplorer"
	"github.com/sleaze5/RobloxAccountManager/internal/platform/protection"
	"github.com/sleaze5/RobloxAccountManager/internal/platform/robloxlogs"
	"github.com/sleaze5/RobloxAccountManager/internal/platform/robloxmulti"
	"github.com/sleaze5/RobloxAccountManager/internal/platform/singleinstance"
	"github.com/sleaze5/RobloxAccountManager/internal/roblox"
	robloxservices "github.com/sleaze5/RobloxAccountManager/internal/roblox/services"
	accountstore "github.com/sleaze5/RobloxAccountManager/internal/storage/accounts"
	gamestore "github.com/sleaze5/RobloxAccountManager/internal/storage/games"
	"github.com/sleaze5/RobloxAccountManager/internal/storage/vault"
	"github.com/wailsapp/wails/v3/pkg/application"
	wailevents "github.com/wailsapp/wails/v3/pkg/events"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	appupdate.HandleHelperMode()
	if monitor, err := logging.RunCrashMonitor(); monitor {
		if err != nil {
			logging.NewConsole(os.Stderr, logging.LevelError).Error("crash monitor failed", "module", "logging", "error", err)
			os.Exit(1)
		}
		return
	}
	if err := run(); err != nil {
		os.Exit(1)
	}
}

func run() (runErr error) {
	launch := logging.NewLaunch()
	var instance *singleinstance.Instance
	defer func() {
		var attributes []any
		if value := recover(); value != nil {
			runErr = errors.New(logging.PanicReason(value))
			attributes = []any{"stack", logging.PanicStack()}
		}
		launch.Finish(runErr, attributes...)
		_ = instance.Close()
	}()
	launch.Begin("single-instance")
	requested, err := appdata.RequestedMode(os.Args[1:])
	if err != nil {
		return err
	}
	moveTarget, err := appdata.RequestedMove(os.Args[1:])
	if err != nil {
		return err
	}
	instance, acquired, err := acquireInstance(requested != "" || moveTarget != "")
	if err != nil {
		return err
	}
	if !acquired {
		singleinstance.ShowAlreadyRunning()
		return nil
	}
	var moved bool
	var moveErr error
	if moveTarget != "" {
		launch.Begin("data-move")
		if moved, moveErr = appdata.Move(moveTarget); moved {
			requested = moveTarget
		}
	}
	launch.Begin("application-location")
	location, err := appdata.Inspect(requested)
	if err != nil {
		return fmt.Errorf("inspect application location: %w", err)
	}
	location.Moved = moved
	var moveError *appdata.MoveError
	if errors.As(moveErr, &moveError) {
		location.MoveError = moveError.Message
	}
	launch.Begin("logging")
	logSystem, err := launch.Open(location.Directory, !location.Initialized)
	if err != nil {
		return fmt.Errorf("initialize launch diagnostics: %w", err)
	}
	launch.Begin("data-paths")
	dataPaths, err := resolvePaths(location)
	if err != nil {
		return fmt.Errorf("initialize application data paths: %w", err)
	}
	launch.Begin("settings")
	logging.Diagnostic(logSystem.Module("settings"), "loading application settings", "operation", "settings-load", "supported_format_version", appsettings.FormatVersion)
	settingsStore, err := openSettings(dataPaths.Settings, location)
	if err != nil {
		return fmt.Errorf("initialize application settings: %w", err)
	}
	enabledLogLevels, err := logging.ParseLevels(settingsStore.Logging().EnabledLevelNames())
	if err != nil {
		return fmt.Errorf("initialize logging settings: %w", err)
	}
	if err := logSystem.SetEnabledLevels(enabledLogLevels); err != nil {
		return fmt.Errorf("apply logging settings: %w", err)
	}
	logging.Diagnostic(logSystem.Module("settings"), "application settings loaded and normalized", "operation", "settings-load", "format_version", appsettings.FormatVersion, "source_format_version", settingsStore.SourceVersion(), "origin", settingsStore.Origin())
	if err := settingsStore.RecoveryError(); err != nil {
		logSystem.Module("settings").Warn("settings could not be loaded; preserved backup and created defaults",
			"operation", "settings-recovery", "backup", "./storage/settings.json.bak", "error", err)
	}
	if fields := settingsStore.ReplacedFields(); len(fields) > 0 {
		logSystem.Module("settings").Warn("invalid settings replaced with defaults", "operation", "settings-load", "fields", fields)
	}
	logLocation(logSystem.Module("application.location"), location, requested, moveTarget, moveErr)
	launch.Begin("roblox-multi-instance")
	multiInstance := robloxmulti.New(logSystem.Module("platform.roblox-multi-instance"))
	defer multiInstance.Close()
	multiInstance.SetEnabled(settingsStore.Roblox().MultiInstance)

	launch.Begin("application-services")
	vaultManager := vault.NewManager(dataPaths, protection.New(), logSystem.Module("vault"))
	repository := accountstore.NewRepository(vaultManager, logSystem.Module("storage.accounts"))
	sessions := roblox.NewSessionManager()
	sessions.SetLoader(repository.GetSession)
	events := appservice.NewEvents()
	client := roblox.NewClient(roblox.ClientOptions{
		Store:    repository,
		Sessions: sessions,
		Observer: events,
		Logger:   logSystem.Module("roblox.client"),
	})
	coreService := appservice.New(
		vaultManager,
		repository,
		sessions,
		robloxservices.NewUsers(client),
		robloxservices.NewAccountSettings(client),
		robloxservices.NewChat(client),
		robloxservices.NewAuth(client),
		robloxservices.NewThumbnails(client),
		robloxservices.NewPresence(client),
		robloxservices.NewGameLinks(client),
		robloxservices.NewGames(client, logSystem.Module("roblox.games")),
		rovalra.New(),
		gamestore.NewRepository(vaultManager, logSystem.Module("storage.games")),
		logsexplorer.NewReader(robloxlogs.Directories, logSystem.Module("logs-explorer")),
		gamelaunch.New(func() string { return settingsStore.Roblox().LinuxClient }),
		multiInstance,
		settingsStore,
		logSystem,
		events,
		logSystem.Module("application.accounts"),
	)
	launch.Begin("browser-runtime")
	runtimeManager, err := browser.NewRuntimeManager(dataPaths, logSystem.Module("browser.runtime"), nil)
	if err != nil {
		return fmt.Errorf("initialize browser runtime: %w", err)
	}
	browserCoordinator := browser.NewCoordinator(dataPaths, runtimeManager, coreService, logSystem.Module("browser.coordinator"), events.BrowserChanged)
	coreService.AttachBrowser(browserCoordinator, runtimeManager)
	updates := appupdate.New(logSystem.Module("application.updates"), events.UpdateChanged)
	var app *application.App
	relaunch := func(arguments []string) error {
		executable, err := os.Executable()
		if err != nil {
			return err
		}
		command := exec.Command(executable, arguments...)
		if err := command.Start(); err != nil {
			return err
		}
		_ = command.Process.Release()
		go app.Quit()
		return nil
	}
	appLocation := appservice.NewLocation(location, dataPaths, launch, relaunch, logSystem.Module("application.location"))
	bindingService := bindings.NewService(coreService, appLocation, updates, events, launch)

	launch.Begin("desktop-shell")
	app = application.New(application.Options{
		Name:        appmeta.DisplayName,
		Description: appmeta.Description,
		Logger:      logSystem.Module("wails"),
		ErrorHandler: func(err error) {
			var fatal *application.FatalError
			if errors.As(err, &fatal) {
				launch.Error("desktop runtime fatal error", fatal.Unwrap())
				return
			}
			launch.Error("desktop runtime error", err)
		},
		PanicHandler: func(details *application.PanicDetails) {
			launch.Error("desktop runtime panic", errors.New(logging.PanicReason(details.Error)), "stack", details.StackTrace)
			os.Exit(2)
		},
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
		Services: []application.Service{
			application.NewService(bindingService),
		},
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: true,
		},
		// WebKitGTK keeps its data in $XDG_DATA_HOME/<program name>, and the
		// default program name would place it in the Standard root.
		Linux: application.LinuxOptions{
			ProgramName: appmeta.Identifier,
		},
	})

	launch.Begin("window-configuration")
	window := app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:                      appmeta.DisplayName,
		Width:                      900,
		Height:                     500,
		MinWidth:                   900,
		MinHeight:                  500,
		Frameless:                  true,
		DefaultContextMenuDisabled: true,
		BackgroundColour:           application.NewRGB(0, 0, 0),
		URL:                        "/",
		Windows: application.WindowsWindow{
			Theme: application.Dark,
			CustomTheme: application.ThemeSettings{
				DarkModeActive: &application.WindowTheme{
					BorderColour: application.NewRGBPtr(0, 0, 0),
				},
				DarkModeInactive: &application.WindowTheme{
					BorderColour: application.NewRGBPtr(0, 0, 0),
				},
			},
		},
	})
	window.OnWindowEvent(wailevents.Common.WindowRuntimeReady, func(_ *application.WindowEvent) {
		launch.Ready()
	})

	launch.Begin("desktop-runtime")
	if err := app.Run(); err != nil {
		return fmt.Errorf("run application: %w", err)
	}
	return nil
}

func logLocation(logger *slog.Logger, location appdata.Location, requested, moveTarget appdata.Mode, moveErr error) {
	state := appdata.RootEmpty
	if candidate, ok := location.Candidate(location.Mode); ok {
		state = candidate.State
	}
	attributes := []any{"operation", "location-load", "mode", location.Mode, "state", state, "confirmed", location.Initialized}
	if location.Choice != appdata.ChoiceNone {
		attributes = append(attributes, "choice", location.Choice)
	}
	if requested != "" {
		attributes = append(attributes, "requested", requested)
	}
	logging.Diagnostic(logger, "data folder resolved", attributes...)
	switch {
	case moveTarget == "":
	case moveErr != nil:
		logger.Error("data move failed", "operation", "data-move", "target", moveTarget, "moved", location.Moved, "error", moveErr)
	default:
		logger.Info("data moved", "operation", "data-move", "target", moveTarget)
	}
}

// After a storage choice, the previous process may still hold the lock.
func acquireInstance(restarted bool) (*singleinstance.Instance, bool, error) {
	deadline := time.Now().Add(15 * time.Second)
	for {
		instance, acquired, err := singleinstance.Acquire(appmeta.Identifier)
		if err != nil || acquired || !restarted || time.Now().After(deadline) {
			return instance, acquired, err
		}
		time.Sleep(200 * time.Millisecond)
	}
}

func resolvePaths(location appdata.Location) (appdata.Paths, error) {
	paths, err := appdata.At(location.Directory)
	if err != nil || !location.Initialized {
		return paths, err
	}
	return paths, paths.Prepare()
}

// appsettings.Open rewrites settings.json, so an unconfirmed root keeps
// defaults in memory and is never written.
func openSettings(path string, location appdata.Location) (*appsettings.Store, error) {
	if !location.Initialized {
		return appsettings.New(path), nil
	}
	return appsettings.Open(path)
}

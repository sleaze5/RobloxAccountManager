package logging

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"runtime"
	"sync"
	"time"

	"github.com/sleaze5/RobloxAccountManager/internal/appdata"
	"github.com/sleaze5/RobloxAccountManager/internal/appmeta"
	"github.com/sleaze5/RobloxAccountManager/internal/appsettings"
)

type LaunchStage struct {
	Name       string  `json:"name"`
	DurationMS float64 `json:"duration_ms"`
}

type Launch struct {
	mu             sync.Mutex
	id             string
	started        time.Time
	stage          string
	since          time.Time
	stages         []LaunchStage
	readyMS        float64
	components     *componentRegistry
	system         *System
	logger         *slog.Logger
	ready          bool
	timer          *time.Timer
	finished       bool
	consoleEnabled bool
}

func NewLaunch() *Launch {
	now := time.Now()
	return &Launch{
		id: fmt.Sprintf("%d-%d", now.UnixMilli(), os.Getpid()), started: now,
		logger:         slog.New(slog.DiscardHandler),
		consoleEnabled: true,
		components:     newComponentRegistry(),
	}
}

func (launch *Launch) Open(root string, hold bool) (*System, error) {
	settings, settingsErr := appsettings.ReadLogging(appdata.SettingsFile(root))
	launch.consoleEnabled = settings.EnabledLevels["error"]
	levels, err := ParseLevels(settings.EnabledLevelNames())
	if err != nil {
		return nil, err
	}
	system, err := Open(Config{Directory: appdata.LogsDirectory(root), LaunchID: launch.id, EnabledLevels: levels, AddSource: true, Hold: hold, components: launch.components})
	if err != nil {
		return nil, err
	}
	launch.mu.Lock()
	launch.system = system
	launch.logger = system.Module("startup").With("operation", "application-launch")
	launch.logger.Info("application starting", "version", appmeta.Version, "go_version", runtime.Version(),
		"os", runtime.GOOS, "arch", runtime.GOARCH, "pid", os.Getpid(),
		"started_at", launch.started.UTC(), "elapsed_ms", milliseconds(time.Since(launch.started)), "stages", launch.stages)
	launch.mu.Unlock()
	if settingsErr != nil {
		launch.logger.Warn("logging preferences could not be loaded; using defaults", "error", settingsErr)
	}
	logFile := "created"
	if len(levels) == 0 {
		logFile = "disabled"
	} else if hold {
		logFile = "held"
	}
	Diagnostic(system.Module("logging"), "launch log opened", "operation", "logging-load",
		"log_format_version", logFormatVersion, "file", logFile, "enabled_levels", len(levels))
	if !hold {
		launch.captureCrashes()
	}
	return system, nil
}

func (launch *Launch) Release(root string) error {
	if err := launch.system.Release(appdata.LogsDirectory(root)); err != nil {
		return err
	}
	launch.captureCrashes()
	return nil
}

func (launch *Launch) captureCrashes() {
	if err := launch.system.CaptureCrashes(launch.id); err != nil {
		launch.Error("could not enable runtime crash capture", err)
	}
}

func (launch *Launch) Begin(stage string) {
	launch.mu.Lock()
	defer launch.mu.Unlock()
	launch.completeStage()
	launch.stage, launch.since = stage, time.Now()
	launch.logger.Info("startup stage started", "stage", stage, "elapsed_ms", milliseconds(time.Since(launch.started)))
}

func (launch *Launch) completeStage() {
	if launch.stage == "" {
		return
	}
	stage := LaunchStage{Name: launch.stage, DurationMS: milliseconds(time.Since(launch.since))}
	launch.stages = append(launch.stages, stage)
	launch.logger.Info("startup stage completed", "stage", stage.Name, "duration_ms", stage.DurationMS,
		"elapsed_ms", milliseconds(time.Since(launch.started)))
}

func (launch *Launch) WaitForWebview() {
	launch.Begin("webview-runtime-ready")
	launch.mu.Lock()
	defer launch.mu.Unlock()
	launch.timer = time.AfterFunc(30*time.Second, func() {
		launch.mu.Lock()
		defer launch.mu.Unlock()
		if !launch.ready && !launch.finished {
			launch.logger.Warn("webview runtime has not reported ready", "stage", launch.stage,
				"elapsed_ms", milliseconds(time.Since(launch.started)), "stages", launch.stages)
		}
	})
}

func (launch *Launch) Ready() {
	launch.mu.Lock()
	defer launch.mu.Unlock()
	if launch.ready || launch.finished {
		return
	}
	launch.ready = true
	if launch.timer != nil {
		launch.timer.Stop()
	}
	launch.completeStage()
	launch.stage = "running"
	launch.since = time.Now()
	launch.readyMS = milliseconds(time.Since(launch.started))
	launch.logger.Info("application backend and webview runtime ready", "elapsed_ms", milliseconds(time.Since(launch.started)), "stages", launch.stages)
}

type LaunchReport struct {
	LaunchID   string            `json:"launchId"`
	StartedAt  int64             `json:"startedAt"`
	Ready      bool              `json:"ready"`
	ReadyMS    float64           `json:"readyMs"`
	Stages     []LaunchStage     `json:"stages"`
	Build      appmeta.Build     `json:"build"`
	Components []ComponentRecord `json:"components"`
}

func (launch *Launch) Report() LaunchReport {
	launch.mu.Lock()
	report := LaunchReport{LaunchID: launch.id, StartedAt: launch.started.UnixMilli(), Ready: launch.ready,
		ReadyMS: launch.readyMS, Stages: append([]LaunchStage{}, launch.stages...)}
	launch.mu.Unlock()
	report.Build = appmeta.CurrentBuild()
	report.Components = launch.components.snapshot()
	return report
}

func (launch *Launch) Error(message string, err error, attributes ...any) {
	launch.mu.Lock()
	defer launch.mu.Unlock()
	if launch.finished {
		return
	}
	attributes = append(attributes, "stage", launch.stage, "startup_complete", launch.ready,
		"elapsed_ms", milliseconds(time.Since(launch.started)), "stage_duration_ms", milliseconds(time.Since(launch.since)),
		"version", appmeta.Version, "stages", launch.stages, "error", err)
	launch.logger.Error(message, attributes...)
	if launch.system != nil {
		if err := launch.system.Sync(); err != nil {
			launch.consoleError("could not flush application log", err)
		}
	}
}

func (launch *Launch) Finish(err error, attributes ...any) {
	if launch.system == nil && err != nil {
		root, openErr := appdata.FallbackRoot()
		if openErr == nil {
			_, openErr = launch.Open(root, false)
		}
		if openErr != nil && launch.system == nil {
			if launch.consoleEnabled {
				NewConsole(os.Stderr, LevelError).Error("startup diagnostics unavailable", "module", "startup",
					"launch_id", launch.id, "stage", launch.stage, "error", errors.Join(err, openErr))
			}
			return
		}
	}
	if err != nil {
		launch.Error("application failed", err, attributes...)
	}
	launch.mu.Lock()
	defer launch.mu.Unlock()
	launch.finished = true
	if launch.timer != nil {
		launch.timer.Stop()
	}
	if launch.system != nil {
		if err == nil {
			launch.logger.Info("application stopped", "startup_complete", launch.ready, "uptime_ms", milliseconds(time.Since(launch.started)))
		}
		if closeErr := launch.system.Close(); closeErr != nil {
			launch.consoleError("could not close application log", closeErr)
		}
	}
}

func (launch *Launch) consoleError(message string, err error) {
	if launch.system.filter.includes(slog.LevelError) {
		NewConsole(os.Stderr, LevelError).Error(message, "module", "startup", "launch_id", launch.id, "error", err)
	}
}

func milliseconds(duration time.Duration) float64 {
	return float64(duration) / float64(time.Millisecond)
}

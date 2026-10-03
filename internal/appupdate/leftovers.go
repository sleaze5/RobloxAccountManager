package appupdate

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// helperLogVariable passes the update helper's log path to the new version.
// The helper copies this process environment into the version it starts.
const helperLogVariable = "ROBLOXACCOUNTMANAGER_UPDATE_HELPER_LOG"

const leftoverAttempts = 30

// helperLogPath returns the log path that updater.Restart gives the helper.
// It must match the name the Wails updater builds from the process ID.
func helperLogPath() string {
	return filepath.Join(os.TempDir(), fmt.Sprintf("wails-update-%d.log", os.Getpid()))
}

// removeLeftovers deletes files the Wails update helper leaves behind. On
// Windows the helper renames the running executable to "<executable>.old.*"
// and creates its log in the temporary directory. The helper can still hold
// these files open while the new version starts, so removal is retried.
func (service *Service) removeLeftovers() {
	var paths []string
	if executable, err := os.Executable(); err == nil {
		paths, _ = filepath.Glob(executable + ".old.*")
	}
	if log := os.Getenv(helperLogVariable); log != "" {
		_ = os.Unsetenv(helperLogVariable)
		if isHelperLog(log) {
			paths = append(paths, log)
		}
	}
	for attempt := 1; len(paths) > 0; attempt++ {
		last := attempt == leftoverAttempts
		paths = service.removeAll(paths, last)
		if last {
			break
		}
		if len(paths) > 0 {
			time.Sleep(time.Second)
		}
	}
}

// isHelperLog accepts only a helper log in the temporary directory, so the
// inherited variable cannot name any other file.
func isHelperLog(path string) bool {
	name := filepath.Base(path)
	return filepath.Dir(path) == filepath.Clean(os.TempDir()) &&
		strings.HasPrefix(name, "wails-update-") && strings.HasSuffix(name, ".log")
}

// removeAll removes each path and returns the paths that are still present.
func (service *Service) removeAll(paths []string, last bool) []string {
	var remaining []string
	for _, path := range paths {
		err := os.Remove(path)
		switch {
		case err == nil || os.IsNotExist(err):
			service.logger.Debug("update leftover removed", "operation", "update-cleanup", "path", path)
		case last:
			service.logger.Warn("update leftover could not be removed", "operation", "update-cleanup", "path", path, "error", err)
		default:
			remaining = append(remaining, path)
		}
	}
	return remaining
}

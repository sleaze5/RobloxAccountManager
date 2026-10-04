package appupdate

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const helperLogVariable = "ROBLOXACCOUNTMANAGER_UPDATE_HELPER_LOG"

const leftoverAttempts = 30

func helperLogPath() string {
	return filepath.Join(os.TempDir(), fmt.Sprintf("wails-update-%d.log", os.Getpid()))
}

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

func isHelperLog(path string) bool {
	name := filepath.Base(path)
	return filepath.Dir(path) == filepath.Clean(os.TempDir()) &&
		strings.HasPrefix(name, "wails-update-") && strings.HasSuffix(name, ".log")
}

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

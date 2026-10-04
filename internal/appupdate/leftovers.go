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

// Wails names its staging directories and helper logs with this prefix, and its
// helper removes a staging directory with it after a successful swap.
const wailsUpdatePrefix = "wails-update-"

const stagingPrefix = wailsUpdatePrefix + "robloxaccountmanager-"

func helperLogPath() string {
	return filepath.Join(os.TempDir(), fmt.Sprintf(wailsUpdatePrefix+"%d.log", os.Getpid()))
}

func (service *Service) removeLeftovers() {
	var paths []string
	if executable, err := os.Executable(); err == nil {
		paths, _ = filepath.Glob(executable + ".old.*")
		service.removeStaging(filepath.Dir(executable))
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

func (service *Service) removeStaging(directory string) {
	staged, _ := filepath.Glob(filepath.Join(directory, stagingPrefix+"*"))
	for _, path := range staged {
		if info, err := os.Lstat(path); err != nil || !info.IsDir() {
			continue
		}
		if err := os.RemoveAll(path); err != nil {
			service.logger.Warn("update staging could not be removed", "operation", "update-cleanup", "path", path, "error", err)
		}
	}
}

func isHelperLog(path string) bool {
	name := filepath.Base(path)
	return filepath.Dir(path) == filepath.Clean(os.TempDir()) &&
		strings.HasPrefix(name, wailsUpdatePrefix) && strings.HasSuffix(name, ".log")
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

//go:build !windows

package appupdate

import (
	"os"
	"path/filepath"
)

// The Wails helper renames the staged update over the installed application,
// which fails when the temporary folder is on another filesystem. Copy the
// update next to the installed application first.
func stageHelperArtifact() {
	if os.Getenv("WAILS_UPDATER_HELPER") != "1" {
		return
	}
	target, staged := os.Getenv("WAILS_UPDATER_HELPER_TARGET"), os.Getenv("WAILS_UPDATER_HELPER_NEW")
	if target == "" || staged == "" {
		return
	}
	directory, err := os.MkdirTemp(filepath.Dir(target), stagingPrefix+"*")
	if err != nil {
		return
	}
	local := filepath.Join(directory, filepath.Base(staged))
	if err := copyStaged(staged, local); err != nil {
		_ = os.RemoveAll(directory)
		return
	}
	_ = os.Setenv("WAILS_UPDATER_HELPER_NEW", local)
}

// helperCopiedUpdate reports whether the helper owns a copy of the update, so
// the application can remove the original download when it quits.
func helperCopiedUpdate() bool {
	executable, err := os.Executable()
	if err != nil {
		return false
	}
	staged, _ := filepath.Glob(filepath.Join(filepath.Dir(installTarget(executable)), stagingPrefix+"*"))
	return len(staged) > 0
}

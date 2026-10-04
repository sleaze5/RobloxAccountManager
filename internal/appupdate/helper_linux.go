//go:build linux

package appupdate

import (
	"io"
	"os"
	"path/filepath"
	"strings"
)

// The Wails helper renames the staged update over the executable, which fails
// when the temporary directory is on another filesystem. Move the update next
// to the executable first.
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
	if err := copyFile(staged, local); err != nil {
		_ = os.RemoveAll(directory)
		return
	}
	_ = os.Setenv("WAILS_UPDATER_HELPER_NEW", local)
	if source := filepath.Dir(staged); strings.HasPrefix(filepath.Base(source), stagingPrefix) {
		_ = os.RemoveAll(source)
	}
}

func copyFile(source, destination string) error {
	input, err := os.Open(source)
	if err != nil {
		return err
	}
	defer input.Close()
	output, err := os.OpenFile(destination, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o700)
	if err != nil {
		return err
	}
	if _, err := io.Copy(output, input); err != nil {
		output.Close()
		return err
	}
	if err := output.Sync(); err != nil {
		output.Close()
		return err
	}
	return output.Close()
}

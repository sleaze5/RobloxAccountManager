//go:build darwin

package appupdate

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// The Wails helper renames the extracted application bundle over the
// installed one, which fails when the temporary folder is on another volume.
// Copy the bundle next to the installed one first.
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
	if err := copyTree(staged, local); err != nil {
		_ = os.RemoveAll(directory)
		return
	}
	_ = os.Setenv("WAILS_UPDATER_HELPER_NEW", local)
}

func helperCopiedUpdate() bool {
	executable, err := os.Executable()
	if err != nil {
		return false
	}
	staged, _ := filepath.Glob(filepath.Join(filepath.Dir(installTarget(executable)), stagingPrefix+"*"))
	return len(staged) > 0
}

func installTarget(executable string) string {
	for directory := filepath.Dir(executable); directory != filepath.Dir(directory); directory = filepath.Dir(directory) {
		if strings.HasSuffix(directory, ".app") {
			return directory
		}
	}
	return executable
}

func copyTree(source, destination string) error {
	err := filepath.WalkDir(source, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		relative, err := filepath.Rel(source, path)
		if err != nil {
			return fmt.Errorf("resolve update path: %w", err)
		}
		target := filepath.Join(destination, relative)
		info, err := entry.Info()
		if err != nil {
			return fmt.Errorf("inspect update file: %w", err)
		}
		switch {
		case entry.Type()&fs.ModeSymlink != 0:
			link, err := os.Readlink(path)
			if err != nil {
				return fmt.Errorf("read update link: %w", err)
			}
			if filepath.IsAbs(link) {
				return errors.New("update contains an absolute link")
			}
			if err := os.Symlink(link, target); err != nil {
				return fmt.Errorf("copy update link: %w", err)
			}
			return nil
		case entry.IsDir():
			// Keep the bundle's permissions, and let the owner read, write, and enter every directory.
			if err := os.Mkdir(target, info.Mode().Perm()|0o700); err != nil {
				return fmt.Errorf("create update directory: %w", err)
			}
			return nil
		case info.Mode().IsRegular():
			return copyFile(path, target, info.Mode().Perm())
		default:
			return errors.New("update contains an unsupported file")
		}
	})
	if err != nil {
		return fmt.Errorf("copy update bundle: %w", err)
	}
	return nil
}

func copyFile(source, destination string, mode fs.FileMode) error {
	input, err := os.Open(source)
	if err != nil {
		return fmt.Errorf("open staged update: %w", err)
	}
	defer func() { _ = input.Close() }()
	output, err := os.OpenFile(destination, os.O_CREATE|os.O_EXCL|os.O_WRONLY, mode)
	if err != nil {
		return fmt.Errorf("create update copy: %w", err)
	}
	if _, err := io.Copy(output, input); err != nil {
		_ = output.Close()
		return fmt.Errorf("copy update: %w", err)
	}
	if err := output.Sync(); err != nil {
		_ = output.Close()
		return fmt.Errorf("flush update copy: %w", err)
	}
	if err := output.Close(); err != nil {
		return fmt.Errorf("close update copy: %w", err)
	}
	return nil
}

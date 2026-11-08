//go:build darwin

package appupdate

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

func installTarget(executable string) string {
	for directory := filepath.Dir(executable); directory != filepath.Dir(directory); directory = filepath.Dir(directory) {
		if strings.HasSuffix(directory, ".app") {
			return directory
		}
	}
	return executable
}

func copyStaged(source, destination string) error {
	sourceRoot, err := os.OpenRoot(source)
	if err != nil {
		return fmt.Errorf("open staged update bundle: %w", err)
	}
	defer func() { _ = sourceRoot.Close() }()
	bundleStatus, err := sourceRoot.Lstat(".")
	if err != nil {
		return fmt.Errorf("inspect update bundle: %w", err)
	}
	destinationRoot, err := createCopyRoot(destination, bundleStatus.Mode().Perm()|0o700)
	if err != nil {
		return err
	}
	defer func() { _ = destinationRoot.Close() }()
	err = fs.WalkDir(sourceRoot.FS(), ".", func(path string, _ fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if path == "." {
			return nil
		}
		return copyTreeEntry(sourceRoot, destinationRoot, path)
	})
	if err != nil {
		return fmt.Errorf("copy update bundle: %w", err)
	}
	return nil
}

func createCopyRoot(destination string, mode fs.FileMode) (*os.Root, error) {
	parent, err := os.OpenRoot(filepath.Dir(destination))
	if err != nil {
		return nil, fmt.Errorf("open update destination: %w", err)
	}
	defer func() { _ = parent.Close() }()
	name := filepath.Base(destination)
	if err := parent.Mkdir(name, mode); err != nil {
		return nil, fmt.Errorf("create update directory: %w", err)
	}
	root, err := parent.OpenRoot(name)
	if err != nil {
		return nil, fmt.Errorf("open update copy: %w", err)
	}
	return root, nil
}

func copyTreeEntry(source, destination *os.Root, path string) error {
	entryStatus, err := source.Lstat(path)
	if err != nil {
		return fmt.Errorf("inspect update file: %w", err)
	}
	switch {
	case entryStatus.Mode()&fs.ModeSymlink != 0:
		link, err := source.Readlink(path)
		if err != nil {
			return fmt.Errorf("read update link: %w", err)
		}
		if filepath.IsAbs(link) {
			return errors.New("update contains an absolute link")
		}
		if err := destination.Symlink(link, path); err != nil {
			return fmt.Errorf("copy update link: %w", err)
		}
		return nil
	case entryStatus.IsDir():
		// Keep the bundle's permissions, and let the owner read, write, and enter every directory.
		if err := destination.Mkdir(path, entryStatus.Mode().Perm()|0o700); err != nil {
			return fmt.Errorf("create update directory: %w", err)
		}
		return nil
	case entryStatus.Mode().IsRegular():
		return copyFile(source, destination, path, entryStatus.Mode().Perm())
	default:
		return errors.New("update contains an unsupported file")
	}
}

func copyFile(source, destination *os.Root, path string, mode fs.FileMode) error {
	input, err := source.OpenFile(path, os.O_RDONLY, 0)
	if err != nil {
		return fmt.Errorf("open staged update: %w", err)
	}
	defer func() { _ = input.Close() }()
	output, err := destination.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, mode)
	if err != nil {
		return fmt.Errorf("create update copy: %w", err)
	}
	return writeUpdateCopy(output, input)
}

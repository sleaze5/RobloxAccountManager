package appdata

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
)

// MoveError is shown to the user as is.
type MoveError struct {
	Message string
	Err     error
}

func (err *MoveError) Error() string { return err.Message }

func (err *MoveError) Unwrap() error { return err.Err }

const notMoved = "The data could not be moved. It stays in the current folder."

func moveError(message string, err error) error {
	return &MoveError{Message: message, Err: err}
}

func CheckMove(target Mode) error {
	_, _, err := moveRoots(target)
	return err
}

// Move must run before anything opens files in either root. An error with
// moved set means only the old copy could not be removed.
func Move(target Mode) (moved bool, err error) {
	source, destination, err := moveRoots(target)
	if err != nil {
		return false, err
	}
	if err := copyStorage(source, destination); err != nil {
		return false, err
	}
	if err := removeStorage(source); err != nil {
		return true, moveError("The data moved, but the old copy could not be removed. Delete it to stop the app from asking which data to use.", err)
	}
	return true, nil
}

func moveRoots(target Mode) (Paths, Paths, error) {
	if _, err := ParseMode(string(target)); err != nil {
		return Paths{}, Paths{}, moveError("This storage option is unavailable.", err)
	}
	executableDirectory, err := ExecutableDirectory()
	if err != nil {
		return Paths{}, Paths{}, moveError("The app folder could not be found.", err)
	}
	portable, standard, err := candidateDirectories(executableDirectory)
	if err != nil || portable == "" {
		return Paths{}, Paths{}, moveError("This storage option is unavailable.", err)
	}
	sourceRoot, destinationRoot := standard, portable
	if target == ModeStandard {
		sourceRoot, destinationRoot = portable, standard
	}
	if state, err := inspectRoot(sourceRoot); err != nil || state != RootVault {
		return Paths{}, Paths{}, moveError("Only a complete vault can be moved.", err)
	}
	destination, err := At(destinationRoot)
	if err != nil {
		return Paths{}, Paths{}, moveError("The storage folder cannot be on a network drive.", err)
	}
	if state, err := inspectRoot(destinationRoot); err != nil || hasVaultData(state) || hasEntries(destination.VaultRoot) {
		return Paths{}, Paths{}, moveError("The other folder already has vault files. Move or delete them first.", err)
	}
	return layout(sourceRoot), destination, nil
}

// Staging means an interrupted move never leaves a partial vault.
func copyStorage(source, destination Paths) error {
	storageRoot := filepath.Dir(destination.VaultRoot)
	if err := PreparePrivateDirectory(storageRoot); err != nil {
		return moveError("The storage folder could not be created.", err)
	}
	staging, err := os.MkdirTemp(storageRoot, ".move-")
	if err != nil {
		return moveError("The storage folder could not be created.", err)
	}
	defer os.RemoveAll(staging)
	if err := restrictDirectory(staging); err != nil {
		return moveError("The storage folder could not be created.", err)
	}
	stagedSettings := filepath.Join(staging, filepath.Base(source.Settings))
	stagedVault := filepath.Join(staging, filepath.Base(source.VaultRoot))
	stagedBackups := filepath.Join(staging, filepath.Base(source.BackupsRoot))
	for _, item := range []struct{ from, to string }{
		{source.Settings, stagedSettings},
		{source.VaultRoot, stagedVault},
		{source.BackupsRoot, stagedBackups},
	} {
		if err := copyVerified(item.from, item.to); err != nil && !errors.Is(err, fs.ErrNotExist) {
			return moveError(notMoved, err)
		}
	}
	if err := commitBackups(stagedBackups, destination.BackupsRoot); err != nil {
		return moveError(notMoved, err)
	}
	if regularFile(stagedSettings) {
		if err := ReplaceFile(stagedSettings, destination.Settings); err != nil {
			return moveError(notMoved, err)
		}
	}
	if err := os.Remove(destination.VaultRoot); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return moveError(notMoved, err)
	}
	if err := os.Rename(stagedVault, destination.VaultRoot); err != nil {
		return moveError(notMoved, err)
	}
	return nil
}

func commitBackups(staged, destination string) error {
	entries, err := os.ReadDir(staged)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if err := PreparePrivateDirectory(destination); err != nil {
		return err
	}
	for _, entry := range entries {
		target := filepath.Join(destination, entry.Name())
		if _, err := os.Lstat(target); err == nil {
			continue
		}
		if err := os.Rename(filepath.Join(staged, entry.Name()), target); err != nil {
			return err
		}
	}
	return nil
}

// The vault goes first, so a later failure never leaves a vault in both roots.
func removeStorage(source Paths) error {
	if err := os.RemoveAll(source.VaultRoot); err != nil {
		return err
	}
	var err error
	for _, path := range []string{source.Settings, source.BackupsRoot, filepath.Dir(source.CfTRuntimeRoot), filepath.Dir(source.CfTInstallRoot)} {
		err = errors.Join(err, os.RemoveAll(path))
	}
	_ = os.Remove(filepath.Dir(source.VaultRoot))
	return err
}

func copyVerified(source, destination string) error {
	info, err := os.Lstat(source)
	if err != nil {
		return err
	}
	switch {
	case info.IsDir():
		if err := os.Mkdir(destination, 0o700); err != nil {
			return err
		}
		if err := restrictDirectory(destination); err != nil {
			return err
		}
		entries, err := os.ReadDir(source)
		if err != nil {
			return err
		}
		for _, entry := range entries {
			if err := copyVerified(filepath.Join(source, entry.Name()), filepath.Join(destination, entry.Name())); err != nil {
				return err
			}
		}
		return nil
	case info.Mode().IsRegular():
		return copyFile(source, destination)
	default:
		return fmt.Errorf("%q is not a regular file", filepath.Base(source))
	}
}

func copyFile(source, destination string) error {
	input, err := os.Open(source)
	if err != nil {
		return err
	}
	defer input.Close()
	output, err := os.OpenFile(destination, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return err
	}
	expected := sha256.New()
	_, copyErr := io.Copy(output, io.TeeReader(input, expected))
	syncErr := output.Sync()
	closeErr := output.Close()
	if err := errors.Join(copyErr, syncErr, closeErr); err != nil {
		return err
	}
	if err := RestrictFile(destination); err != nil {
		return err
	}
	actual, err := fileHash(destination)
	if err != nil {
		return err
	}
	if !bytes.Equal(actual, expected.Sum(nil)) {
		return fmt.Errorf("copy of %q does not match", filepath.Base(source))
	}
	return nil
}

func fileHash(path string) ([]byte, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return nil, err
	}
	return hash.Sum(nil), nil
}

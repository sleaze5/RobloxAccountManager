package appdata

import (
	"fmt"
	"os"
	"path/filepath"
)

const (
	storageDirectory = "storage"
	logsDirectory    = "logs"
)

type Paths struct {
	VaultRoot       string
	LogsRoot        string
	Settings        string
	Vault           string
	VaultKey        string
	AutoUnlockKey   string
	BackupsRoot     string
	RecoveryRoot    string
	CreateStaging   string
	CfTRuntimeRoot  string
	CfTInstallRoot  string
	BrowserTempRoot string
}

func ExecutableDirectory() (string, error) {
	executable, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("resolve executable path: %w", err)
	}
	executable, err = filepath.Abs(executable)
	if err != nil {
		return "", fmt.Errorf("resolve absolute executable path: %w", err)
	}
	return filepath.Dir(executable), nil
}

func LogsDirectory(root string) string {
	return filepath.Join(root, logsDirectory)
}

func SettingsFile(root string) string {
	return filepath.Join(root, storageDirectory, "settings.json")
}

func At(dataRoot string) (Paths, error) {
	root, err := filepath.Abs(dataRoot)
	if err != nil {
		return Paths{}, fmt.Errorf("resolve data folder: %w", err)
	}
	if err := rejectNetworkPath(existingAncestor(root)); err != nil {
		return Paths{}, err
	}
	return layout(root), nil
}

func layout(root string) Paths {
	storageRoot := filepath.Join(root, storageDirectory)
	vaultRoot := filepath.Join(storageRoot, "vault")
	return Paths{
		VaultRoot:       vaultRoot,
		LogsRoot:        LogsDirectory(root),
		Settings:        SettingsFile(root),
		Vault:           filepath.Join(vaultRoot, "vault.db"),
		VaultKey:        filepath.Join(vaultRoot, "vault.key"),
		AutoUnlockKey:   filepath.Join(vaultRoot, "autounlock.key"),
		BackupsRoot:     filepath.Join(storageRoot, "backups"),
		RecoveryRoot:    filepath.Join(vaultRoot, "recovery"),
		CreateStaging:   filepath.Join(vaultRoot, ".vault-create"),
		CfTRuntimeRoot:  filepath.Join(storageRoot, "runtime", "cft"),
		CfTInstallRoot:  filepath.Join(storageRoot, "temp", "cft-install"),
		BrowserTempRoot: filepath.Join(storageRoot, "temp", "browser"),
	}
}

// A Standard root may not exist yet, so check the filesystem of its closest
// existing parent.
func existingAncestor(path string) string {
	for {
		if _, err := os.Stat(path); err == nil {
			return path
		}
		parent := filepath.Dir(path)
		if parent == path {
			return path
		}
		path = parent
	}
}

// Prepare must run only for a confirmed location.
func (paths Paths) Prepare() error {
	return PreparePrivateDirectory(paths.VaultRoot)
}

func PreparePrivateDirectory(directory string) error {
	if err := os.MkdirAll(directory, 0o700); err != nil {
		return fmt.Errorf("create application directory %q: %w", filepath.Base(directory), err)
	}
	return restrictDirectory(directory)
}

func RestrictFile(path string) error {
	if err := os.Chmod(path, 0o600); err != nil {
		return fmt.Errorf("restrict file permissions: %w", err)
	}
	return restrictACL(path, false)
}

func WritePrivateFile(path string, data []byte) error {
	temporary := path + ".tmp"
	file, err := os.OpenFile(temporary, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return fmt.Errorf("create %q: %w", filepath.Base(path), err)
	}
	ok := false
	defer func() {
		_ = file.Close()
		if !ok {
			_ = os.Remove(temporary)
		}
	}()
	if err := RestrictFile(temporary); err != nil {
		return err
	}
	if _, err := file.Write(data); err != nil {
		return fmt.Errorf("write %q: %w", filepath.Base(path), err)
	}
	if err := file.Sync(); err != nil {
		return fmt.Errorf("sync %q: %w", filepath.Base(path), err)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("close %q: %w", filepath.Base(path), err)
	}
	if err := ReplaceFile(temporary, path); err != nil {
		return err
	}
	ok = true
	return RestrictFile(path)
}

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

func Resolve() (Paths, error) {
	executable, err := os.Executable()
	if err != nil {
		return Paths{}, fmt.Errorf("resolve executable path: %w", err)
	}
	return resolveFromExecutable(executable)
}

func ExecutableDirectory() (string, error) {
	executable, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("resolve executable path: %w", err)
	}
	return filepath.Dir(executable), nil
}

func resolveFromExecutable(executable string) (Paths, error) {
	absoluteExecutable, err := filepath.Abs(executable)
	if err != nil {
		return Paths{}, fmt.Errorf("resolve absolute executable path: %w", err)
	}
	applicationRoot := filepath.Dir(absoluteExecutable)
	if err := os.Chdir(applicationRoot); err != nil {
		return Paths{}, fmt.Errorf("set executable working directory: %w", err)
	}
	return At(applicationRoot)
}

func At(applicationDirectory string) (Paths, error) {
	applicationRoot, err := filepath.Abs(applicationDirectory)
	if err != nil {
		return Paths{}, fmt.Errorf("resolve application directory: %w", err)
	}
	if err := rejectNetworkPath(applicationRoot); err != nil {
		return Paths{}, err
	}
	storageRoot := filepath.Join(applicationRoot, storageDirectory)
	vaultRoot := filepath.Join(storageRoot, "vault")
	logsRoot := filepath.Join(applicationRoot, logsDirectory)
	if err := PreparePrivateDirectory(vaultRoot); err != nil {
		return Paths{}, err
	}
	return Paths{
		VaultRoot:       vaultRoot,
		LogsRoot:        logsRoot,
		Settings:        filepath.Join(storageRoot, "settings.json"),
		Vault:           filepath.Join(vaultRoot, "vault.db"),
		VaultKey:        filepath.Join(vaultRoot, "vault.key"),
		AutoUnlockKey:   filepath.Join(vaultRoot, "autounlock.key"),
		BackupsRoot:     filepath.Join(storageRoot, "backups"),
		RecoveryRoot:    filepath.Join(vaultRoot, "recovery"),
		CreateStaging:   filepath.Join(vaultRoot, ".vault-create"),
		CfTRuntimeRoot:  filepath.Join(storageRoot, "runtime", "cft"),
		CfTInstallRoot:  filepath.Join(storageRoot, "temp", "cft-install"),
		BrowserTempRoot: filepath.Join(storageRoot, "temp", "browser"),
	}, nil
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

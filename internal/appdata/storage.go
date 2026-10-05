package appdata

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const maxListedItems = 3

// Mode names where the data root lives. Portable keeps it beside the
// executable. Standard keeps it in the per-user application data folder of the
// operating system.
type Mode string

const (
	ModePortable Mode = "portable"
	ModeStandard Mode = "standard"
)

// RootState describes what a candidate data root already holds. It is
// detected from the existing folder layout and vault files.
type RootState string

const (
	// RootEmpty has no storage folder.
	RootEmpty RootState = "empty"
	// RootStructure has a storage folder but no vault files. Settings, logs,
	// backups, and runtime data can be re-created, so they do not count as a
	// vault.
	RootStructure RootState = "structure"
	// RootIncomplete has some vault files, but not a complete vault.
	RootIncomplete RootState = "incomplete"
	// RootVault has vault.db and vault.key and no unfinished vault creation.
	RootVault RootState = "vault"
)

// ChoiceReason says why the user must choose a data root before the
// application can continue. An empty reason means no choice is needed.
type ChoiceReason string

const (
	ChoiceNone ChoiceReason = ""
	// ChoiceFirstRun means that no data root exists yet.
	ChoiceFirstRun ChoiceReason = "first-run"
	// ChoicePortableWithoutVault means that the Portable root has data but
	// no usable vault.
	ChoicePortableWithoutVault ChoiceReason = "portable-without-vault"
	// ChoiceConflict means that both roots hold vault data.
	ChoiceConflict ChoiceReason = "conflict"
)

// Candidate is one possible data root. OtherItems lists at most
// maxListedItems names of unrelated entries in an empty Portable folder.
// OtherItemCount counts every such entry.
type Candidate struct {
	Mode           Mode      `json:"mode"`
	Directory      string    `json:"directory"`
	State          RootState `json:"state"`
	OtherItems     []string  `json:"otherItems"`
	OtherItemCount int       `json:"otherItemCount"`
}

// Location is the resolved data root. Until Initialized is true, Mode and
// Directory name the provisional root that the running process uses, and
// Choice says why the user must confirm a root. Candidates lists the roots
// that this platform offers.
type Location struct {
	Mode        Mode         `json:"mode"`
	Directory   string       `json:"directory"`
	Initialized bool         `json:"initialized"`
	Choice      ChoiceReason `json:"choice"`
	Candidates  []Candidate  `json:"candidates"`
}

// Candidate returns the candidate for mode.
func (location Location) Candidate(mode Mode) (Candidate, bool) {
	for _, candidate := range location.Candidates {
		if candidate.Mode == mode {
			return candidate, true
		}
	}
	return Candidate{}, false
}

// ParseMode accepts only the storage modes that this platform supports.
func ParseMode(value string) (Mode, error) {
	switch mode := Mode(value); mode {
	case ModeStandard:
		return mode, nil
	case ModePortable:
		if portableSupported {
			return mode, nil
		}
	}
	return "", fmt.Errorf("storage mode %q is not supported", value)
}

// Inspect finds the data root. A non-empty requested mode selects that root
// without asking, which is how the application restarts after the user
// chooses a root. Inspect also makes the executable folder the working
// directory, so no code depends on where the application was started from.
func Inspect(requested Mode) (Location, error) {
	executableDirectory, err := ExecutableDirectory()
	if err != nil {
		return Location{}, err
	}
	if err := os.Chdir(executableDirectory); err != nil {
		return Location{}, fmt.Errorf("set executable working directory: %w", err)
	}
	standardDirectory, standardErr := StandardRoot()
	var portable, standard *Candidate
	if standardErr == nil {
		candidate, err := inspectCandidate(ModeStandard, standardDirectory)
		if err != nil {
			return Location{}, err
		}
		standard = &candidate
	}
	if portableSupported && (standard == nil || !sameDirectory(executableDirectory, standardDirectory)) {
		candidate, err := inspectCandidate(ModePortable, executableDirectory)
		if err != nil {
			return Location{}, err
		}
		portable = &candidate
	}
	if portable == nil && standard == nil {
		return Location{}, fmt.Errorf("resolve standard data folder: %w", standardErr)
	}
	return resolve(portable, standard, requested)
}

func resolve(portable, standard *Candidate, requested Mode) (Location, error) {
	location := Location{Candidates: []Candidate{}}
	for _, candidate := range []*Candidate{portable, standard} {
		if candidate != nil {
			location.Candidates = append(location.Candidates, *candidate)
		}
	}
	use := func(candidate *Candidate, initialized bool, choice ChoiceReason) (Location, error) {
		location.Mode, location.Directory = candidate.Mode, candidate.Directory
		location.Initialized, location.Choice = initialized, choice
		return location, nil
	}
	if requested != "" {
		for _, candidate := range []*Candidate{portable, standard} {
			if candidate != nil && candidate.Mode == requested {
				return use(candidate, true, ChoiceNone)
			}
		}
		return Location{}, fmt.Errorf("storage mode %q is unavailable", requested)
	}
	if portable == nil {
		// macOS, or a Portable root that is the Standard root.
		return use(standard, true, ChoiceNone)
	}
	if standard == nil {
		return use(portable, portable.State != RootEmpty, choiceFor(portable.State == RootEmpty, ChoiceFirstRun))
	}
	portableVault, standardVault := hasVaultData(portable.State), hasVaultData(standard.State)
	switch {
	case portableVault && standardVault:
		return use(portable, false, ChoiceConflict)
	case portable.State == RootVault:
		return use(portable, true, ChoiceNone)
	case standardVault:
		return use(standard, true, ChoiceNone)
	case portable.State != RootEmpty:
		return use(portable, false, ChoicePortableWithoutVault)
	case standard.State != RootEmpty:
		return use(standard, true, ChoiceNone)
	default:
		return use(portable, false, ChoiceFirstRun)
	}
}

func choiceFor(needed bool, reason ChoiceReason) ChoiceReason {
	if needed {
		return reason
	}
	return ChoiceNone
}

func hasVaultData(state RootState) bool {
	return state == RootVault || state == RootIncomplete
}

func inspectCandidate(mode Mode, directory string) (Candidate, error) {
	candidate := Candidate{Mode: mode, Directory: directory, OtherItems: []string{}}
	state, err := inspectRoot(directory)
	if err != nil {
		return Candidate{}, fmt.Errorf("inspect %s data folder: %w", mode, err)
	}
	candidate.State = state
	if mode != ModePortable || state != RootEmpty {
		return candidate, nil
	}
	executable, err := os.Executable()
	if err != nil {
		return Candidate{}, fmt.Errorf("resolve executable path: %w", err)
	}
	entries, err := os.ReadDir(directory)
	if err != nil {
		return Candidate{}, fmt.Errorf("list application directory: %w", err)
	}
	for _, entry := range entries {
		if isExecutable(executable, directory, entry) {
			continue
		}
		candidate.OtherItemCount++
		if len(candidate.OtherItems) < maxListedItems {
			candidate.OtherItems = append(candidate.OtherItems, entry.Name())
		}
	}
	return candidate, nil
}

// inspectRoot mirrors the vault file states of internal/storage/vault, so a
// root whose vault is incomplete is kept for recovery instead of being
// treated as unused.
func inspectRoot(root string) (RootState, error) {
	paths := layout(root)
	info, err := os.Stat(filepath.Join(root, storageDirectory))
	if errors.Is(err, os.ErrNotExist) {
		return RootEmpty, nil
	}
	if err != nil {
		return "", err
	}
	if !info.IsDir() {
		return "", fmt.Errorf("%q is not a folder", filepath.Join(root, storageDirectory))
	}
	if !hasKnownStorage(paths) {
		return RootEmpty, nil
	}
	database := regularFile(paths.Vault)
	key := regularFile(paths.VaultKey)
	automatic := regularFile(paths.AutoUnlockKey)
	staged := hasEntries(paths.CreateStaging)
	switch {
	case database && key && !staged:
		return RootVault, nil
	case database || key || automatic || staged:
		return RootIncomplete, nil
	default:
		return RootStructure, nil
	}
}

// hasKnownStorage ignores a storage folder that holds none of this
// application's entries. Earlier Linux versions let WebKitGTK write its own
// storage folder into the Standard root.
func hasKnownStorage(paths Paths) bool {
	for _, path := range []string{paths.Settings, paths.VaultRoot, paths.BackupsRoot, filepath.Dir(paths.CfTRuntimeRoot), filepath.Dir(paths.CfTInstallRoot)} {
		if _, err := os.Lstat(path); err == nil {
			return true
		}
	}
	return false
}

func regularFile(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.Mode().IsRegular()
}

func hasEntries(directory string) bool {
	entries, err := os.ReadDir(directory)
	return err == nil && len(entries) > 0
}

func isExecutable(executable, directory string, entry os.DirEntry) bool {
	if !strings.EqualFold(entry.Name(), filepath.Base(executable)) {
		return false
	}
	executableInfo, err := os.Stat(executable)
	if err != nil {
		return false
	}
	entryInfo, err := os.Stat(filepath.Join(directory, entry.Name()))
	return err == nil && os.SameFile(executableInfo, entryInfo)
}

func sameDirectory(first, second string) bool {
	if filepath.Clean(first) == filepath.Clean(second) {
		return true
	}
	firstInfo, err := os.Stat(first)
	if err != nil {
		return false
	}
	secondInfo, err := os.Stat(second)
	return err == nil && os.SameFile(firstInfo, secondInfo)
}

// FallbackRoot is where startup diagnostics go when the application fails
// before it resolves a data root: the executable folder on platforms with
// Portable storage, otherwise the Standard root.
func FallbackRoot() (string, error) {
	if portableSupported {
		return ExecutableDirectory()
	}
	return StandardRoot()
}

// storageArgument selects a storage mode when the application restarts after
// the user chose a data root.
const storageArgument = "--storage="

// RestartArguments returns the arguments that make a new process use mode.
func RestartArguments(mode Mode) []string {
	return []string{storageArgument + string(mode)}
}

// RequestedMode returns the storage mode that the arguments select, if any.
func RequestedMode(arguments []string) (Mode, error) {
	for _, argument := range arguments {
		if value, ok := strings.CutPrefix(argument, storageArgument); ok {
			return ParseMode(value)
		}
	}
	return "", nil
}

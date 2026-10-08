package appdata

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const maxListedItems = 3

type Mode string

const (
	ModePortable Mode = "portable"
	ModeStandard Mode = "standard"
)

type RootState string

const (
	RootEmpty      RootState = "empty"
	RootStructure  RootState = "structure"
	RootIncomplete RootState = "incomplete"
	RootVault      RootState = "vault"
)

type ChoiceReason string

const (
	ChoiceNone                 ChoiceReason = ""
	ChoiceFirstRun             ChoiceReason = "first-run"
	ChoicePortableWithoutVault ChoiceReason = "portable-without-vault"
	ChoiceConflict             ChoiceReason = "conflict"
)

// OtherItems lists at most maxListedItems names. OtherItemCount counts every entry except the executable.
type Candidate struct {
	Mode           Mode      `json:"mode"`
	Directory      string    `json:"directory"`
	State          RootState `json:"state"`
	OtherItems     []string  `json:"otherItems"`
	OtherItemCount int       `json:"otherItemCount"`
}

// Until Initialized is true, Mode and Directory name the provisional root.
type Location struct {
	Mode        Mode         `json:"mode"`
	Directory   string       `json:"directory"`
	Initialized bool         `json:"initialized"`
	Choice      ChoiceReason `json:"choice"`
	Candidates  []Candidate  `json:"candidates"`
	Moved       bool         `json:"moved"`
	MoveError   string       `json:"moveError"`
}

func (location Location) Candidate(mode Mode) (Candidate, bool) {
	for _, candidate := range location.Candidates {
		if candidate.Mode == mode {
			return candidate, true
		}
	}
	return Candidate{}, false
}

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

func Inspect(requested Mode) (Location, error) {
	executableDirectory, err := ExecutableDirectory()
	if err != nil {
		return Location{}, err
	}
	if err := os.Chdir(executableDirectory); err != nil {
		return Location{}, fmt.Errorf("set executable working directory: %w", err)
	}
	portableDirectory, standardDirectory, standardErr := candidateDirectories(executableDirectory)
	var portable, standard *Candidate
	if standardErr == nil {
		candidate, err := inspectCandidate(ModeStandard, standardDirectory)
		if err != nil {
			return Location{}, err
		}
		standard = &candidate
	}
	if portableDirectory != "" {
		candidate, err := inspectCandidate(ModePortable, portableDirectory)
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

func candidateDirectories(executableDirectory string) (string, string, error) {
	standard, err := StandardRoot()
	if !portableSupported || err == nil && sameDirectory(executableDirectory, standard) {
		return "", standard, err
	}
	return executableDirectory, standard, err
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
		if portable.State == RootEmpty {
			return use(portable, false, ChoiceFirstRun)
		}
		return use(portable, true, ChoiceNone)
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

// inspectRoot mirrors vault.FileState, so an incomplete vault is kept for
// recovery instead of being treated as unused.
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

// Earlier Linux versions let WebKitGTK write its own storage folder into the
// Standard root.
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

func FallbackRoot() (string, error) {
	if portableSupported {
		return ExecutableDirectory()
	}
	return StandardRoot()
}

const (
	storageArgument = "--storage="
	moveArgument    = "--move-storage="
)

func RestartArguments(mode Mode) []string {
	return []string{storageArgument + string(mode)}
}

func MoveArguments(target Mode) []string {
	return []string{moveArgument + string(target)}
}

func RequestedMode(arguments []string) (Mode, error) {
	return argumentMode(arguments, storageArgument)
}

func RequestedMove(arguments []string) (Mode, error) {
	return argumentMode(arguments, moveArgument)
}

func argumentMode(arguments []string, prefix string) (Mode, error) {
	for _, argument := range arguments {
		if value, ok := strings.CutPrefix(argument, prefix); ok {
			return ParseMode(value)
		}
	}
	return "", nil
}

package robloxcleaner

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"syscall"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

const Supported = true

const (
	startMenuPrograms = `Microsoft\Windows\Start Menu\Programs`
	classes           = `Software\Classes`
	uninstallEntries  = `Software\Microsoft\Windows\CurrentVersion\Uninstall`
)

var procRegDeleteTree = windows.NewLazySystemDLL("advapi32.dll").NewProc("RegDeleteTreeW")

var (
	robloxApplications = []string{"Roblox", "Roblox Studio"}
	// The Roblox installers name their shortcuts "Roblox Player" and
	// "Roblox Studio". Older installers named the player shortcut "Roblox".
	robloxShortcutNames = []string{"Roblox", "Roblox Player", "Roblox Studio"}
	protocols           = []string{"roblox", "roblox-player", "roblox-studio", "roblox-studio-auth"}
	fileExtensions      = []string{".rbxl", ".rbxlx", ".rbxm", ".rbxmx"}
)

// environmentPath is a path below a folder that an environment variable
// names. Its label, not the resolved path, goes into logs, because resolved
// paths usually contain the Windows user name.
type environmentPath struct {
	variable string
	relative string
}

func (path environmentPath) base() (string, bool) {
	base := os.Getenv(path.variable)
	if base == "" || !filepath.IsAbs(base) {
		return "", false
	}
	return filepath.Clean(base), true
}

func (path environmentPath) resolve() (string, bool) {
	base, ok := path.base()
	if !ok {
		return "", false
	}
	return filepath.Join(base, path.relative), true
}

func (path environmentPath) label() string {
	return filepath.Join("%"+path.variable+"%", path.relative)
}

func (path environmentPath) join(name string) environmentPath {
	return environmentPath{variable: path.variable, relative: filepath.Join(path.relative, name)}
}

var (
	localRobloxFolder = environmentPath{"LOCALAPPDATA", "Roblox"}
	// Folders that hold Roblox caches and cookies, deleted in both scopes.
	dataFolders = []environmentPath{
		{"APPDATA", `Roblox\http`},
		{"APPDATA", `Roblox\LocalStorage`},
		{"TEMP", "Roblox"},
		{"SystemRoot", `Temp\Roblox`},
	}
	// Folders that hold Roblox logs, deleted unless the user keeps logs.
	logFolders = []environmentPath{
		{"LOCALAPPDATA", `Roblox\logs`},
		{"APPDATA", `Roblox\logs`},
		{"TEMP", "RobloxLogs"},
		{"SystemRoot", `Temp\RobloxLogs`},
	}
	foldersWithLooseRobloxFiles = []environmentPath{
		{"TEMP", ""},
		{"SystemRoot", "Temp"},
		{"SystemRoot", "Prefetch"},
	}
	foldersWithApplicationFolders = []environmentPath{
		{"LOCALAPPDATA", ""},
		{"APPDATA", ""},
		{"ProgramData", ""},
		{"ProgramFiles", ""},
		{"ProgramFiles(x86)", ""},
	}
	// Start menus hold a folder of shortcuts for each Roblox application.
	foldersWithApplicationShortcuts = []environmentPath{
		{"APPDATA", startMenuPrograms},
		{"ProgramData", startMenuPrograms},
	}
	foldersWithShortcutFiles = []environmentPath{
		{"APPDATA", startMenuPrograms},
		{"ProgramData", startMenuPrograms},
		{"USERPROFILE", "Desktop"},
		{"PUBLIC", "Desktop"},
	}
	userContentFolders = []environmentPath{
		{"USERPROFILE", `Documents\ROBLOX`},
		{"USERPROFILE", `Pictures\Roblox`},
		{"USERPROFILE", `Videos\Roblox`},
	}
)

// isKeptLocalEntry reports whether an entry of %LOCALAPPDATA%\Roblox survives
// cache cleaning. Everything not listed is deleted, so files that new Roblox
// versions add are cleaned too. The list keeps the installed versions, the
// Roblox settings files, the fast flag settings, and Roblox Studio plugins.
// It also keeps the logs, which the log phase deletes when the user wants it.
func isKeptLocalEntry(name string) bool {
	lowercaseName := strings.ToLower(name)
	switch {
	case slices.Contains([]string{"versions", "clientsettings", "plugins", "logs"}, lowercaseName):
		return true
	default:
		return strings.HasSuffix(lowercaseName, ".xml") &&
			(strings.HasPrefix(lowercaseName, "globalbasicsettings_") || strings.HasPrefix(lowercaseName, "globalsettings_"))
	}
}

type registryKey struct {
	root registry.Key
	path string
}

type registryValue struct {
	key  registryKey
	name string
}

// Roblox Studio keeps the cookies of its signed-in account here.
var dataRegistryKeys = []registryKey{{registry.CURRENT_USER, `Software\Roblox\RobloxStudioBrowser`}}

func uninstallRegistryKeys() []registryKey {
	groups := []struct {
		parent string
		names  []string
	}{
		{classes, protocols},
		{classes, fileExtensions},
		{classes, []string{"Roblox.Place"}},
		{classes + `\Local Settings\Software\Microsoft\Windows\CurrentVersion\AppModel\PackageRepository\Extensions\windows.protocol`, protocols},
		{`Software\Microsoft\Internet Explorer\ProtocolExecute`, protocols},
		{`Software\WOW6432Node\Microsoft\Internet Explorer\ProtocolExecute`, protocols},
		{`Software\Microsoft\Windows\Shell\Associations\UrlAssociations`, protocols},
		{`Software\Microsoft\Windows\CurrentVersion\Explorer\FileExts`, fileExtensions},
		{`Software\Microsoft\RADAR\HeapLeakDetection\DiagnosedApplications`, []string{"RobloxPlayerBeta.exe", "RobloxStudioBeta.exe"}},
		{classes + `\Applications`, []string{"RobloxPlayerBeta.exe", "RobloxPlayerLauncher.exe", "RobloxStudioBeta.exe", "RobloxStudioLauncherBeta.exe"}},
		{uninstallEntries, protocols},
		{"Software", []string{"ROBLOX Corporation", "Roblox"}},
		{`Software\WOW6432Node`, []string{"ROBLOX Corporation", "Roblox"}},
	}
	var keys []registryKey
	for _, root := range []registry.Key{registry.CURRENT_USER, registry.LOCAL_MACHINE} {
		for _, group := range groups {
			for _, name := range group.names {
				keys = append(keys, registryKey{root, group.parent + `\` + name})
			}
		}
	}
	return keys
}

func uninstallRegistryValues() []registryValue {
	var values []registryValue
	for _, root := range []registry.Key{registry.CURRENT_USER, registry.LOCAL_MACHINE} {
		values = append(values, registryValue{registryKey{root, `Software\Microsoft\Windows\CurrentVersion\Run`}, "RobloxPlayerBeta"})
		for _, protocol := range protocols {
			values = append(values, registryValue{registryKey{root, `Software\RegisteredApplications`}, protocol})
		}
	}
	return values
}

// cleanupTarget is one file, folder, registry key, or registry value to
// delete. Its label goes into logs, and its display name, which can contain
// the Windows user name, is shown to the user. delete reports whether the
// target existed.
type cleanupTarget struct {
	label       string
	displayName string
	delete      func() (bool, error)
}

func listPhaseTargets(phase Phase) []cleanupTarget {
	switch phase {
	case PhaseRemovingApplications:
		return pathTargets(append(applicationPaths(), shortcutPaths()...))
	case PhaseRemovingRegistryEntries:
		return append(registryKeyTargets(uninstallRegistryKeys()), registryValueTargets(uninstallRegistryValues())...)
	case PhaseDeletingUserContent:
		return pathTargets(userContentFolders)
	case PhaseDeletingCaches:
		targets := append(pathTargets(dataFolders), localRobloxEntryTargets()...)
		targets = append(targets, registryKeyTargets(dataRegistryKeys)...)
		return append(targets, looseRobloxFileTargets()...)
	case PhaseDeletingLogs:
		return pathTargets(logFolders)
	case PhaseClosingProcesses:
		return nil
	}
	return nil
}

// retryDelays are the pauses before targets that were in use are deleted
// again. A closed process can hold its files for a moment after it exits, and
// antivirus and search indexing open new files briefly.
var retryDelays = []time.Duration{250 * time.Millisecond, time.Second}

func deleteTargets(ctx context.Context, targets []cleanupTarget, tracker *progressTracker) {
	var inUseTargets []cleanupTarget
	failures := map[string]error{}
	for _, target := range targets {
		if ctx.Err() != nil {
			return
		}
		wasDeleted, err := target.delete()
		switch {
		case isInUseError(err):
			inUseTargets = append(inUseTargets, target)
			failures[target.label] = err
		case err != nil:
			tracker.completeItem(itemSkipped, target.label, target.displayName, err)
		case wasDeleted:
			tracker.completeItem(itemDeleted, target.label, target.displayName, nil)
		default:
			tracker.completeItem(itemMissing, target.label, target.displayName, nil)
		}
	}
	for _, retryDelay := range retryDelays {
		if len(inUseTargets) == 0 || !waitForRetry(ctx, retryDelay) {
			break
		}
		var stillInUseTargets []cleanupTarget
		for _, target := range inUseTargets {
			// A retry can find the target gone, because the first attempt deleted
			// everything except the files that were in use.
			if _, err := target.delete(); isInUseError(err) {
				stillInUseTargets = append(stillInUseTargets, target)
				failures[target.label] = err
			} else if err != nil {
				tracker.completeItem(itemSkipped, target.label, target.displayName, err)
			} else {
				tracker.completeItem(itemDeleted, target.label, target.displayName, nil)
			}
		}
		inUseTargets = stillInUseTargets
	}
	for _, target := range inUseTargets {
		tracker.completeItem(itemSkipped, target.label, target.displayName, failures[target.label])
	}
}

func waitForRetry(ctx context.Context, retryDelay time.Duration) bool {
	timer := time.NewTimer(retryDelay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}

// isInUseError reports whether a deletion failed because another process
// holds a file open. A folder whose files are pending deletion is not empty
// until their last handle closes.
func isInUseError(err error) bool {
	return errors.Is(err, windows.ERROR_SHARING_VIOLATION) ||
		errors.Is(err, windows.ERROR_LOCK_VIOLATION) ||
		errors.Is(err, windows.ERROR_DIR_NOT_EMPTY)
}

// skipReasonOf names why a target or process was skipped for the user.
func skipReasonOf(err error) SkipReason {
	switch {
	case isInUseError(err):
		return SkipReasonInUse
	case errors.Is(err, windows.ERROR_ACCESS_DENIED), errors.Is(err, windows.ERROR_PRIVILEGE_NOT_HELD):
		return SkipReasonNeedsAdministratorRights
	case errors.Is(err, errReparsePoint):
		return SkipReasonLink
	default:
		return SkipReasonFailed
	}
}

// displayName returns the resolved path, or the label when the environment
// variable is not set.
func (path environmentPath) displayName() string {
	if resolved, ok := path.resolve(); ok {
		return resolved
	}
	return path.label()
}

func pathTargets(paths []environmentPath) []cleanupTarget {
	targets := make([]cleanupTarget, 0, len(paths))
	for _, path := range paths {
		targets = append(targets, cleanupTarget{path.label(), path.displayName(), func() (bool, error) { return removePath(path) }})
	}
	return targets
}

func registryKeyTargets(keys []registryKey) []cleanupTarget {
	targets := make([]cleanupTarget, 0, len(keys))
	for _, key := range keys {
		label := registryLabel(key)
		targets = append(targets, cleanupTarget{label, label, func() (bool, error) { return deleteRegistryKeyIfPresent(key) }})
	}
	return targets
}

func registryValueTargets(values []registryValue) []cleanupTarget {
	targets := make([]cleanupTarget, 0, len(values))
	for _, value := range values {
		label := registryLabel(value.key) + ` (value ` + value.name + `)`
		targets = append(targets, cleanupTarget{label, label, func() (bool, error) { return deleteRegistryValueIfPresent(value) }})
	}
	return targets
}

// failedListingTarget stands for a folder that could not be listed, so that
// the folder counts as skipped.
func failedListingTarget(folder environmentPath, err error) cleanupTarget {
	return cleanupTarget{folder.label(), folder.displayName(), func() (bool, error) { return false, err }}
}

func applicationPaths() []environmentPath {
	var paths []environmentPath
	for _, application := range robloxApplications {
		for _, folder := range foldersWithApplicationFolders {
			paths = append(paths, folder.join(application))
		}
	}
	return paths
}

func shortcutPaths() []environmentPath {
	var paths []environmentPath
	for _, folder := range foldersWithApplicationShortcuts {
		for _, application := range robloxApplications {
			paths = append(paths, folder.join(application))
		}
	}
	for _, folder := range foldersWithShortcutFiles {
		for _, shortcutName := range robloxShortcutNames {
			paths = append(paths, folder.join(shortcutName+".lnk"))
		}
	}
	return paths
}

func localRobloxEntryTargets() []cleanupTarget {
	base, ok := localRobloxFolder.base()
	if !ok {
		return nil
	}
	release, err := lockFolders(base, localRobloxFolder.relative)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return []cleanupTarget{failedListingTarget(localRobloxFolder, err)}
	}
	defer release()
	entries, err := os.ReadDir(filepath.Join(base, localRobloxFolder.relative))
	if err != nil {
		return []cleanupTarget{failedListingTarget(localRobloxFolder, err)}
	}
	var paths []environmentPath
	for _, entry := range entries {
		if !isKeptLocalEntry(entry.Name()) {
			paths = append(paths, localRobloxFolder.join(entry.Name()))
		}
	}
	return pathTargets(paths)
}

// removePath deletes a file or folder with everything in it. It reports
// whether the path existed.
func removePath(path environmentPath) (bool, error) {
	base, ok := path.base()
	if !ok {
		return false, nil
	}
	release, err := lockFolders(base, filepath.Dir(path.relative))
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	defer release()
	resolved := filepath.Join(base, path.relative)
	targetStatus, err := os.Lstat(resolved)
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("read path attributes: %w", err)
	}
	if isReparsePoint(targetStatus) {
		return false, errReparsePoint
	}
	// RemoveAll deletes relative to open folder handles and removes links
	// inside the target without following them. It deletes everything it
	// can before it returns the first error, so a retry only finds the rest.
	if err := os.RemoveAll(resolved); err != nil {
		return false, fmt.Errorf("remove path: %w", err)
	}
	return true, nil
}

var errReparsePoint = errors.New("the path is a link or junction")

// lockFolders opens the base folder and every folder of relativeFolder below
// it, and fails if one is a link or junction. The handles share reading and
// writing but not deletion, so while they are open no process can rename or
// delete a folder and put a link in its place, which would turn a deletion,
// possibly an elevated one, toward files outside of Roblox.
func lockFolders(base, relativeFolder string) (release func(), err error) {
	folders := []string{base}
	if relativeFolder != "." {
		folder := base
		for name := range strings.SplitSeq(relativeFolder, string(filepath.Separator)) {
			folder = filepath.Join(folder, name)
			folders = append(folders, folder)
		}
	}
	handles := make([]windows.Handle, 0, len(folders))
	release = func() {
		for _, handle := range handles {
			windows.CloseHandle(handle)
		}
	}
	for _, folder := range folders {
		handle, err := openFolderWithoutLinks(folder)
		if err != nil {
			release()
			return nil, err
		}
		handles = append(handles, handle)
	}
	return release, nil
}

func openFolderWithoutLinks(folder string) (windows.Handle, error) {
	path, err := windows.UTF16PtrFromString(folder)
	if err != nil {
		return 0, fmt.Errorf("encode folder path: %w", err)
	}
	// Windows records the share mode only for handles with data access, so
	// FILE_LIST_DIRECTORY is needed for the share mode to block renaming.
	handle, err := windows.CreateFile(path, windows.FILE_LIST_DIRECTORY|windows.FILE_READ_ATTRIBUTES,
		windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE, nil, windows.OPEN_EXISTING,
		windows.FILE_FLAG_BACKUP_SEMANTICS|windows.FILE_FLAG_OPEN_REPARSE_POINT, 0)
	if errors.Is(err, windows.ERROR_FILE_NOT_FOUND) || errors.Is(err, windows.ERROR_PATH_NOT_FOUND) {
		return 0, os.ErrNotExist
	}
	if err != nil {
		return 0, fmt.Errorf("open folder: %w", err)
	}
	var information windows.ByHandleFileInformation
	if err := windows.GetFileInformationByHandle(handle, &information); err != nil {
		windows.CloseHandle(handle)
		return 0, fmt.Errorf("read folder attributes: %w", err)
	}
	switch {
	case information.FileAttributes&windows.FILE_ATTRIBUTE_REPARSE_POINT != 0:
		windows.CloseHandle(handle)
		return 0, errReparsePoint
	case information.FileAttributes&windows.FILE_ATTRIBUTE_DIRECTORY == 0:
		windows.CloseHandle(handle)
		return 0, os.ErrNotExist
	}
	return handle, nil
}

func isReparsePoint(status os.FileInfo) bool {
	attributes, ok := status.Sys().(*syscall.Win32FileAttributeData)
	return ok && attributes.FileAttributes&windows.FILE_ATTRIBUTE_REPARSE_POINT != 0
}

func looseRobloxFileTargets() []cleanupTarget {
	var targets []cleanupTarget
	for _, folder := range foldersWithLooseRobloxFiles {
		resolved, ok := folder.resolve()
		if !ok {
			continue
		}
		entries, err := os.ReadDir(resolved)
		if err != nil {
			// Listing Windows\Prefetch needs administrator rights.
			if !errors.Is(err, os.ErrNotExist) {
				targets = append(targets, failedListingTarget(folder, err))
			}
			continue
		}
		var paths []environmentPath
		for _, entry := range entries {
			if entry.Type().IsRegular() && strings.HasPrefix(strings.ToLower(entry.Name()), "roblox") {
				paths = append(paths, folder.join(entry.Name()))
			}
		}
		targets = append(targets, pathTargets(paths)...)
	}
	return targets
}

// deleteRegistryKeyIfPresent deletes a key with everything in it. It reports
// whether the key existed.
func deleteRegistryKeyIfPresent(key registryKey) (bool, error) {
	existing, err := registry.OpenKey(key.root, key.path, registry.QUERY_VALUE)
	if errors.Is(err, registry.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("open registry key: %w", err)
	}
	_ = existing.Close()
	if err := deleteRegistryTree(key); err != nil {
		return false, err
	}
	return true, nil
}

// deleteRegistryTree deletes a key with all of its subkeys and values.
// RegDeleteTree empties the key, and DeleteKey removes the empty key itself.
func deleteRegistryTree(key registryKey) error {
	path, err := windows.UTF16PtrFromString(key.path)
	if err != nil {
		return fmt.Errorf("encode registry path: %w", err)
	}
	status, _, _ := procRegDeleteTree.Call(uintptr(key.root), uintptr(unsafe.Pointer(path)))
	if status != 0 && !errors.Is(syscall.Errno(status), windows.ERROR_FILE_NOT_FOUND) {
		return fmt.Errorf("delete registry tree: %w", syscall.Errno(status))
	}
	if err := registry.DeleteKey(key.root, key.path); err != nil && !errors.Is(err, registry.ErrNotExist) {
		return fmt.Errorf("delete registry key: %w", err)
	}
	return nil
}

// deleteRegistryValueIfPresent reports whether the value existed.
func deleteRegistryValueIfPresent(value registryValue) (bool, error) {
	reader, err := registry.OpenKey(value.key.root, value.key.path, registry.QUERY_VALUE)
	if errors.Is(err, registry.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("open registry key: %w", err)
	}
	_, _, readError := reader.GetValue(value.name, nil)
	_ = reader.Close()
	if errors.Is(readError, registry.ErrNotExist) {
		return false, nil
	}
	if err := deleteRegistryValue(value); err != nil {
		return false, err
	}
	return true, nil
}

func deleteRegistryValue(value registryValue) error {
	writer, err := registry.OpenKey(value.key.root, value.key.path, registry.SET_VALUE)
	if err != nil {
		return fmt.Errorf("open registry key for writing: %w", err)
	}
	deleteError := writer.DeleteValue(value.name)
	if errors.Is(deleteError, registry.ErrNotExist) {
		deleteError = nil
	}
	if deleteError != nil {
		deleteError = fmt.Errorf("delete registry value: %w", deleteError)
	}
	if err := writer.Close(); err != nil {
		return errors.Join(deleteError, fmt.Errorf("close registry key: %w", err))
	}
	return deleteError
}

func registryLabel(key registryKey) string {
	if key.root == registry.LOCAL_MACHINE {
		return `HKLM\` + key.path
	}
	return `HKCU\` + key.path
}

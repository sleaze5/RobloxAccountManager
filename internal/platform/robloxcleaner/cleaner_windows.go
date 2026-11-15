package robloxcleaner

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"syscall"
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
	protocols          = []string{"roblox", "roblox-player", "roblox-studio", "roblox-studio-auth"}
	fileExtensions     = []string{".rbxl", ".rbxlx", ".rbxm", ".rbxmx"}
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
		{"APPDATA", startMenuPrograms},
		{"ProgramData", startMenuPrograms},
	}
	foldersWithApplicationShortcuts = []environmentPath{
		{"APPDATA", startMenuPrograms},
		{"ProgramData", startMenuPrograms},
		{"USERPROFILE", "Desktop"},
	}
	userContentFolders = []environmentPath{
		{"USERPROFILE", `Documents\ROBLOX`},
		{"USERPROFILE", `Pictures\Roblox`},
		{"USERPROFILE", `Videos\Roblox`},
	}
)

// isKeptLocalEntry reports whether an entry of %LOCALAPPDATA%\Roblox survives
// data cleaning. Everything not listed is deleted, so files that new Roblox
// versions add are cleaned too. The list keeps the installed versions, the
// Roblox settings files, the fast flag settings, Roblox Studio plugins, and
// the logs when the user keeps them.
func isKeptLocalEntry(name string, shouldDeleteLogs bool) bool {
	lowercaseName := strings.ToLower(name)
	switch {
	case slices.Contains([]string{"versions", "clientsettings", "plugins"}, lowercaseName):
		return true
	case lowercaseName == "logs":
		return !shouldDeleteLogs
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

type cleanup struct {
	ctx          context.Context
	logger       *slog.Logger
	deletedCount int
	skippedCount int
}

// cleanupStep is one file, folder, registry key, or registry value to delete.
type cleanupStep struct {
	stage  Stage
	delete func()
}

func clean(ctx context.Context, scope Scope, shouldDeleteLogs bool, reportProgress func(Progress), logger *slog.Logger) (Outcome, error) {
	logger = logger.With("scope", scope, "should_delete_logs", shouldDeleteLogs)
	run := &cleanup{ctx: ctx, logger: logger}
	steps := run.plan(scope, shouldDeleteLogs)
	for index, step := range steps {
		if ctx.Err() != nil {
			break
		}
		reportProgress(Progress{Stage: step.stage, CompletedCount: index, TotalCount: len(steps)})
		step.delete()
	}
	if err := ctx.Err(); err != nil {
		return Outcome{}, fmt.Errorf("clean Roblox: %w", err)
	}
	logger.Info("Roblox cleaned", "deleted_count", run.deletedCount, "skipped_count", run.skippedCount)
	return Outcome{DeletedCount: run.deletedCount, SkippedCount: run.skippedCount}, nil
}

// plan lists every deletion first, so that progress can show the total.
func (run *cleanup) plan(scope Scope, shouldDeleteLogs bool) []cleanupStep {
	var paths []environmentPath
	var keys []registryKey
	var values []registryValue
	if shouldDeleteLogs {
		paths = append(paths, logFolders...)
	}
	switch scope {
	case ScopeData:
		paths = append(paths, run.localRobloxEntriesToDelete(shouldDeleteLogs)...)
		keys = dataRegistryKeys
	case ScopeUninstall:
		paths = append(paths, uninstallPaths()...)
		keys = uninstallRegistryKeys()
		values = uninstallRegistryValues()
	}
	paths = append(paths, dataFolders...)
	paths = append(paths, run.looseRobloxFiles()...)
	steps := make([]cleanupStep, 0, len(paths)+len(keys)+len(values))
	for _, path := range paths {
		steps = append(steps, cleanupStep{StageDeletingFiles, func() { run.removePath(path) }})
	}
	for _, key := range keys {
		steps = append(steps, cleanupStep{StageDeletingRegistryEntries, func() { run.deleteRegistryKeyIfPresent(key) }})
	}
	for _, value := range values {
		steps = append(steps, cleanupStep{StageDeletingRegistryEntries, func() { run.deleteRegistryValueIfPresent(value) }})
	}
	return steps
}

func uninstallPaths() []environmentPath {
	var paths []environmentPath
	for _, application := range robloxApplications {
		for _, folder := range foldersWithApplicationFolders {
			paths = append(paths, folder.join(application))
		}
		for _, folder := range foldersWithApplicationShortcuts {
			paths = append(paths, folder.join(application+".lnk"))
		}
	}
	return append(paths, userContentFolders...)
}

func (run *cleanup) localRobloxEntriesToDelete(shouldDeleteLogs bool) []environmentPath {
	base, ok := localRobloxFolder.base()
	if !ok {
		return nil
	}
	release, err := lockFolders(base, localRobloxFolder.relative)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		run.skip(localRobloxFolder.label(), err)
		return nil
	}
	defer release()
	entries, err := os.ReadDir(filepath.Join(base, localRobloxFolder.relative))
	if err != nil {
		run.skip(localRobloxFolder.label(), err)
		return nil
	}
	var paths []environmentPath
	for _, entry := range entries {
		if !isKeptLocalEntry(entry.Name(), shouldDeleteLogs) {
			paths = append(paths, localRobloxFolder.join(entry.Name()))
		}
	}
	return paths
}

func (run *cleanup) removePath(path environmentPath) {
	if run.ctx.Err() != nil {
		return
	}
	base, ok := path.base()
	if !ok {
		return
	}
	release, err := lockFolders(base, filepath.Dir(path.relative))
	if errors.Is(err, os.ErrNotExist) {
		return
	}
	if err != nil {
		run.skip(path.label(), err)
		return
	}
	defer release()
	resolved := filepath.Join(base, path.relative)
	targetStatus, err := os.Lstat(resolved)
	if errors.Is(err, os.ErrNotExist) {
		return
	}
	if err != nil {
		run.skip(path.label(), err)
		return
	}
	if isReparsePoint(targetStatus) {
		run.skip(path.label(), errReparsePoint)
		return
	}
	// RemoveAll deletes relative to open folder handles and removes links
	// inside the target without following them.
	if err := os.RemoveAll(resolved); err != nil {
		run.skip(path.label(), err)
		return
	}
	run.deletedCount++
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

func (run *cleanup) looseRobloxFiles() []environmentPath {
	var paths []environmentPath
	for _, folder := range foldersWithLooseRobloxFiles {
		resolved, ok := folder.resolve()
		if !ok {
			continue
		}
		entries, err := os.ReadDir(resolved)
		if err != nil {
			// Listing Windows\Prefetch needs administrator rights.
			if !errors.Is(err, os.ErrNotExist) {
				run.skip(folder.label(), err)
			}
			continue
		}
		for _, entry := range entries {
			if entry.Type().IsRegular() && strings.HasPrefix(strings.ToLower(entry.Name()), "roblox") {
				paths = append(paths, folder.join(entry.Name()))
			}
		}
	}
	return paths
}

func (run *cleanup) deleteRegistryKeyIfPresent(key registryKey) {
	existing, err := registry.OpenKey(key.root, key.path, registry.QUERY_VALUE)
	if errors.Is(err, registry.ErrNotExist) {
		return
	}
	if err != nil {
		run.skip(registryLabel(key), err)
		return
	}
	_ = existing.Close()
	if err := deleteRegistryTree(key); err != nil {
		run.skip(registryLabel(key), err)
		return
	}
	run.deletedCount++
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

func (run *cleanup) deleteRegistryValueIfPresent(value registryValue) {
	label := registryLabel(value.key) + `\` + value.name
	reader, err := registry.OpenKey(value.key.root, value.key.path, registry.QUERY_VALUE)
	if errors.Is(err, registry.ErrNotExist) {
		return
	}
	if err != nil {
		run.skip(label, err)
		return
	}
	_, _, readError := reader.GetValue(value.name, nil)
	_ = reader.Close()
	if errors.Is(readError, registry.ErrNotExist) {
		return
	}
	if err := deleteRegistryValue(value); err != nil {
		run.skip(label, err)
		return
	}
	run.deletedCount++
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

func (run *cleanup) skip(label string, err error) {
	run.skippedCount++
	run.logger.Debug("Roblox item skipped", "item", label, "error", err)
}

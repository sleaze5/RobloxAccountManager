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

func (path environmentPath) resolve() (string, bool) {
	base := os.Getenv(path.variable)
	if base == "" || !filepath.IsAbs(base) {
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
// data cleaning: the installed versions, the Roblox settings files, the fast
// flag settings, the user's Roblox Studio plugins, and optionally the logs.
func isKeptLocalEntry(name string, shouldDeleteLogs bool) bool {
	lowercaseName := strings.ToLower(name)
	if slices.Contains([]string{"versions", "clientsettings", "plugins"}, lowercaseName) {
		return true
	}
	if lowercaseName == "logs" {
		return !shouldDeleteLogs
	}
	return strings.HasSuffix(lowercaseName, ".xml") &&
		(strings.HasPrefix(lowercaseName, "globalbasicsettings_") || strings.HasPrefix(lowercaseName, "globalsettings_"))
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

func clean(ctx context.Context, scope Scope, shouldDeleteLogs bool, logger *slog.Logger) (bool, error) {
	logger = logger.With("scope", scope, "should_delete_logs", shouldDeleteLogs)
	run := &cleanup{ctx: ctx, logger: logger}
	if shouldDeleteLogs {
		run.removePaths(logFolders)
	}
	switch scope {
	case ScopeData:
		run.cleanLocalRobloxFolder(shouldDeleteLogs)
		run.removePaths(dataFolders)
		run.removeLooseRobloxFiles()
		run.deleteRegistryKeys(dataRegistryKeys)
	case ScopeUninstall:
		run.deleteRegistryKeys(uninstallRegistryKeys())
		run.deleteRegistryValues(uninstallRegistryValues())
		run.removePaths(uninstallPaths())
		run.removePaths(dataFolders)
		run.removeLooseRobloxFiles()
	}
	if err := ctx.Err(); err != nil {
		return false, fmt.Errorf("clean Roblox: %w", err)
	}
	logger.Info("Roblox cleaned", "deleted_count", run.deletedCount, "skipped_count", run.skippedCount)
	return run.skippedCount > 0, nil
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

func (run *cleanup) cleanLocalRobloxFolder(shouldDeleteLogs bool) {
	folder, ok := localRobloxFolder.resolve()
	if !ok {
		return
	}
	entries, err := os.ReadDir(folder)
	if errors.Is(err, os.ErrNotExist) {
		return
	}
	if err != nil {
		run.skip(localRobloxFolder.label(), err)
		return
	}
	for _, entry := range entries {
		if !isKeptLocalEntry(entry.Name(), shouldDeleteLogs) {
			run.removePath(localRobloxFolder.join(entry.Name()))
		}
	}
}

func (run *cleanup) removePaths(paths []environmentPath) {
	for _, path := range paths {
		run.removePath(path)
	}
}

func (run *cleanup) removePath(path environmentPath) {
	if run.ctx.Err() != nil {
		return
	}
	resolved, ok := path.resolve()
	if !ok {
		return
	}
	if _, err := os.Lstat(resolved); err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			run.skip(path.label(), err)
		}
		return
	}
	// RemoveAll removes links without following them, so a junction inside a
	// Roblox folder can never lead the deletion outside of it.
	if err := os.RemoveAll(resolved); err != nil {
		run.skip(path.label(), err)
		return
	}
	run.deletedCount++
}

func (run *cleanup) removeLooseRobloxFiles() {
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
				run.removePath(folder.join(entry.Name()))
			}
		}
	}
}

func (run *cleanup) deleteRegistryKeys(keys []registryKey) {
	for _, key := range keys {
		if run.ctx.Err() != nil {
			return
		}
		existing, err := registry.OpenKey(key.root, key.path, registry.QUERY_VALUE)
		if errors.Is(err, registry.ErrNotExist) {
			continue
		}
		if err != nil {
			run.skip(registryLabel(key), err)
			continue
		}
		_ = existing.Close()
		if err := deleteRegistryTree(key); err != nil {
			run.skip(registryLabel(key), err)
			continue
		}
		run.deletedCount++
	}
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

func (run *cleanup) deleteRegistryValues(values []registryValue) {
	for _, value := range values {
		if run.ctx.Err() != nil {
			return
		}
		label := registryLabel(value.key) + `\` + value.name
		reader, err := registry.OpenKey(value.key.root, value.key.path, registry.QUERY_VALUE)
		if errors.Is(err, registry.ErrNotExist) {
			continue
		}
		if err != nil {
			run.skip(label, err)
			continue
		}
		_, _, readError := reader.GetValue(value.name, nil)
		_ = reader.Close()
		if errors.Is(readError, registry.ErrNotExist) {
			continue
		}
		if err := deleteRegistryValue(value); err != nil {
			run.skip(label, err)
			continue
		}
		run.deletedCount++
	}
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

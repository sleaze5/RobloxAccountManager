package browser

import (
	"archive/zip"
	"context"
	"crypto/rand"
	"embed"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/sleaze5/RobloxAccountManager/internal/appdata"
	"github.com/sleaze5/RobloxAccountManager/internal/logging"
)

const (
	runtimeManifestVersion = 1
	runtimeMetadataVersion = 1
	runtimeMetadataName    = "runtime.json"
	archiveName            = "chrome.zip"
	approximateSize        = int64(170 * 1024 * 1024)
	invalidRuntimeError    = "The installed browser is invalid. Redownload it."
)

//go:embed runtime_manifest.json
var manifestFS embed.FS

type Manifest struct {
	SchemaVersion int               `json:"schemaVersion"`
	Version       string            `json:"version"`
	SourceURL     string            `json:"sourceUrl"`
	GeneratedAt   string            `json:"generatedAt"`
	Downloads     map[string]string `json:"downloads"`
}

type RuntimeStatus string

const (
	RuntimeMissing     RuntimeStatus = "missing"
	RuntimeReady       RuntimeStatus = "ready"
	RuntimeDamaged     RuntimeStatus = "damaged"
	RuntimeDownloading RuntimeStatus = "downloading"
	RuntimeInstalling  RuntimeStatus = "installing"
)

type RuntimeState struct {
	Status                 RuntimeStatus `json:"status"`
	RequiredVersion        string        `json:"requiredVersion"`
	InstalledVersion       string        `json:"installedVersion,omitempty"`
	InstalledBytes         int64         `json:"installedBytes"`
	ApproximateBytes       int64         `json:"approximateBytes"`
	DownloadedBytes        int64         `json:"downloadedBytes"`
	DownloadTotalBytes     int64         `json:"downloadTotalBytes"`
	DownloadBytesPerSecond float64       `json:"downloadBytesPerSecond"`
	DownloadETASeconds     *int64        `json:"downloadEtaSeconds"`
	Error                  string        `json:"error,omitempty"`
	CanCancel              bool          `json:"canCancel"`
}

type runtimeMetadata struct {
	SchemaVersion int    `json:"schemaVersion"`
	Version       string `json:"version"`
	InstalledAtMs int64  `json:"installedAtMs"`
}

type RuntimeManager struct {
	paths    appdata.Paths
	manifest Manifest
	client   *http.Client
	logger   *slog.Logger
	changed  func()

	mu     sync.Mutex
	state  RuntimeState
	cancel context.CancelFunc
	done   chan struct{}
}

func NewRuntimeManager(paths appdata.Paths, logger *slog.Logger, changed func()) (*RuntimeManager, error) {
	data, err := manifestFS.ReadFile("runtime_manifest.json")
	if err != nil {
		return nil, fmt.Errorf("read embedded CfT manifest: %w", err)
	}
	var manifest Manifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("invalid embedded CfT manifest")
	}
	logging.Diagnostic(logger, "browser manifest loaded", "operation", "browser-runtime-load",
		"schema_version", manifest.SchemaVersion, "supported_schema_version", runtimeManifestVersion, "browser_version", manifest.Version)
	if manifest.SchemaVersion != runtimeManifestVersion || manifest.Version == "" {
		return nil, fmt.Errorf("unsupported or incomplete embedded CfT manifest: schema version %d", manifest.SchemaVersion)
	}
	if err := validateManifestDownload(manifest); err != nil {
		return nil, err
	}
	if manifest.Downloads[runtimeDownloadKey] == "" {
		return nil, fmt.Errorf("managed browser is unavailable on this platform")
	}
	manager := &RuntimeManager{
		paths: paths, manifest: manifest,
		client: &http.Client{Timeout: 30 * time.Minute}, logger: logger, changed: changed,
		state: RuntimeState{RequiredVersion: manifest.Version, ApproximateBytes: approximateSize},
	}
	manager.cleanupStaging()
	manager.refreshLocked()
	logging.Diagnostic(logger, "browser runtime inspected", "operation", "browser-runtime-load",
		"status", manager.state.Status, "required_version", manager.state.RequiredVersion,
		"installed_version", manager.state.InstalledVersion, "supported_metadata_version", runtimeMetadataVersion)
	if manager.state.Status == RuntimeDamaged {
		manager.logger.Warn(
			"invalid CfT runtime detected",
			"required_version", manager.state.RequiredVersion,
			"installed_version", manager.state.InstalledVersion,
		)
	}
	return manager, nil
}

func (manager *RuntimeManager) Manifest() Manifest { return manager.manifest }
func (manager *RuntimeManager) MarkDamaged() {
	manager.mu.Lock()
	manager.state.Status = RuntimeDamaged
	manager.state.Error = invalidRuntimeError
	manager.mu.Unlock()
	manager.logger.Warn("CfT runtime marked damaged", "version", manager.manifest.Version)
	manager.notify()
}
func (manager *RuntimeManager) SetChanged(changed func()) {
	manager.mu.Lock()
	manager.changed = changed
	manager.mu.Unlock()
}

func (manager *RuntimeManager) State() RuntimeState {
	manager.mu.Lock()
	defer manager.mu.Unlock()
	return manager.state
}

func (manager *RuntimeManager) Executable() (string, error) {
	manager.mu.Lock()
	defer manager.mu.Unlock()
	manager.refreshLocked()
	if manager.state.Status != RuntimeReady {
		return "", errors.New("managed browser runtime is not installed or is damaged")
	}
	return filepath.Join(manager.installDirectory(), runtimeArchiveRoot, runtimeExecutableName), nil
}

func (manager *RuntimeManager) Download(parent context.Context, replace bool) error {
	manager.mu.Lock()
	if manager.cancel != nil {
		manager.mu.Unlock()
		return errors.New("a browser runtime download is already active")
	}
	manager.refreshLocked()
	if !replace && manager.state.Status == RuntimeReady {
		manager.mu.Unlock()
		return nil
	}
	ctx, cancel := context.WithCancel(parent)
	manager.cancel = cancel
	manager.done = make(chan struct{})
	manager.state.Status = RuntimeDownloading
	manager.state.Error = ""
	manager.state.DownloadedBytes = 0
	manager.state.DownloadTotalBytes = 0
	manager.state.CanCancel = true
	manager.mu.Unlock()
	manager.logger.Info("CfT runtime download started", "version", manager.manifest.Version, "replacement", replace)
	manager.notify()
	go manager.download(ctx)
	return nil
}

func (manager *RuntimeManager) CancelDownload() {
	manager.mu.Lock()
	cancel := manager.cancel
	manager.mu.Unlock()
	if cancel != nil {
		manager.logger.Info("CfT runtime download cancellation requested", "version", manager.manifest.Version)
		cancel()
	}
}

func (manager *RuntimeManager) download(ctx context.Context) {
	stopProgress := make(chan struct{})
	progressDone := make(chan struct{})
	go func() {
		manager.reportDownloadProgress(stopProgress)
		close(progressDone)
	}()
	err := manager.install(ctx)
	close(stopProgress)
	<-progressDone
	manager.finishDownload(err)
}

func (manager *RuntimeManager) install(ctx context.Context) error {
	defer manager.cleanupStaging()
	stageID, err := randomID(18)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(manager.paths.CfTRuntimeRoot, 0o755); err != nil {
		return fmt.Errorf("create CfT runtime root: %w", err)
	}
	if err := os.MkdirAll(manager.paths.CfTInstallRoot, 0o755); err != nil {
		return fmt.Errorf("create CfT staging root: %w", err)
	}
	stage := filepath.Join(manager.paths.CfTInstallRoot, stageID)
	if err := os.Mkdir(stage, 0o755); err != nil {
		return fmt.Errorf("create installation staging directory: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, manager.manifest.Downloads[runtimeDownloadKey], nil)
	if err != nil {
		return err
	}
	response, err := manager.client.Do(req)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("browser download returned status %d", response.StatusCode)
	}
	manager.setDownloadTotal(response.ContentLength)
	archive := filepath.Join(stage, archiveName)
	file, err := os.OpenFile(archive, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(file, &progressReader{reader: response.Body, update: manager.addProgress})
	closeErr := file.Close()
	if copyErr != nil {
		return copyErr
	}
	if closeErr != nil {
		return closeErr
	}
	manager.setInstalling()
	extracted := filepath.Join(stage, "extracted")
	if err := extractRuntime(ctx, archive, extracted); err != nil {
		return err
	}
	if err := validateRuntimeFiles(extracted); err != nil {
		return err
	}
	metadata, _ := json.MarshalIndent(runtimeMetadata{SchemaVersion: runtimeMetadataVersion, Version: manager.manifest.Version, InstalledAtMs: time.Now().UnixMilli()}, "", "  ")
	if err := os.WriteFile(filepath.Join(extracted, runtimeMetadataName), append(metadata, '\n'), 0o600); err != nil {
		return err
	}
	if err := manager.replaceRuntime(extracted, stage); err != nil {
		return err
	}
	return nil
}

func (manager *RuntimeManager) Remove() error {
	if err := manager.StopDownload(30 * time.Second); err != nil {
		return err
	}
	manager.mu.Lock()
	defer manager.mu.Unlock()
	for _, directory := range manager.managedRuntimeDirectories() {
		if err := removeManagedDirectory(manager.paths.CfTRuntimeRoot, directory); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
	}
	manager.refreshLocked()
	manager.logger.Info("CfT runtime removed", "version", manager.manifest.Version)
	manager.notifyAsync()
	return nil
}

func (manager *RuntimeManager) managedRuntimeDirectories() []string {
	directories := []string{manager.installDirectory()}
	entries, err := os.ReadDir(manager.paths.CfTRuntimeRoot)
	if err != nil {
		return directories
	}
	for _, entry := range entries {
		if !entry.IsDir() || entry.Name() == manager.manifest.Version {
			continue
		}
		directory := filepath.Join(manager.paths.CfTRuntimeRoot, entry.Name())
		metadataData, err := os.ReadFile(filepath.Join(directory, runtimeMetadataName))
		if err != nil {
			continue
		}
		var metadata runtimeMetadata
		if json.Unmarshal(metadataData, &metadata) == nil && metadata.SchemaVersion == runtimeMetadataVersion && metadata.Version == entry.Name() {
			directories = append(directories, directory)
		}
	}
	return directories
}

func (manager *RuntimeManager) StopDownload(timeout time.Duration) error {
	manager.mu.Lock()
	cancel, done := manager.cancel, manager.done
	manager.mu.Unlock()
	if cancel == nil || done == nil {
		return nil
	}
	cancel()
	select {
	case <-done:
		return nil
	case <-time.After(timeout):
		return errors.New("browser runtime work did not stop in time")
	}
}

func (manager *RuntimeManager) replaceRuntime(extracted, stage string) error {
	destination := manager.installDirectory()
	backup := filepath.Join(stage, "previous")
	hadPrevious := false
	if _, err := os.Stat(destination); err == nil {
		if err := renameRuntimeDirectory(destination, backup); err != nil {
			return fmt.Errorf("preserve previous runtime: %w", err)
		}
		hadPrevious = true
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if err := renameRuntimeDirectory(extracted, destination); err != nil {
		if hadPrevious {
			_ = renameRuntimeDirectory(backup, destination)
		}
		return fmt.Errorf("install browser runtime: %w", err)
	}
	if hadPrevious {
		_ = removeManagedDirectory(stage, backup)
	}
	return nil
}

func (manager *RuntimeManager) finishDownload(downloadErr error) {
	manager.mu.Lock()
	done := manager.done
	manager.cancel = nil
	manager.done = nil
	manager.state.CanCancel = false
	if downloadErr != nil {
		manager.refreshLocked()
		if errors.Is(downloadErr, context.Canceled) {
			manager.state.Error = ""
			manager.logger.Info("CfT runtime download cancelled", "version", manager.manifest.Version)
		} else {
			manager.state.Error = "The browser could not be installed. Try again."
			manager.logger.Error("CfT installation failed", "error", downloadErr)
		}
	} else {
		manager.refreshLocked()
		manager.logger.Info("CfT runtime installed", "version", manager.manifest.Version)
	}
	manager.mu.Unlock()
	if done != nil {
		close(done)
	}
	manager.notify()
}

func (manager *RuntimeManager) refreshLocked() {
	state := RuntimeState{Status: RuntimeMissing, RequiredVersion: manager.manifest.Version, ApproximateBytes: approximateSize}
	metadataData, err := os.ReadFile(filepath.Join(manager.installDirectory(), runtimeMetadataName))
	if errors.Is(err, os.ErrNotExist) {
		if version, installedBytes, found := manager.findInstalledRuntime(); found {
			state.Status = RuntimeDamaged
			state.InstalledVersion = version
			state.InstalledBytes = installedBytes
			state.Error = invalidRuntimeError
		}
		manager.state = state
		return
	}
	if err != nil {
		state.Status, state.Error = RuntimeDamaged, invalidRuntimeError
		manager.state = state
		return
	}
	var metadata runtimeMetadata
	state.InstalledBytes, _ = directorySize(manager.installDirectory())
	decodeErr := json.Unmarshal(metadataData, &metadata)
	if manager.state.InstalledVersion != metadata.Version || manager.state.Status == "" {
		if decodeErr != nil {
			manager.logger.Warn("browser runtime metadata could not be decoded", "operation", "browser-runtime-load", "error", decodeErr)
		} else {
			logging.Diagnostic(manager.logger, "browser runtime metadata read", "operation", "browser-runtime-load",
				"schema_version", metadata.SchemaVersion, "supported_schema_version", runtimeMetadataVersion, "browser_version", metadata.Version)
		}
	}
	if decodeErr != nil || metadata.SchemaVersion != runtimeMetadataVersion || metadata.Version != manager.manifest.Version || validateRuntimeFiles(manager.installDirectory()) != nil {
		state.Status, state.Error = RuntimeDamaged, invalidRuntimeError
		state.InstalledVersion = metadata.Version
		manager.state = state
		return
	}
	state.Status = RuntimeReady
	state.InstalledVersion = metadata.Version
	manager.state = state
}

func (manager *RuntimeManager) findInstalledRuntime() (string, int64, bool) {
	entries, err := os.ReadDir(manager.paths.CfTRuntimeRoot)
	if err != nil {
		return "", 0, false
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		root := filepath.Join(manager.paths.CfTRuntimeRoot, entry.Name())
		metadataData, _ := os.ReadFile(filepath.Join(root, runtimeMetadataName))
		var metadata runtimeMetadata
		if json.Unmarshal(metadataData, &metadata) != nil || metadata.SchemaVersion != runtimeMetadataVersion {
			metadata.Version = ""
		}
		installedBytes, _ := directorySize(root)
		return metadata.Version, installedBytes, true
	}
	return "", 0, false
}

func renameRuntimeDirectory(source, destination string) error {
	var err error
	for attempt := 1; attempt <= 5; attempt++ {
		if err = os.Rename(source, destination); err == nil {
			return nil
		}
		if attempt < 5 {
			time.Sleep(time.Duration(attempt) * 100 * time.Millisecond)
		}
	}
	return err
}

func (manager *RuntimeManager) installDirectory() string {
	return filepath.Join(manager.paths.CfTRuntimeRoot, manager.manifest.Version)
}

func (manager *RuntimeManager) setDownloadTotal(total int64) {
	manager.mu.Lock()
	manager.state.DownloadTotalBytes = max(total, 0)
	manager.mu.Unlock()
	manager.notify()
}

func (manager *RuntimeManager) addProgress(count int64) {
	manager.mu.Lock()
	manager.state.DownloadedBytes += count
	manager.mu.Unlock()
}

func (manager *RuntimeManager) reportDownloadProgress(done <-chan struct{}) {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	lastSample := time.Now()
	var lastBytes int64
	for {
		select {
		case <-done:
			return
		case <-ticker.C:
			manager.mu.Lock()
			if manager.state.Status != RuntimeDownloading {
				manager.mu.Unlock()
				return
			}
			now := time.Now()
			downloaded := manager.state.DownloadedBytes
			speed := float64(downloaded-lastBytes) / now.Sub(lastSample).Seconds()
			manager.state.DownloadBytesPerSecond = speed
			manager.state.DownloadETASeconds = nil
			if manager.state.DownloadTotalBytes > 0 && speed > 0 {
				remaining := max(manager.state.DownloadTotalBytes-downloaded, 0)
				seconds := int64(math.Ceil(float64(remaining) / speed))
				manager.state.DownloadETASeconds = &seconds
			}
			manager.mu.Unlock()
			lastBytes, lastSample = downloaded, now
			manager.notify()
		}
	}
}

func (manager *RuntimeManager) setInstalling() {
	manager.mu.Lock()
	manager.state.Status = RuntimeInstalling
	manager.state.CanCancel = false
	manager.state.DownloadBytesPerSecond = 0
	manager.state.DownloadETASeconds = nil
	manager.mu.Unlock()
	manager.notify()
}

func (manager *RuntimeManager) notify() {
	if manager.changed != nil {
		manager.changed()
	}
}
func (manager *RuntimeManager) notifyAsync() { go manager.notify() }

type progressReader struct {
	reader io.Reader
	update func(int64)
}

func (reader *progressReader) Read(buffer []byte) (int, error) {
	n, err := reader.reader.Read(buffer)
	if n > 0 {
		reader.update(int64(n))
	}
	return n, err
}

func extractRuntime(ctx context.Context, archive, destination string) error {
	reader, err := zip.OpenReader(archive)
	if err != nil {
		return fmt.Errorf("open browser archive: %w", err)
	}
	defer reader.Close()
	if err := os.Mkdir(destination, 0o755); err != nil {
		return err
	}
	// Links are created after every file, so no file is written through one.
	var links []archiveLink
	for _, entry := range reader.File {
		if err := ctx.Err(); err != nil {
			return err
		}
		name := filepath.Clean(filepath.FromSlash(entry.Name))
		if name == "." || filepath.IsAbs(name) || strings.HasPrefix(name, ".."+string(filepath.Separator)) {
			return errors.New("browser archive contains an unsafe path")
		}
		if name != runtimeArchiveRoot && !strings.HasPrefix(name, runtimeArchiveRoot+string(filepath.Separator)) {
			return errors.New("browser archive has an unexpected structure")
		}
		target := filepath.Join(destination, name)
		if entry.FileInfo().Mode()&os.ModeSymlink != 0 {
			if !runtimeArchiveLinks {
				return errors.New("browser archive contains a link")
			}
			link, err := readArchiveLink(entry)
			if err != nil {
				return err
			}
			links = append(links, archiveLink{path: target, target: link})
			continue
		}
		if entry.FileInfo().IsDir() {
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		source, err := entry.Open()
		if err != nil {
			return err
		}
		output, err := os.OpenFile(target, os.O_CREATE|os.O_EXCL|os.O_WRONLY, entry.Mode().Perm()|0o600)
		if err != nil {
			source.Close()
			return err
		}
		_, copyErr := io.Copy(output, io.LimitReader(source, int64(entry.UncompressedSize64)+1))
		closeErr := output.Close()
		source.Close()
		if copyErr != nil {
			return copyErr
		}
		if closeErr != nil {
			return closeErr
		}
	}
	for _, link := range links {
		if err := os.MkdirAll(filepath.Dir(link.path), 0o755); err != nil {
			return err
		}
		if err := os.Symlink(link.target, link.path); err != nil {
			return err
		}
	}
	return nil
}

type archiveLink struct {
	path   string
	target string
}

// readArchiveLink accepts only relative link targets without parent
// references, so a link and any chain of links stay inside the runtime.
func readArchiveLink(entry *zip.File) (string, error) {
	if entry.UncompressedSize64 == 0 || entry.UncompressedSize64 > 4096 {
		return "", errors.New("browser archive contains an invalid link")
	}
	source, err := entry.Open()
	if err != nil {
		return "", err
	}
	defer source.Close()
	data, err := io.ReadAll(io.LimitReader(source, 4097))
	if err != nil {
		return "", err
	}
	target := string(data)
	if len(data) > 4096 || strings.ContainsRune(target, 0) || strings.HasPrefix(target, "/") || filepath.IsAbs(target) {
		return "", errors.New("browser archive contains an unsafe link")
	}
	for _, part := range strings.Split(target, "/") {
		if part == ".." {
			return "", errors.New("browser archive contains an unsafe link")
		}
	}
	return target, nil
}

func validateRuntimeFiles(root string) error {
	for _, relative := range requiredRuntimeFiles {
		info, err := os.Stat(filepath.Join(root, runtimeArchiveRoot, relative))
		if err != nil || !info.Mode().IsRegular() {
			return fmt.Errorf("browser runtime is missing %s", relative)
		}
	}
	return nil
}

func directorySize(root string) (int64, error) {
	var size int64
	err := filepath.WalkDir(root, func(_ string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !entry.IsDir() {
			info, infoErr := entry.Info()
			if infoErr != nil {
				return infoErr
			}
			size += info.Size()
		}
		return nil
	})
	return size, err
}

func removeManagedDirectory(root, target string) error {
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return err
	}
	targetAbs, err := filepath.Abs(target)
	if err != nil {
		return err
	}
	relative, err := filepath.Rel(rootAbs, targetAbs)
	if err != nil || relative == "." || relative == "" || strings.HasPrefix(relative, ".."+string(filepath.Separator)) || filepath.IsAbs(relative) || strings.Contains(relative, string(filepath.Separator)) {
		return errors.New("refusing unsafe managed-directory cleanup")
	}
	info, err := os.Lstat(targetAbs)
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return errors.New("refusing linked managed-directory cleanup")
	}
	if err := filepath.WalkDir(targetAbs, func(_ string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		// RemoveAll removes a link without following it. Only runtimes that
		// ship links may contain them.
		if entry.Type()&os.ModeSymlink != 0 && !runtimeArchiveLinks {
			return errors.New("refusing managed-directory cleanup containing a link")
		}
		return nil
	}); err != nil {
		return err
	}
	return os.RemoveAll(targetAbs)
}

func removeEmptyDirectory(directory string) error {
	entries, err := os.ReadDir(directory)
	if errors.Is(err, os.ErrNotExist) || len(entries) > 0 {
		return nil
	}
	if err != nil {
		return err
	}
	err = os.Remove(directory)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}

func randomID(bytes int) (string, error) {
	value := make([]byte, bytes)
	if _, err := rand.Read(value); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(value), nil
}

func validRandomID(value string, bytes int) bool {
	decoded, err := base64.RawURLEncoding.DecodeString(value)
	return err == nil && len(decoded) == bytes
}

func validateManifestDownload(manifest Manifest) error {
	parsed, err := url.Parse(manifest.Downloads[runtimeDownloadKey])
	if err != nil || parsed.Scheme != "https" || parsed.Hostname() != "storage.googleapis.com" {
		return fmt.Errorf("embedded CfT manifest has an invalid %s URL", runtimeDownloadKey)
	}
	expected := "/chrome-for-testing-public/" + manifest.Version + "/" + runtimeDownloadKey + "/"
	if !strings.HasPrefix(parsed.Path, expected) {
		return fmt.Errorf("embedded CfT manifest has a mismatched %s URL", runtimeDownloadKey)
	}
	return nil
}

func (manager *RuntimeManager) cleanupStaging() {
	entries, err := os.ReadDir(manager.paths.CfTInstallRoot)
	if errors.Is(err, os.ErrNotExist) {
		return
	}
	if err != nil {
		manager.logger.Warn("CfT staging cleanup failed", "error", err)
		return
	}
	for _, entry := range entries {
		if entry.IsDir() && validRandomID(entry.Name(), 18) {
			if err := removeManagedDirectory(manager.paths.CfTInstallRoot, filepath.Join(manager.paths.CfTInstallRoot, entry.Name())); err != nil {
				manager.logger.Warn("stale CfT staging cleanup failed", "error", err)
			}
		}
	}
	if err := removeEmptyDirectory(manager.paths.CfTInstallRoot); err != nil {
		manager.logger.Warn("CfT staging root cleanup failed", "error", err)
	}
}

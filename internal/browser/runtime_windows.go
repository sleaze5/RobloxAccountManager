package browser

import "path/filepath"

const (
	runtimeDownloadKey    = "win64"
	runtimeArchiveRoot    = "chrome-win64"
	runtimeArchiveLinks   = false
	runtimeExecutableName = "chrome.exe"
)

var requiredRuntimeFiles = []string{"chrome.exe", "chrome.dll", "icudtl.dat", "resources.pak", filepath.Join("locales", "en-US.pak")}

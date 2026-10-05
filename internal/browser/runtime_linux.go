package browser

import "path/filepath"

const (
	runtimeDownloadKey    = "linux64"
	runtimeArchiveRoot    = "chrome-linux64"
	runtimeArchiveLinks   = false
	runtimeExecutableName = "chrome"
)

var requiredRuntimeFiles = []string{"chrome", "chrome_crashpad_handler", "icudtl.dat", "resources.pak", filepath.Join("locales", "en-US.pak")}

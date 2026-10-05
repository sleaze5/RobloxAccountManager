package browser

const (
	runtimeDownloadKey    = "mac-arm64"
	runtimeArchiveRoot    = "chrome-mac-arm64"
	runtimeArchiveLinks   = true
	runtimeExecutableName = macRuntimeBundle + "/Contents/MacOS/Google Chrome for Testing"
)

var requiredRuntimeFiles = macRequiredRuntimeFiles

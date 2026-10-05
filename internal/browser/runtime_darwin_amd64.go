package browser

const (
	runtimeDownloadKey    = "mac-x64"
	runtimeArchiveRoot    = "chrome-mac-x64"
	runtimeArchiveLinks   = true
	runtimeExecutableName = macRuntimeBundle + "/Contents/MacOS/Google Chrome for Testing"
)

var requiredRuntimeFiles = macRequiredRuntimeFiles

//go:build darwin

package browser

import "path/filepath"

const macRuntimeBundle = "Google Chrome for Testing.app"

var macRequiredRuntimeFiles = []string{
	filepath.Join(macRuntimeBundle, "Contents", "MacOS", "Google Chrome for Testing"),
	filepath.Join(macRuntimeBundle, "Contents", "Info.plist"),
	filepath.Join(macRuntimeBundle, "Contents", "Frameworks", "Google Chrome for Testing Framework.framework", "Google Chrome for Testing Framework"),
}

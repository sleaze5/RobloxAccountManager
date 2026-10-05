//go:build darwin

package browser

import "path/filepath"

// The macOS runtime is an application bundle. Its framework uses relative
// links, so the archive may contain links.
const macRuntimeBundle = "Google Chrome for Testing.app"

var macRequiredRuntimeFiles = []string{
	filepath.Join(macRuntimeBundle, "Contents", "MacOS", "Google Chrome for Testing"),
	filepath.Join(macRuntimeBundle, "Contents", "Info.plist"),
	filepath.Join(macRuntimeBundle, "Contents", "Frameworks", "Google Chrome for Testing Framework.framework", "Google Chrome for Testing Framework"),
}

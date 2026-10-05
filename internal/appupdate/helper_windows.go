//go:build windows

package appupdate

func helperCopiedUpdate() bool { return false }

func stageHelperArtifact() {}

// installTarget returns the path that an update replaces.
func installTarget(executable string) string { return executable }

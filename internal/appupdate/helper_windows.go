//go:build windows

package appupdate

func helperCopiedUpdate() bool { return false }

func stageHelperArtifact() {}

func installTarget(executable string) string { return executable }

package appupdate

import "github.com/wailsapp/wails/v3/pkg/updater"

func HandleHelperMode() {
	stageHelperArtifact()
	updater.HandleHelperMode()
}

package bindings

import (
	"errors"
	"fmt"

	"github.com/wailsapp/wails/v3/pkg/application"
)

const (
	exportFilterName    = "Roblox Account Manager export"
	exportFilterPattern = "*.ramexport"
)

var errDialogsUnavailable = errors.New("file dialogs are unavailable")

// An empty path without an error means the user cancelled the dialog.
func chooseSaveFile(directory, name string) (string, error) {
	app := application.Get()
	if app == nil {
		return "", errDialogsUnavailable
	}
	dialog := app.Dialog.SaveFile().
		SetDirectory(directory).
		SetFilename(name).
		AddFilter(exportFilterName, exportFilterPattern).
		CanCreateDirectories(true)
	if window := app.Window.Current(); window != nil {
		dialog.AttachToWindow(window)
	}
	path, err := dialog.PromptForSingleSelection()
	if err != nil {
		return "", fmt.Errorf("show save dialog: %w", err)
	}
	return path, nil
}

// An empty path without an error means the user cancelled the dialog.
func chooseOpenFile(directory string) (string, error) {
	app := application.Get()
	if app == nil {
		return "", errDialogsUnavailable
	}
	dialog := app.Dialog.OpenFile().
		SetDirectory(directory).
		AddFilter(exportFilterName, exportFilterPattern).
		CanChooseFiles(true)
	if window := app.Window.Current(); window != nil {
		dialog.AttachToWindow(window)
	}
	path, err := dialog.PromptForSingleSelection()
	if err != nil {
		return "", fmt.Errorf("show open dialog: %w", err)
	}
	return path, nil
}

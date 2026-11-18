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

// On Windows, Wails reports a dialog that the user cancelled as an error with
// this message, from an internal package that does not export the error. On
// other systems a cancelled dialog returns an empty path.
const windowsDialogCancelledMessage = "cancelled by user"

func isDialogCancelled(err error) bool {
	for ; err != nil; err = errors.Unwrap(err) {
		if err.Error() == windowsDialogCancelledMessage {
			return true
		}
	}
	return false
}

// An empty path without an error means the user cancelled the dialog.
func chooseSaveFile(directory, name, filterName, filterPattern string) (string, error) {
	app := application.Get()
	if app == nil {
		return "", errDialogsUnavailable
	}
	dialog := app.Dialog.SaveFile().
		SetDirectory(directory).
		SetFilename(name).
		AddFilter(filterName, filterPattern).
		CanCreateDirectories(true)
	if window := app.Window.Current(); window != nil {
		dialog.AttachToWindow(window)
	}
	path, err := dialog.PromptForSingleSelection()
	if isDialogCancelled(err) {
		return "", nil
	}
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
	if isDialogCancelled(err) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("show open dialog: %w", err)
	}
	return path, nil
}

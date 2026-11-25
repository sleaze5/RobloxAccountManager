export function isTextField(target: EventTarget | null): boolean {
	return (
		target instanceof HTMLInputElement ||
		target instanceof HTMLTextAreaElement ||
		(target instanceof HTMLElement && target.isContentEditable)
	)
}

// The WebView selects every text node in the window on Ctrl+A. Like a
// desktop app, select all only inside the focused text field or read-only
// textbox, such as the document viewer.
export function restrictSelectAllToFocusedText(event: KeyboardEvent): boolean {
	if (!(event.ctrlKey || event.metaKey) || event.key.toLowerCase() !== "a") {
		return false
	}
	const target = event.target
	if (isTextField(target)) return false
	event.preventDefault()
	if (target instanceof HTMLElement && target.getAttribute("role") === "textbox") {
		window.getSelection()?.selectAllChildren(target)
	}
	return true
}

// pageHistoryStepOfKey returns -1 for Alt+Left and 1 for Alt+Right, which go
// back and forward through what the page showed, and 0 for any other key.
export function pageHistoryStepOfKey(event: KeyboardEvent): -1 | 0 | 1 {
	if (!event.altKey || event.ctrlKey || event.metaKey || event.shiftKey) return 0
	if (event.key === "ArrowLeft") return -1
	return event.key === "ArrowRight" ? 1 : 0
}

// pageHistoryStepOfMouseButton returns -1 for the back button of a mouse, 1
// for its forward button, and 0 for any other button.
export function pageHistoryStepOfMouseButton(event: MouseEvent): -1 | 0 | 1 {
	if (event.button === 3) return -1
	return event.button === 4 ? 1 : 0
}

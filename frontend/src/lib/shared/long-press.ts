const longPressDelayMs = 500

// A long press runs its action when the primary pointer stays down for the
// delay. The click that follows the release is then skipped, so the button's
// normal click action does not also run.
export class LongPress {
	private pressTimer = 0
	private wasLongPressed = false

	constructor(private readonly onLongPress: () => void) {}

	beginPress = (event: PointerEvent): void => {
		if (event.button !== 0) return
		this.wasLongPressed = false
		window.clearTimeout(this.pressTimer)
		this.pressTimer = window.setTimeout(() => {
			this.wasLongPressed = true
			this.onLongPress()
		}, longPressDelayMs)
	}

	endPress = (): void => {
		window.clearTimeout(this.pressTimer)
	}

	forgetLongPress = (): void => {
		this.wasLongPressed = false
	}

	consumeLongPress(): boolean {
		const wasLongPressed = this.wasLongPressed
		this.wasLongPressed = false
		return wasLongPressed
	}
}

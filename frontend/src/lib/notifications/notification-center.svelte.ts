export interface NotificationAction {
	label: string
	emphasis?: boolean
	onClick: () => void | Promise<void>
}

export type NotificationTone = "info" | "success" | "warning" | "error"

export interface NotificationInput {
	id?: string
	tone?: NotificationTone
	title: string
	message: string
	actions?: NotificationAction[]
	dismissible?: boolean
	durationMs?: number
}

export interface AppNotification {
	id: string
	tone: NotificationTone
	title: string
	message: string
	actions: NotificationAction[]
	dismissible: boolean
}

export class NotificationCenter {
	items = $state<AppNotification[]>([])
	#nextId = 0
	readonly #timers = new Map<string, number>()
	readonly #durations = new Map<string, number>()

	show(input: NotificationInput): string {
		const id = input.id ?? `notification-${++this.#nextId}`,
			durationMs = input.durationMs ?? 5000,
			notification: AppNotification = {
				actions: input.actions ?? [],
				dismissible: input.dismissible ?? true,
				id,
				message: input.message,
				title: input.title,
				tone: input.tone ?? "info",
			}
		this.items = [...this.items.filter((item) => item.id !== id), notification]
		this.#clearTimer(id)
		this.#durations.delete(id)
		if (notification.actions.length === 0 && durationMs > 0) {
			this.#durations.set(id, durationMs)
			this.resume(id)
		}
		return id
	}

	pause(id: string): void {
		this.#clearTimer(id)
	}

	resume(id: string): void {
		const durationMs = this.#durations.get(id)
		if (durationMs === undefined || this.#timers.has(id)) return
		this.#timers.set(
			id,
			window.setTimeout(() => this.dismiss(id), durationMs),
		)
	}

	#clearTimer(id: string): void {
		window.clearTimeout(this.#timers.get(id))
		this.#timers.delete(id)
	}

	dismiss(id: string): void {
		this.#clearTimer(id)
		this.#durations.delete(id)
		if (!this.items.some((item) => item.id === id)) {
			return
		}
		this.items = this.items.filter((item) => item.id !== id)
	}

	dispose(): void {
		for (const timer of this.#timers.values()) window.clearTimeout(timer)
		this.#timers.clear()
		this.#durations.clear()
		this.items = []
	}
}

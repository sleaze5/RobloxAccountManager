export interface NotificationAction {
	label: string
	emphasis?: boolean
	onClick: () => void | Promise<void>
}

export interface NotificationInput {
	id?: string
	title: string
	message: string
	actions?: NotificationAction[]
	dismissible?: boolean
	durationMs?: number
}

export interface AppNotification {
	id: string
	title: string
	message: string
	actions: NotificationAction[]
	dismissible: boolean
}

export class NotificationCenter {
	items = $state<AppNotification[]>([])
	#nextId = 0
	readonly #timers = new Map<string, number>()

	show(input: NotificationInput): string {
		const id = input.id ?? `notification-${++this.#nextId}`,
			durationMs = input.durationMs ?? 5000,
			notification: AppNotification = {
				actions: input.actions ?? [],
				dismissible: input.dismissible ?? true,
				id,
				message: input.message,
				title: input.title,
			}
		this.items = [...this.items.filter((item) => item.id !== id), notification]
		this.#clearTimer(id)
		if (notification.actions.length === 0 && durationMs > 0) {
			this.#timers.set(
				id,
				window.setTimeout(() => this.dismiss(id), durationMs),
			)
		}
		return id
	}

	#clearTimer(id: string): void {
		window.clearTimeout(this.#timers.get(id))
		this.#timers.delete(id)
	}

	dismiss(id: string): void {
		this.#clearTimer(id)
		if (!this.items.some((item) => item.id === id)) {
			return
		}
		this.items = this.items.filter((item) => item.id !== id)
	}

	dispose(): void {
		for (const timer of this.#timers.values()) window.clearTimeout(timer)
		this.#timers.clear()
		this.items = []
	}
}

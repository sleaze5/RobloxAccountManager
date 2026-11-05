import { accountBackend } from "../backend/bridge"
import type {
	PrivateServerOption,
	PrivateServerSettings,
	PrivateServerUser,
} from "../backend/bridge"
import type { NotificationCenter } from "../notifications/notification-center.svelte"
import { errorMessage } from "../shared/errors"

export type PrivateServerAction =
	| "load"
	| "rename"
	| "link"
	| "add-user"
	| PrivateServerOption
	| `remove-${number}`

export class PrivateServerState {
	settings = $state<PrivateServerSettings | null>(null)
	pending = $state<PrivateServerAction | null>(null)
	error = $state<{ action: PrivateServerAction; message: string } | null>(null)
	#active = true
	#request: { cancel(): unknown } | null = null

	constructor(
		readonly accountId: number,
		readonly serverId: number,
		private readonly notifications: NotificationCenter,
		private readonly onChanged: (settings: PrivateServerSettings) => void,
	) {}

	errorFor(...actions: PrivateServerAction[]): string {
		return this.error && actions.includes(this.error.action)
			? this.error.message
			: ""
	}

	load(): Promise<boolean> {
		return this.#run(
			"load",
			() => accountBackend.GetPrivateServer(this.accountId, this.serverId),
			"Could not load the private server. Try again.",
		)
	}

	rename(name: string): Promise<boolean> {
		return this.#run("rename", () =>
			accountBackend.RenamePrivateServer(this.accountId, this.serverId, name),
		)
	}

	setOption(option: PrivateServerOption, enabled: boolean): Promise<boolean> {
		return this.#run(option, () =>
			accountBackend.SetPrivateServerOption(
				this.accountId,
				this.serverId,
				option,
				enabled,
			),
		)
	}

	resetLink(): Promise<boolean> {
		return this.#run("link", () =>
			accountBackend.ResetPrivateServerLink(this.accountId, this.serverId),
		)
	}

	addUser(user: string): Promise<boolean> {
		return this.#run("add-user", () =>
			accountBackend.AddPrivateServerUser(this.accountId, this.serverId, user),
		)
	}

	removeUser(user: PrivateServerUser): Promise<boolean> {
		return this.#run(`remove-${user.userId}`, () =>
			accountBackend.RemovePrivateServerUser(
				this.accountId,
				this.serverId,
				user.userId,
			),
		)
	}

	async copyLink(): Promise<void> {
		if (!this.settings?.joinLink) return
		try {
			await navigator.clipboard.writeText(this.settings.joinLink)
			this.notifications.show({
				id: "clipboard-copy",
				title: "Copied",
				tone: "success",
				message: "Join link copied to clipboard.",
			})
		} catch {
			this.error = {
				action: "link",
				message: "The join link could not be copied. Try again.",
			}
		}
	}

	async #run(
		action: PrivateServerAction,
		request: () => Promise<PrivateServerSettings> & { cancel(): unknown },
		fallback = "Roblox did not confirm the change. Refresh to check.",
	): Promise<boolean> {
		if (!this.#active || this.pending !== null) return false
		this.pending = action
		this.error = null
		const current = request()
		this.#request = current
		try {
			const settings = await current
			if (!this.#active) return false
			this.settings = settings
			if (action !== "load") this.onChanged(settings)
			return true
		} catch (error) {
			if (this.#active)
				this.error = { action, message: errorMessage(error, fallback) }
			return false
		} finally {
			this.#request = null
			if (this.#active) this.pending = null
		}
	}

	dispose(): void {
		this.#active = false
		if (this.pending === "load") void this.#request?.cancel()
		this.#request = null
	}
}

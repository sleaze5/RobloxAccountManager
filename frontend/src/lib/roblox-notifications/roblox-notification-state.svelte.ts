import { accountBackend, RobloxNotificationCategory } from "../backend/bridge"
import type { RobloxNotificationView } from "../backend/bridge"
import type { AccountStore } from "../accounts/account-store.svelte"
import { AccountScope } from "../accounts/account-scope"
import { errorMessage } from "../shared/errors"

export class RobloxNotificationState {
	notifications = $state<RobloxNotificationView[]>([])
	nextStartIndex = $state(0)
	loading = $state(false)
	loadingMore = $state(false)
	loadMoreFailed = $state(false)
	error = $state("")
	dismissingIds = $state<string[]>([])
	dismissingAll = $state(false)
	dismissAllError = $state("")
	// The empty category shows every type.
	category = $state<RobloxNotificationCategory>(RobloxNotificationCategory.$zero)
	visibleNotifications = $derived(
		this.category
			? this.notifications.filter(
					(notification) => notification.category === this.category,
				)
			: this.notifications,
	)
	#listGeneration = 0
	readonly #scope: AccountScope
	readonly #store: AccountStore

	constructor(accountId: number, store: AccountStore) {
		this.#scope = new AccountScope(accountId, store)
		this.#store = store
	}

	get hasDismissible(): boolean {
		return this.visibleNotifications.some(
			(notification) => notification.dismissAction,
		)
	}

	async refresh(): Promise<void> {
		if (!this.#scope.isCurrent || this.loading) return
		const generation = ++this.#listGeneration
		this.loading = true
		this.error = ""
		this.loadMoreFailed = false
		try {
			const page = await this.#scope.track(
				accountBackend.GetRobloxNotifications(this.#scope.accountId, 0),
			)
			if (!this.#scope.isCurrent || generation !== this.#listGeneration) return
			this.notifications = page.notifications ?? []
			this.nextStartIndex = page.nextStartIndex
			void this.#markSeen()
		} catch (error) {
			if (this.#scope.isCurrent && generation === this.#listGeneration) {
				this.error = errorMessage(
					error,
					"Could not load notifications. Try again.",
				)
			}
		} finally {
			if (generation === this.#listGeneration) this.loading = false
		}
	}

	// Showing the stream marks it seen on Roblox, as the website does. A failure
	// only leaves the unread badge in place, and the backend logs it.
	async #markSeen(): Promise<void> {
		try {
			await this.#scope.track(
				accountBackend.MarkRobloxNotificationsSeen(this.#scope.accountId),
			)
			if (!this.#scope.isCurrent) return
			this.#store.getAccountProfile(
				this.#scope.accountId,
			).snapshot.unreadNotificationCount = 0
		} catch {
			// The unread badge stays until the next successful mark.
		}
	}

	async loadMore(): Promise<void> {
		if (
			!this.#scope.isCurrent ||
			this.loading ||
			this.loadingMore ||
			!this.nextStartIndex
		) {
			return
		}
		const generation = this.#listGeneration
		this.loadingMore = true
		this.loadMoreFailed = false
		this.error = ""
		try {
			const page = await this.#scope.track(
				accountBackend.GetRobloxNotifications(
					this.#scope.accountId,
					this.nextStartIndex,
				),
			)
			if (!this.#scope.isCurrent || generation !== this.#listGeneration) return
			const known = new Set(this.notifications.map((listed) => listed.id))
			this.notifications = [
				...this.notifications,
				...(page.notifications ?? []).filter((listed) => !known.has(listed.id)),
			]
			this.nextStartIndex = page.nextStartIndex
		} catch (error) {
			if (this.#scope.isCurrent && generation === this.#listGeneration) {
				this.error = errorMessage(
					error,
					"Could not load more notifications. Try again.",
				)
				this.loadMoreFailed = true
			}
		} finally {
			this.loadingMore = false
		}
	}

	async dismiss(notification: RobloxNotificationView): Promise<void> {
		if (
			!this.#scope.isCurrent ||
			!notification.dismissAction ||
			this.dismissingAll ||
			this.dismissingIds.includes(notification.id)
		) {
			return
		}
		this.dismissingIds = [...this.dismissingIds, notification.id]
		this.error = ""
		try {
			await this.#scope.track(
				accountBackend.DismissRobloxNotification(
					this.#scope.accountId,
					notification.id,
					notification.dismissAction,
				),
			)
			if (!this.#scope.isCurrent) return
			this.notifications = this.notifications.filter(
				(listed) => listed.id !== notification.id,
			)
			// The dismissed entry no longer counts toward the next page offset.
			if (this.nextStartIndex > 0) this.nextStartIndex--
		} catch (error) {
			if (this.#scope.isCurrent) {
				this.error = errorMessage(
					error,
					"Could not dismiss this notification. Try again.",
				)
			}
		} finally {
			this.dismissingIds = this.dismissingIds.filter(
				(id) => id !== notification.id,
			)
		}
	}

	async dismissAll(): Promise<boolean> {
		if (!this.#scope.isCurrent || this.dismissingAll) return false
		this.dismissingAll = true
		this.dismissAllError = ""
		let isDismissed = false
		try {
			await this.#scope.track(
				accountBackend.DismissAllRobloxNotifications(
					this.#scope.accountId,
					this.category,
				),
			)
			isDismissed = true
		} catch (error) {
			if (this.#scope.isCurrent) {
				this.dismissAllError = errorMessage(
					error,
					"Could not dismiss notifications. Try again.",
				)
			}
		} finally {
			this.dismissingAll = false
		}
		// Dismiss all can stop partway, so reload what Roblox still lists.
		await this.refresh()
		return isDismissed
	}

	dispose(): void {
		this.#scope.dispose()
		this.#listGeneration++
	}
}

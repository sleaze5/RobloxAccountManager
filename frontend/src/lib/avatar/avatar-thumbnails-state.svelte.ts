import { accountBackend } from "../backend/bridge"
import type {
	AvatarThumbnailChange,
	AvatarThumbnails,
	AvatarThumbnailType,
} from "../backend/bridge"
import type { AccountStore } from "../accounts/account-store.svelte"
import { AccountScope } from "../accounts/account-scope"
import type { NotificationCenter } from "../notifications/notification-center.svelte"
import { errorMessage } from "../shared/errors"

// AvatarThumbnailsState shows how Roblox poses and frames the avatar images
// of one user. When the user is the account itself, it also saves them.
export class AvatarThumbnailsState {
	thumbnails = $state.raw<AvatarThumbnails | null>(null)
	isLoading = $state(false)
	error = $state("")
	// The customization that saves, or undefined.
	savingType = $state<AvatarThumbnailType | undefined>(undefined)
	saveError = $state("")
	readonly userId: number
	readonly #scope: AccountScope
	readonly #notifications: NotificationCenter
	#loadRequest: ReturnType<typeof accountBackend.GetAvatarThumbnails> | null = null

	constructor(
		accountId: number,
		userId: number,
		store: AccountStore,
		notifications: NotificationCenter,
	) {
		this.#scope = new AccountScope(accountId, store)
		this.#notifications = notifications
		this.userId = userId
	}

	async refresh(): Promise<void> {
		if (!this.#scope.isCurrent) return
		void this.#loadRequest?.cancel()
		const request = accountBackend.GetAvatarThumbnails(
			this.#scope.accountId,
			this.userId,
		)
		this.#loadRequest = request
		this.isLoading = true
		this.error = ""
		try {
			const thumbnails = await this.#scope.track(request)
			if (request !== this.#loadRequest || !this.#scope.isCurrent) return
			this.thumbnails = thumbnails
		} catch (error) {
			if (request !== this.#loadRequest || !this.#scope.isCurrent) return
			this.error = errorMessage(
				error,
				"Could not load the thumbnails. Try again.",
			)
		} finally {
			if (request === this.#loadRequest) {
				this.#loadRequest = null
				this.isLoading = false
			}
		}
	}

	// save replaces one customization, reads the customizations again, and
	// reports whether Roblox saved it. Roblox renders the images a few
	// seconds later.
	async save(change: AvatarThumbnailChange): Promise<boolean> {
		if (!this.#scope.isCurrent || this.savingType) return false
		this.savingType = change.type
		this.saveError = ""
		try {
			await this.#scope.track(
				accountBackend.SaveAvatarThumbnail(this.#scope.accountId, change),
			)
			if (!this.#scope.isCurrent) return false
			this.#notifications.show({
				id: "avatar-thumbnail-save",
				tone: "success",
				title: "Thumbnail saved",
				message: "Roblox shows the new images in a few seconds.",
			})
			void this.refresh()
			return true
		} catch (error) {
			if (this.#scope.isCurrent) {
				this.saveError = errorMessage(
					error,
					"Could not save the thumbnail. Try again.",
				)
			}
			return false
		} finally {
			this.savingType = undefined
		}
	}

	dispose(): void {
		this.#scope.dispose()
	}
}

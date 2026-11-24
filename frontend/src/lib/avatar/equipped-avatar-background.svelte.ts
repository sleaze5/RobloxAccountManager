import { accountBackend } from "../backend/bridge"
import type { AvatarBackground } from "../backend/bridge"

// EquippedAvatarBackground holds the Marketplace background that a user
// equipped, which a profile shows behind the avatar.
export class EquippedAvatarBackground {
	background = $state.raw<AvatarBackground | null>(null)
	isLoading = $state(false)
	hasFailed = $state(false)

	// load reads the background of the user as the account sees it, and
	// returns a function that cancels the read.
	load(accountId: number, userId: number): () => void {
		this.background = null
		this.hasFailed = false
		if (accountId <= 0 || userId <= 0) {
			this.isLoading = false
			return () => {}
		}
		this.isLoading = true
		let isCurrent = true
		const request = accountBackend.GetAvatarBackground(accountId, userId)
		void (async () => {
			try {
				const background = await request
				if (isCurrent) this.background = background
			} catch {
				// The backend logs the failure. The avatar shows without a
				// background, and the fact says that it is unavailable.
				if (isCurrent) this.hasFailed = true
			} finally {
				if (isCurrent) this.isLoading = false
			}
		})()
		return () => {
			isCurrent = false
			void request.cancel()
		}
	}
}

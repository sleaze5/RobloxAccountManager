import { accountBackend } from "../backend/bridge"
import type { AvatarPortrait } from "../backend/bridge"

// AvatarPortraitLoad holds the full-body avatar image of a user and the
// Marketplace background that the user equipped, which a profile shows
// together.
export class AvatarPortraitLoad {
	portrait = $state.raw<AvatarPortrait | null>(null)
	isLoading = $state(false)
	hasFailed = $state(false)

	// load reads the portrait of the user as the account sees it, and returns
	// a function that cancels the read.
	load(accountId: number, userId: number): () => void {
		this.portrait = null
		this.hasFailed = false
		if (accountId <= 0 || userId <= 0) {
			this.isLoading = false
			return () => {}
		}
		this.isLoading = true
		let isCurrent = true
		const request = accountBackend.GetAvatarPortrait(accountId, userId)
		void (async () => {
			try {
				const portrait = await request
				if (isCurrent) this.portrait = portrait
			} catch {
				// The backend logs the failure. The profile shows the headshot,
				// and the background fact says that it is unavailable.
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

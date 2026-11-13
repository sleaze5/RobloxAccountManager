import { accountBackend } from "../backend/bridge"
import type { GameBadge } from "../backend/bridge"
import { errorMessage } from "../shared/errors"
import { isFresh } from "../shared/freshness"

// UniverseBadges holds the badges of one universe and when the shown account
// earned each of them. The games store keeps it while the place stays
// selected, so that the Badges page reopens at once.
export class UniverseBadges {
	badges = $state<GameBadge[] | null>(null)
	loading = $state(false)
	error = $state("")
	awardedAtByBadgeId = $state<Map<number, number> | null>(null)
	loadingAwards = $state(false)
	awardsError = $state("")
	#accountId = 0
	#badgesFetchedAt = 0
	#awardsFetchedAt = 0
	#badgeRequest: ReturnType<typeof accountBackend.ListGameBadges> | null = null
	#awardRequest: ReturnType<typeof accountBackend.ListBadgeAwards> | null = null

	constructor(readonly universeId: number) {}

	// showAccount shows the awards of another account. It loads them when the
	// kept awards belong to another account or are no longer fresh.
	showAccount(accountId: number): void {
		if (accountId === this.#accountId && isFresh(this.#awardsFetchedAt)) return
		if (accountId !== this.#accountId) {
			this.#cancelAwards()
			this.awardedAtByBadgeId = null
			this.awardsError = ""
			this.#accountId = accountId
		}
		if (this.badges) void this.#loadAwards()
	}

	// load shows the kept badges at once and requests them again when they are
	// no longer fresh. The awards load after the badges, because Roblox reads
	// them by badge ID.
	async load(): Promise<void> {
		if (this.#badgeRequest || (this.badges && isFresh(this.#badgesFetchedAt))) {
			return
		}
		await this.refresh()
	}

	async refresh(): Promise<void> {
		void this.#badgeRequest?.cancel()
		const request = accountBackend.ListGameBadges(this.universeId)
		this.#badgeRequest = request
		this.loading = true
		this.error = ""
		try {
			const badges = await request
			if (this.#badgeRequest !== request) return
			this.badges = badges ?? []
			this.#badgesFetchedAt = Date.now()
		} catch (error) {
			if (this.#badgeRequest === request) {
				this.error = errorMessage(
					error,
					"Badges could not be loaded. Try again.",
				)
			}
			return
		} finally {
			if (this.#badgeRequest === request) {
				this.#badgeRequest = null
				this.loading = false
			}
		}
		await this.#loadAwards()
	}

	async #loadAwards(): Promise<void> {
		this.#cancelAwards()
		const accountId = this.#accountId,
			badges = this.badges
		if (!accountId || !badges) return
		const request = accountBackend.ListBadgeAwards({
			accountId,
			badgeIds: badges.map((badge) => badge.badgeId),
		})
		this.#awardRequest = request
		this.loadingAwards = true
		this.awardsError = ""
		try {
			const awards = await request
			if (this.#awardRequest !== request) return
			this.awardedAtByBadgeId = new Map(
				(awards ?? []).map((award) => [award.badgeId, award.awardedAtMs]),
			)
			this.#awardsFetchedAt = Date.now()
		} catch (error) {
			if (this.#awardRequest === request) {
				this.awardsError = errorMessage(
					error,
					"Earned badges could not be loaded. Try again.",
				)
			}
		} finally {
			if (this.#awardRequest === request) {
				this.#awardRequest = null
				this.loadingAwards = false
			}
		}
	}

	#cancelAwards(): void {
		void this.#awardRequest?.cancel()
		this.#awardRequest = null
		this.#awardsFetchedAt = 0
		this.loadingAwards = false
	}

	dispose(): void {
		void this.#badgeRequest?.cancel()
		this.#badgeRequest = null
		this.#cancelAwards()
	}
}

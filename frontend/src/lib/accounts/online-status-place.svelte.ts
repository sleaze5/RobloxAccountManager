import { OnlineStatusType } from "../backend/bridge"
import type { GamePlace, GamePlaceSummary, OnlineStatus } from "../backend/bridge"
import { loadGamePlace } from "../games/game-place-cache"

export function isInGame(onlineStatus: OnlineStatus): boolean {
	return (
		onlineStatus.statusType === OnlineStatusType.OnlineStatusTypeInGame ||
		onlineStatus.statusType === OnlineStatusType.OnlineStatusTypeInStudio
	)
}

// OnlineStatusPlace loads the place that an online status is in. Roblox names the
// universe in the online status location, so the place itself, which can be a
// subplace, comes from the games API.
export class OnlineStatusPlace {
	#summary = $state<GamePlaceSummary | null>(null)
	readonly #onlineStatus: () => OnlineStatus
	readonly #universeId = $derived.by(() => {
		const onlineStatus = this.#onlineStatus()
		return isInGame(onlineStatus) ? (onlineStatus.universeId ?? 0) : 0
	})
	readonly #placeId = $derived.by(() => this.#onlineStatus().placeId ?? 0)

	constructor(onlineStatus: () => OnlineStatus) {
		this.#onlineStatus = onlineStatus
		$effect(() => {
			const universeId = this.#universeId,
				placeId = this.#placeId
			this.#summary = null
			if (universeId <= 0 || placeId <= 0) return undefined
			let isCurrent = true
			void loadGamePlace(universeId, placeId).then((summary) => {
				if (isCurrent) this.#summary = summary
				return summary
			})
			return () => {
				isCurrent = false
			}
		})
	}

	get place(): GamePlace | null {
		return this.#summary?.place ?? null
	}
}

import { PresenceType } from "../backend/bridge"
import type { GamePlace, GamePlaceSummary, UserPresence } from "../backend/bridge"
import { loadGamePlace } from "../games/game-place-cache"

export function isInExperience(presence: UserPresence): boolean {
	return (
		presence.userPresenceType === PresenceType.PresenceTypeInGame ||
		presence.userPresenceType === PresenceType.PresenceTypeInStudio
	)
}

// PresencePlace loads the place that a presence is in. Roblox names the
// universe in the presence location, so the place itself, which can be a
// subplace, comes from the games API.
export class PresencePlace {
	#summary = $state<GamePlaceSummary | null>(null)
	readonly #presence: () => UserPresence
	readonly #universeId = $derived.by(() => {
		const presence = this.#presence()
		return isInExperience(presence) ? (presence.universeId ?? 0) : 0
	})
	readonly #placeId = $derived.by(() => this.#presence().placeId ?? 0)

	constructor(presence: () => UserPresence) {
		this.#presence = presence
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

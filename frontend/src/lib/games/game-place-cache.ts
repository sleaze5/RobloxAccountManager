import { accountBackend } from "../backend/bridge"
import type { GamePlaceSummary } from "../backend/bridge"

const places = new Map<string, Promise<GamePlaceSummary | null>>()

export function loadGamePlace(
	universeId: number,
	placeId: number,
): Promise<GamePlaceSummary | null> {
	const key = `${universeId}:${placeId}`
	let request = places.get(key)
	if (!request) {
		request = accountBackend.GetGamePlace(universeId, placeId).then(
			(summary) => summary,
			() => {
				places.delete(key)
				return null
			},
		)
		places.set(key, request)
	}
	return request
}

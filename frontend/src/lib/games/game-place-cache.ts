import { accountBackend } from "../backend/bridge"
import type { GamePlace } from "../backend/bridge"

const places = new Map<number, Promise<GamePlace | null>>()

export function loadGamePlace(universeId: number): Promise<GamePlace | null> {
	let request = places.get(universeId)
	if (!request) {
		request = accountBackend.GetGamePlace(universeId).then(
			(place) => place,
			() => {
				places.delete(universeId)
				return null
			},
		)
		places.set(universeId, request)
	}
	return request
}

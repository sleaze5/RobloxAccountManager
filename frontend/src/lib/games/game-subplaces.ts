import { accountBackend } from "../backend/bridge"
import type { GamePlace } from "../backend/bridge"

// The subplaces of a game rarely change, so the game details and the game
// preview share one read of them for each game. A failed read is read again.
const subplacesByUniverseId = new Map<number, Promise<GamePlace[]>>()

export function loadGameSubplaces(universeId: number): Promise<GamePlace[]> {
	const savedRequest = subplacesByUniverseId.get(universeId)
	if (savedRequest) return savedRequest
	const request = accountBackend.ListGameSubplaces(universeId).then(
		(subplaces) => subplaces ?? [],
		(error: unknown) => {
			subplacesByUniverseId.delete(universeId)
			throw error
		},
	)
	subplacesByUniverseId.set(universeId, request)
	return request
}

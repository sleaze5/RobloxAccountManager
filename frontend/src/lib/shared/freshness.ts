const detailsFreshMs = 60_000

// Kept details show at once while they refresh, instead of a loading state.
export const detailsKeepMs = 30 * 60_000

export function isFresh(fetchedAt: number): boolean {
	return Date.now() - fetchedAt < detailsFreshMs
}

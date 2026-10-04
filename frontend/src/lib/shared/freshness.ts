export const detailsFreshMs = 60_000

export const detailsKeepMs = 5 * 60_000

export function isFresh(fetchedAt: number): boolean {
	return Date.now() - fetchedAt < detailsFreshMs
}

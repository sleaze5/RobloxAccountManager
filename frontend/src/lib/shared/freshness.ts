/** Details younger than this are shown without asking Roblox again. */
export const detailsFreshMs = 60_000

/** Older details stay on screen while newer ones load, up to this age. */
export const detailsKeepMs = 5 * 60_000

export function isFresh(fetchedAt: number): boolean {
	return Date.now() - fetchedAt < detailsFreshMs
}

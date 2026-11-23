import {
	accountBackend,
	CatalogCreatorType,
	CatalogSortOrder,
	UserGameList,
} from "../backend/bridge"
import type { CategorizedInventoryItem, GamePlace, UserGroup } from "../backend/bridge"
import { defaultCatalogSearch } from "../catalog/catalog-store.svelte"
import { isFresh } from "../shared/freshness"
import { AccountScope } from "./account-scope"
import type { AccountStore } from "./account-store.svelte"

export type ProfileShowcasePart =
	| "groups"
	| "favoriteGames"
	| "createdGames"
	| "creations"

// ProfileShowcase holds the groups, games, and Marketplace creations at the
// bottom of one user's profile, as one account sees them. Each part loads on
// its own, so that one failure leaves the others.
export class ProfileShowcase {
	groups = $state.raw<UserGroup[] | null>(null)
	primaryGroupId = $state(0)
	favoriteGames = $state.raw<GamePlace[] | null>(null)
	createdGames = $state.raw<GamePlace[] | null>(null)
	creations = $state.raw<CategorizedInventoryItem[] | null>(null)
	hasMoreCreations = $state(false)
	failedParts = $state.raw<ProfileShowcasePart[]>([])
	#loadedAt = 0
	readonly #scope: AccountScope

	constructor(
		accountId: number,
		readonly userId: number,
		readonly username: string,
		accounts: AccountStore,
	) {
		this.#scope = new AccountScope(accountId, accounts)
	}

	// isReusable reports whether a profile that opens again can show these
	// parts without reading them again.
	get isReusable(): boolean {
		return this.#scope.isCurrent && isFresh(this.#loadedAt)
	}

	load(): void {
		this.#loadedAt = Date.now()
		this.failedParts = []
		void this.#loadPart("groups", async () => {
			const list = await this.#scope.track(
				accountBackend.GetUserGroups(this.#scope.accountId, this.userId),
			)
			return () => {
				this.groups = list.groups ?? []
				this.primaryGroupId = list.primaryGroupId
			}
		})
		void this.#loadPart("favoriteGames", async () => {
			const places = await this.#scope.track(
				accountBackend.ListUserGames(
					this.userId,
					UserGameList.UserGameListFavorites,
				),
			)
			return () => (this.favoriteGames = places ?? [])
		})
		void this.#loadPart("createdGames", async () => {
			const places = await this.#scope.track(
				accountBackend.ListUserGames(
					this.userId,
					UserGameList.UserGameListCreated,
				),
			)
			return () => (this.createdGames = places ?? [])
		})
		void this.#loadPart("creations", async () => {
			const page = await this.#scope.track(
				accountBackend.SearchCatalog(
					this.#scope.accountId,
					{
						...defaultCatalogSearch(),
						creatorName: this.username,
						creatorType: CatalogCreatorType.CatalogCreatorUser,
						sortOrder: CatalogSortOrder.CatalogSortByRecentlyPublished,
						shouldIncludeHiddenItems: true,
					},
					"",
				),
			)
			return () => {
				this.creations = page.items ?? []
				this.hasMoreCreations = page.nextCursor !== ""
			}
		})
	}

	// loadPart applies a part only while the scope is current, and records a
	// failed part, which the backend logs.
	async #loadPart(
		part: ProfileShowcasePart,
		read: () => Promise<() => void>,
	): Promise<void> {
		try {
			const apply = await read()
			if (this.#scope.isCurrent) apply()
		} catch {
			if (this.#scope.isCurrent) this.failedParts = [...this.failedParts, part]
		}
	}

	dispose(): void {
		this.#scope.dispose()
	}
}

// The showcases of the profiles shown last. The Accounts page remounts the
// profile when the selected account settles, so the showcase outlives it.
const keptShowcases = new Map<string, ProfileShowcase>(),
	maxKeptShowcases = 20

// profileShowcaseFor returns the kept showcase of the user's profile as the
// account reads it, and loads a new one when none is reusable.
export function profileShowcaseFor(
	accountId: number,
	userId: number,
	username: string,
	accounts: AccountStore,
): ProfileShowcase {
	const key = `${accountId}:${userId}`,
		kept = keptShowcases.get(key)
	if (kept?.isReusable) return kept
	kept?.dispose()
	keptShowcases.delete(key)
	const showcase = new ProfileShowcase(accountId, userId, username, accounts)
	keptShowcases.set(key, showcase)
	for (const [oldestKey, oldest] of keptShowcases) {
		if (keptShowcases.size <= maxKeptShowcases) break
		oldest.dispose()
		keptShowcases.delete(oldestKey)
	}
	showcase.load()
	return showcase
}

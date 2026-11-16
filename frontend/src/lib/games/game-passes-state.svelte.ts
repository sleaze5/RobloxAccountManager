import { AccountScope } from "../accounts/account-scope"
import type { AccountStore } from "../accounts/account-store.svelte"
import { readRobuxBalance } from "../accounts/robux-balance"
import { accountBackend, InventoryCategory } from "../backend/bridge"
import type { GamePass } from "../backend/bridge"
import { errorMessage } from "../shared/errors"
import { isFresh } from "../shared/freshness"

// UniverseGamePasses holds a universe's game passes as the shown account sees
// them. The store keeps it while the place is selected.
export class UniverseGamePasses {
	gamePasses = $state<GamePass[] | null>(null)
	loading = $state(false)
	error = $state("")
	isBuying = $state(false)
	purchaseError = $state("")
	// undefined while loading, null if unavailable.
	robuxBalance = $state<number | null | undefined>(undefined)
	pendingPurchaseGamePassIds = $state(new Set<number>())
	readonly #accounts: AccountStore
	// The page reads accountId, so the scope is state.
	#scope: AccountScope
	#fetchedAt = 0
	// Lists fetched before or just after a purchase can still read unowned.
	#boughtGamePassIds = new Set<number>()
	#listRequest: ReturnType<typeof accountBackend.ListGamePasses> | null = null

	constructor(
		readonly universeId: number,
		accounts: AccountStore,
	) {
		this.#accounts = accounts
		this.#scope = $state.raw(
			new AccountScope(accounts.selectedAccount.id, accounts),
		)
	}

	// hasOwnershipFor reports whether isOwned is loaded for the account.
	hasOwnershipFor(accountId: number): boolean {
		return (
			accountId !== 0 &&
			this.#scope.accountId === accountId &&
			this.gamePasses !== null
		)
	}

	// showAccount reloads for another account. Only the open page calls it.
	showAccount(accountId: number): void {
		if (accountId === this.#scope.accountId) return
		this.#scope.dispose()
		this.#scope = new AccountScope(accountId, this.#accounts)
		this.#listRequest = null
		this.#fetchedAt = 0
		this.gamePasses = null
		this.loading = this.isBuying = false
		this.error = this.purchaseError = ""
		this.pendingPurchaseGamePassIds = new Set()
		this.#boughtGamePassIds = new Set()
		void this.refresh()
	}

	// load shows kept passes and refetches stale ones.
	async load(): Promise<void> {
		if (this.#listRequest || (this.gamePasses && isFresh(this.#fetchedAt))) return
		await this.refresh()
	}

	async refresh(): Promise<void> {
		void this.#listRequest?.cancel()
		const scope = this.#scope,
			request = accountBackend.ListGamePasses(scope.accountId, this.universeId)
		this.#listRequest = request
		this.loading = true
		this.error = ""
		try {
			const gamePasses = await scope.track(request)
			if (this.#listRequest !== request || !scope.isCurrent) return
			const listedGamePasses = gamePasses ?? []
			this.#endDeliveredPurchases(listedGamePasses)
			for (const gamePass of listedGamePasses) {
				if (this.#boughtGamePassIds.has(gamePass.gamePassId))
					gamePass.isOwned = true
			}
			this.gamePasses = listedGamePasses
			this.#fetchedAt = Date.now()
		} catch (error) {
			if (this.#listRequest === request && scope.isCurrent) {
				this.error = errorMessage(
					error,
					"Couldn't load game passes. Try again.",
				)
			}
		} finally {
			if (this.#listRequest === request) {
				this.#listRequest = null
				this.loading = false
			}
		}
	}

	// Pending ends once Roblox lists the pass as owned.
	#endDeliveredPurchases(listedGamePasses: GamePass[]): void {
		const ownedIds = new Set(
				listedGamePasses
					.filter((gamePass) => gamePass.isOwned)
					.map((gamePass) => gamePass.gamePassId),
			),
			pendingIds = [...this.pendingPurchaseGamePassIds]
		if (pendingIds.some((gamePassId) => ownedIds.has(gamePassId))) {
			this.pendingPurchaseGamePassIds = new Set(
				pendingIds.filter((gamePassId) => !ownedIds.has(gamePassId)),
			)
		}
	}

	async loadRobuxBalance(): Promise<void> {
		const scope = this.#scope
		this.robuxBalance = undefined
		const robuxBalance = await readRobuxBalance(scope.accountId, (request) =>
			scope.track(request),
		)
		if (scope.isCurrent) this.robuxBalance = robuxBalance
	}

	clearPurchaseError(): void {
		this.purchaseError = ""
	}

	// buy buys at the confirmed price for the confirmed account and reports
	// success.
	async buy(gamePass: GamePass, accountId: number): Promise<boolean> {
		const scope = this.#scope
		if (this.isBuying || !accountId || accountId !== scope.accountId) {
			return false
		}
		this.isBuying = true
		this.purchaseError = ""
		try {
			const isPending = await scope.track(
				accountBackend.BuyInventoryItem(
					scope.accountId,
					InventoryCategory.InventoryGamePasses,
					gamePass.gamePassId,
					gamePass.priceInRobux,
				),
			)
			if (!scope.isCurrent) return false
			this.#boughtGamePassIds.add(gamePass.gamePassId)
			this.gamePasses =
				this.gamePasses?.map((listedGamePass) =>
					listedGamePass.gamePassId === gamePass.gamePassId
						? { ...listedGamePass, isOwned: true }
						: listedGamePass,
				) ?? null
			if (isPending) {
				this.pendingPurchaseGamePassIds = new Set(
					this.pendingPurchaseGamePassIds,
				).add(gamePass.gamePassId)
			}
			return true
		} catch (error) {
			if (scope.isCurrent) {
				this.purchaseError = errorMessage(
					error,
					"Couldn't buy the game pass. Try again.",
				)
			}
			return false
		} finally {
			if (scope === this.#scope) this.isBuying = false
		}
	}

	dispose(): void {
		this.#listRequest = null
		this.#scope.dispose()
	}
}

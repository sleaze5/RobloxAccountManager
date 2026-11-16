import { AccountScope } from "../accounts/account-scope"
import type { AccountStore } from "../accounts/account-store.svelte"
import { readRobuxBalance } from "../accounts/robux-balance"
import { accountBackend, InventoryCategory } from "../backend/bridge"
import type { GamePass } from "../backend/bridge"
import { errorMessage } from "../shared/errors"
import { isFresh } from "../shared/freshness"

// UniverseGamePasses holds the game passes of one universe as the shown
// account sees them, because Roblox reports ownership per account. The games
// store keeps it while the place stays selected, so that the Game passes page
// reopens at once.
export class UniverseGamePasses {
	gamePasses = $state<GamePass[] | null>(null)
	loading = $state(false)
	error = $state("")
	isBuying = $state(false)
	purchaseError = $state("")
	// undefined while the balance loads, and null when it is unavailable.
	robuxBalance = $state<number | null | undefined>(undefined)
	pendingPurchaseGamePassIds = $state(new Set<number>())
	readonly #accounts: AccountStore
	// The page reads accountId, so the scope is state.
	#scope: AccountScope
	#fetchedAt = 0
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

	get accountId(): number {
		return this.#scope.accountId
	}

	// showAccount lists the game passes as another account sees them. It does
	// nothing for the shown account, so that a failed request is retried only
	// on request instead of each time the account list refreshes.
	showAccount(accountId: number): void {
		if (accountId === this.#scope.accountId) return
		const wasRequested = this.gamePasses !== null || this.#listRequest !== null
		this.#scope.dispose()
		this.#scope = new AccountScope(accountId, this.#accounts)
		this.#listRequest = null
		this.#fetchedAt = 0
		this.gamePasses = null
		this.loading = this.isBuying = false
		this.error = this.purchaseError = ""
		this.pendingPurchaseGamePassIds = new Set()
		if (wasRequested) void this.refresh()
	}

	// load shows the kept game passes at once and requests them again when they
	// are no longer fresh.
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
			this.gamePasses = gamePasses ?? []
			this.#fetchedAt = Date.now()
		} catch (error) {
			if (this.#listRequest === request && scope.isCurrent) {
				this.error = errorMessage(
					error,
					"Game passes could not be loaded. Try again.",
				)
			}
		} finally {
			if (this.#listRequest === request) {
				this.#listRequest = null
				this.loading = false
			}
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

	// buy buys the game pass for the shown account at the price that the user
	// confirmed, and reports whether Roblox sold it.
	async buy(gamePass: GamePass): Promise<boolean> {
		const scope = this.#scope
		if (this.isBuying || !scope.accountId) return false
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
					"Could not buy the game pass. Try again.",
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

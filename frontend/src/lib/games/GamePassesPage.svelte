<script lang="ts">
	import caretLeftIcon from "@phosphor-icons/core/regular/caret-left.svg?raw"
	import checkCircleFillIcon from "@phosphor-icons/core/fill/check-circle-fill.svg?raw"
	import clockIcon from "@phosphor-icons/core/regular/clock.svg?raw"
	import funnelIcon from "@phosphor-icons/core/regular/funnel.svg?raw"
	import prohibitIcon from "@phosphor-icons/core/regular/prohibit.svg?raw"
	import ticketIcon from "@phosphor-icons/core/regular/ticket.svg?raw"
	import userPlusIcon from "@phosphor-icons/core/regular/user-plus.svg?raw"
	import warningIcon from "@phosphor-icons/core/regular/warning.svg?raw"
	import { onMount, tick, untrack } from "svelte"
	import type { AccountStore } from "../accounts/account-store.svelte"
	import type { GamePass } from "../backend/bridge"
	import { salePriceLabel } from "../inventory/inventory-prices"
	import { itemGridClass } from "../inventory/item-grid"
	import Icon from "../shared/Icon.svelte"
	import ItemTile from "../shared/ItemTile.svelte"
	import type { ItemTileBadge } from "../shared/ItemTile.svelte"
	import RefreshIcon from "../shared/RefreshIcon.svelte"
	import SkeletonTile from "../shared/SkeletonTile.svelte"
	import DeveloperProductList from "./DeveloperProductList.svelte"
	import GamePassDialog from "./GamePassDialog.svelte"
	import type { GamesStore, StoreView } from "./games-store.svelte"

	let {
		store,
		accounts,
		onBack,
	}: {
		store: GamesStore
		accounts: AccountStore
		onBack: () => void
	} = $props()

	type GamePassFilter = "all" | "owned" | "unowned"

	const place = untrack(() => store.selected!),
		universeGamePasses = untrack(() =>
			store.gamePassesFor(place.universeId, accounts),
		),
		universeDeveloperProducts = untrack(() =>
			store.developerProductsFor(place.universeId),
		),
		filterID = $props.id(),
		views: { id: StoreView; label: string; tooltip: string }[] = [
			{
				id: "gamePasses",
				label: "Game passes",
				tooltip: "Passes that an account buys once and keeps",
			},
			{
				id: "developerProducts",
				label: "Developer products",
				tooltip: "Items that players buy inside the game",
			},
		],
		numbers = new Intl.NumberFormat(),
		skeletonTileIndexes = Array.from({ length: 12 }, (_, index) => index)
	let heading = $state<HTMLHeadingElement | undefined>(undefined),
		list = $state<HTMLUListElement | undefined>(undefined),
		filter = $state<GamePassFilter>("all"),
		shown = $state<GamePass | null>(null)
	const view = $derived(store.storeView),
		isRefreshing = $derived(
			view === "gamePasses"
				? universeGamePasses.loading
				: universeDeveloperProducts.loading,
		),
		account = $derived(accounts.selectedAccount),
		hasOwnership = $derived(universeGamePasses.hasOwnershipFor(account.id)),
		gamePasses = $derived(universeGamePasses.gamePasses ?? []),
		ownedCount = $derived(gamePasses.filter((gamePass) => gamePass.isOwned).length),
		shownGamePasses = $derived(
			!hasOwnership || filter === "all"
				? gamePasses
				: gamePasses.filter(
						(gamePass) => gamePass.isOwned === (filter === "owned"),
					),
		),
		filterOptions: { id: GamePassFilter; label: string }[] = [
			{ id: "all", label: "All" },
			{ id: "owned", label: "Owned" },
			{ id: "unowned", label: "Not owned" },
		]

	// Reads the ID alone, so it runs only on account change, before the first
	// load.
	const accountId = $derived(account.id)
	$effect(() => {
		const shownAccountId = accountId
		untrack(() => {
			shown = null
			filter = "all"
			universeGamePasses.showAccount(shownAccountId)
		})
	})

	onMount(() => {
		heading?.focus({ preventScroll: true })
	})

	$effect(() => {
		const shownView = view
		untrack(() => {
			if (shownView === "gamePasses") void universeGamePasses.load()
			else void universeDeveloperProducts.load()
		})
	})

	function refresh(): void {
		if (view === "gamePasses") void universeGamePasses.refresh()
		else void universeDeveloperProducts.refresh()
	}

	// Scroll the pass opened from Inventory into view once, when the list first
	// loads. The page remounts on each navigation from Inventory.
	let hasRevealedHighlight = false
	$effect(() => {
		const highlightedGamePassId = store.highlightedGamePassId
		if (
			hasRevealedHighlight ||
			!highlightedGamePassId ||
			!universeGamePasses.gamePasses
		) {
			return
		}
		hasRevealedHighlight = true
		untrack(async () => {
			filter = "all"
			await tick()
			list?.querySelector(
				`[data-game-pass="${highlightedGamePassId}"]`,
			)?.scrollIntoView({ block: "center" })
		})
	})

	// The marks of a pass's tile, in priority order.
	function gamePassBadges(gamePass: GamePass): ItemTileBadge[] {
		const badges: ItemTileBadge[] = []
		if (hasOwnership) {
			if (
				universeGamePasses.pendingPurchaseGamePassIds.has(gamePass.gamePassId)
			) {
				badges.push({
					label: "Still being delivered",
					icon: clockIcon,
					tone: "text-fg-muted",
				})
			} else if (gamePass.isOwned) {
				badges.push({
					label: "You own this",
					icon: checkCircleFillIcon,
					tone: "text-success",
				})
			}
		}
		if (gamePass.isModerated) {
			badges.push({
				label: "Deleted by Roblox",
				icon: prohibitIcon,
				tone: "text-warning",
			})
		}
		return badges
	}

	// A refresh can drop the pass, so focus falls back to the heading.
	async function closeGamePass(gamePassId: number): Promise<void> {
		shown = null
		await tick()
		const tile = list?.querySelector<HTMLButtonElement>(
			`[data-game-pass="${gamePassId}"] button`,
		)
		if (tile) tile.focus()
		else heading?.focus({ preventScroll: true })
	}
</script>

<section class="settings-page min-w-0" aria-labelledby="game-passes-title">
	<header class="settings-header">
		<button
			class="button-ghost"
			type="button"
			aria-label="Back to game details"
			data-tooltip="Back to game details"
			onclick={onBack}>
			<Icon svg={caretLeftIcon} data-nudge="left" size={16} aria-hidden="true" />
		</button>
		<h1 id="game-passes-title" tabindex="-1" bind:this={heading}>
			{view === "gamePasses" ? "Game passes" : "Developer products"}
		</h1>
		<span class="mr-auto min-w-0 text-xs wrap-anywhere text-fg-muted"
			>{store.displayName(place)}</span>
		<div class="segmented" role="radiogroup" aria-label="Store list">
			{#each views as option (option.id)}
				<label data-tooltip={option.tooltip} data-tooltip-side="bottom-end">
					<input
						type="radio"
						name={`${filterID}-view`}
						value={option.id}
						checked={view === option.id}
						onchange={() => (store.storeView = option.id)} />
					<span>{option.label}</span>
				</label>
			{/each}
		</div>
		<button
			class="button-ghost"
			type="button"
			aria-label={isRefreshing
				? view === "gamePasses"
					? "Refreshing game passes…"
					: "Refreshing developer products…"
				: view === "gamePasses"
					? "Refresh game passes"
					: "Refresh developer products"}
			data-tooltip={view === "gamePasses"
				? "Refresh game passes"
				: "Refresh developer products"}
			data-tooltip-side="bottom-end"
			aria-busy={isRefreshing}
			disabled={isRefreshing}
			onclick={refresh}>
			<RefreshIcon {isRefreshing} />
		</button>
	</header>

	{#if view === "developerProducts"}
		<DeveloperProductList
			{universeDeveloperProducts}
			gameName={store.displayName(place)} />
	{:else if account.id !== 0}
		<div
			class="flex flex-wrap items-end gap-x-3 gap-y-2 border-b border-line-subtle p-3"
			role="group"
			aria-label="Game pass filters">
			<div class="flex flex-col gap-1 text-2xs text-fg-muted">
				<span id={`${filterID}-filter`}>Show</span>
				<div
					class="segmented"
					role="radiogroup"
					aria-labelledby={`${filterID}-filter`}>
					{#each filterOptions as option (option.id)}
						<label>
							<input
								type="radio"
								name={`${filterID}-filter`}
								value={option.id}
								checked={filter === option.id}
								disabled={!hasOwnership}
								onchange={() => (filter = option.id)} />
							<span>{option.label}</span>
						</label>
					{/each}
				</div>
			</div>
			{#if hasOwnership && gamePasses.length > 0}
				<p class="ml-auto font-mono text-2xs text-fg-muted" role="status">
					<strong class="font-semibold text-fg"
						>{numbers.format(ownedCount)}</strong>
					of {numbers.format(gamePasses.length)} owned
				</p>
			{/if}
		</div>
	{/if}

	{#if view === "gamePasses"}
		<div class="@container min-h-0 flex-1 overflow-auto">
			{#if universeGamePasses.error && !universeGamePasses.gamePasses}
				<div class="empty-state h-full" role="alert">
					<Icon svg={warningIcon} size={28} aria-hidden="true" />
					<h2>Game passes unavailable</h2>
					<p>{universeGamePasses.error}</p>
					<button
						class="button"
						type="button"
						onclick={() => void universeGamePasses.refresh()}
						>Try again</button>
				</div>
			{:else if !universeGamePasses.gamePasses}
				<div class="p-3" role="status" aria-busy="true">
					<span class="sr-only">Loading game passes…</span>
					<ul class={itemGridClass} aria-hidden="true">
						{#each skeletonTileIndexes as index (index)}
							<SkeletonTile />
						{/each}
					</ul>
				</div>
			{:else if gamePasses.length === 0}
				<div class="empty-state h-full" role="status">
					<Icon svg={ticketIcon} size={28} aria-hidden="true" />
					<h2>No game passes</h2>
				</div>
			{:else if shownGamePasses.length === 0}
				<div class="empty-state h-full" role="status">
					<Icon svg={funnelIcon} size={28} aria-hidden="true" />
					<h2>
						{filter === "owned"
							? "No owned game passes"
							: "All game passes owned"}
					</h2>
					<button
						class="button"
						type="button"
						onclick={() => (filter = "all")}>Show all</button>
				</div>
			{:else}
				{#if universeGamePasses.error}
					<div
						class="mx-3 mt-3 flex items-center gap-3 notice-error"
						role="alert">
						<span class="flex-1">{universeGamePasses.error}</span>
						<button
							class="button"
							type="button"
							onclick={() => void universeGamePasses.refresh()}
							>Try again</button>
					</div>
				{/if}
				{#if account.id === 0}
					<p
						class="flex items-center gap-2 border-b border-line-subtle px-3 py-2 text-xs text-fg-muted">
						<Icon svg={userPlusIcon} size={14} aria-hidden="true" />
						Add an account to buy game passes.
					</p>
				{/if}
				<ul
					class={[itemGridClass, "p-3"]}
					aria-label="Game passes"
					aria-busy={universeGamePasses.loading}
					bind:this={list}>
					{#each shownGamePasses as gamePass (gamePass.gamePassId)}
						<li class="min-w-0" data-game-pass={gamePass.gamePassId}>
							<ItemTile
								name={gamePass.name || "Unnamed game pass"}
								imageUrl={gamePass.iconUrl}
								fallbackIcon={ticketIcon}
								badges={gamePassBadges(gamePass)}
								priceLabel={salePriceLabel(gamePass, true)}
								isPriceMuted={!gamePass.isForSale}
								isSelected={store.highlightedGamePassId ===
									gamePass.gamePassId}
								actionLabel="Show details"
								onOpen={() => (shown = gamePass)} />
						</li>
					{/each}
				</ul>
			{/if}
		</div>
	{/if}
</section>

{#if shown}
	{@const gamePassId = shown.gamePassId}
	<GamePassDialog
		{universeGamePasses}
		gamePass={shown}
		gameName={store.displayName(place)}
		{account}
		onClose={() => void closeGamePass(gamePassId)} />
{/if}

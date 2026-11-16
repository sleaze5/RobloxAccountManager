<script lang="ts">
	import caretLeftIcon from "@phosphor-icons/core/regular/caret-left.svg?raw"
	import checkCircleIcon from "@phosphor-icons/core/regular/check-circle.svg?raw"
	import clockIcon from "@phosphor-icons/core/regular/clock.svg?raw"
	import funnelIcon from "@phosphor-icons/core/regular/funnel.svg?raw"
	import shoppingCartIcon from "@phosphor-icons/core/regular/shopping-cart.svg?raw"
	import ticketIcon from "@phosphor-icons/core/regular/ticket.svg?raw"
	import userPlusIcon from "@phosphor-icons/core/regular/user-plus.svg?raw"
	import warningIcon from "@phosphor-icons/core/regular/warning.svg?raw"
	import { onMount, tick, untrack } from "svelte"
	import type { AccountStore } from "../accounts/account-store.svelte"
	import type { GamePass } from "../backend/bridge"
	import { formatRobux } from "../inventory/inventory-prices"
	import Icon from "../shared/Icon.svelte"
	import ProfileImage from "../shared/ProfileImage.svelte"
	import RefreshIcon from "../shared/RefreshIcon.svelte"
	import SkeletonList from "../shared/SkeletonList.svelte"
	import SkeletonMediaRow from "../shared/SkeletonMediaRow.svelte"
	import GamePassDialog from "./GamePassDialog.svelte"
	import type { GamesStore } from "./games-store.svelte"

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
		filterID = $props.id(),
		numbers = new Intl.NumberFormat()
	let heading = $state<HTMLHeadingElement | undefined>(undefined),
		list = $state<HTMLOListElement | undefined>(undefined),
		filter = $state<GamePassFilter>("all"),
		shown = $state<{
			gamePass: GamePass
			openedView: "details" | "purchase"
		} | null>(null)
	const account = $derived(accounts.selectedAccount),
		// Ownership belongs to the account that the list was read for, so it
		// shows only once the list of the selected account arrives.
		hasOwnership = $derived(
			account.id !== 0 &&
				universeGamePasses.accountId === account.id &&
				!!universeGamePasses.gamePasses,
		),
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

	// The account list replaces its accounts when it refreshes, so the effect
	// reads the ID alone and runs only when another account is selected. It runs
	// before the first load, so that the load requests the shown account.
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
		void universeGamePasses.load()
	})

	// The pass that Inventory opened scrolls into view once the list shows it.
	$effect(() => {
		const highlightedGamePassId = store.highlightedGamePassId
		if (!highlightedGamePassId || !universeGamePasses.gamePasses) return
		untrack(async () => {
			filter = "all"
			await tick()
			list?.querySelector(
				`[data-game-pass="${highlightedGamePassId}"]`,
			)?.scrollIntoView({ block: "center" })
		})
	})

	// A purchase can replace the Buy button of the pass, so focus returns to
	// the details button of the pass, which stays.
	async function closeGamePass(gamePassId: number): Promise<void> {
		shown = null
		await tick()
		const detailsButton = list?.querySelector<HTMLButtonElement>(
			`[data-game-pass="${gamePassId}"] [data-show-game-pass]`,
		)
		if (detailsButton) detailsButton.focus()
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
		<h1 id="game-passes-title" tabindex="-1" bind:this={heading}>Game passes</h1>
		<span class="mr-auto min-w-0 text-xs wrap-anywhere text-fg-muted"
			>{store.displayName(place)}</span>
		<button
			class="button-ghost"
			type="button"
			aria-label={universeGamePasses.loading
				? "Refreshing game passes"
				: "Refresh game passes"}
			data-tooltip="Refresh game passes"
			data-tooltip-side="bottom-end"
			aria-busy={universeGamePasses.loading}
			disabled={universeGamePasses.loading}
			onclick={() => void universeGamePasses.refresh()}>
			<RefreshIcon isRefreshing={universeGamePasses.loading} />
		</button>
	</header>

	{#if account.id !== 0}
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
					@{account.username} owns
					<strong class="font-semibold text-fg"
						>{numbers.format(ownedCount)}</strong>
					of {numbers.format(gamePasses.length)}
				</p>
			{/if}
		</div>
	{/if}

	<div class="min-h-0 flex-1 overflow-auto">
		{#if universeGamePasses.error && !universeGamePasses.gamePasses}
			<div class="empty-state h-full" role="alert">
				<Icon svg={warningIcon} size={28} aria-hidden="true" />
				<h2>Game passes are unavailable</h2>
				<p>{universeGamePasses.error}</p>
				<button
					class="button"
					type="button"
					onclick={() => void universeGamePasses.refresh()}>Try again</button>
			</div>
		{:else if !universeGamePasses.gamePasses}
			<SkeletonList label="Loading game passes">
				{#snippet row(index)}<SkeletonMediaRow
						{index}
						className="h-16" />{/snippet}
			</SkeletonList>
		{:else if gamePasses.length === 0}
			<div class="empty-state h-full" role="status">
				<Icon svg={ticketIcon} size={28} aria-hidden="true" />
				<h2>No game passes</h2>
				<p>This game has no game passes for sale.</p>
			</div>
		{:else if shownGamePasses.length === 0}
			<div class="empty-state h-full" role="status">
				<Icon svg={funnelIcon} size={28} aria-hidden="true" />
				<h2>
					{filter === "owned"
						? "No owned game passes"
						: "All game passes owned"}
				</h2>
				<p>
					{filter === "owned"
						? `@${account.username} doesn't own a game pass in this game yet.`
						: `@${account.username} owns every game pass in this game.`}
				</p>
				<button class="button" type="button" onclick={() => (filter = "all")}
					>Show all game passes</button>
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
					Add an account on the Accounts page to buy game passes.
				</p>
			{/if}
			<ol
				aria-label="Game passes"
				aria-busy={universeGamePasses.loading}
				bind:this={list}>
				{#each shownGamePasses as gamePass (gamePass.gamePassId)}
					{@const name = gamePass.name || "Unnamed game pass"}
					{@const isPending =
						universeGamePasses.pendingPurchaseGamePassIds.has(
							gamePass.gamePassId,
						)}
					<li
						class="list-row flex min-h-16 items-center border-b border-line-subtle"
						data-game-pass={gamePass.gamePassId}
						data-selected={store.highlightedGamePassId ===
							gamePass.gamePassId || undefined}
						data-open={shown?.gamePass.gamePassId === gamePass.gamePassId ||
							undefined}>
						<button
							class="flex min-w-0 flex-1 items-center gap-3 self-stretch px-3 py-2 text-left"
							type="button"
							aria-label={`Show details of ${name}`}
							data-show-game-pass
							onclick={() =>
								(shown = { gamePass, openedView: "details" })}>
							<span
								class={[
									"grid size-9 shrink-0 place-items-center overflow-hidden rounded-md border border-line-strong bg-raised",
									!gamePass.isForSale &&
										!gamePass.isOwned &&
										"opacity-45",
								]}
								aria-hidden="true">
								<ProfileImage
									url={gamePass.iconUrl}
									fallbackIcon={ticketIcon} />
							</span>
							<span class="flex min-w-0 flex-1 flex-col gap-0.5">
								<strong
									class="flex min-w-0 items-center gap-1.5 text-xs font-semibold text-fg">
									<span class="truncate">{name}</span>
								</strong>
								{#if gamePass.description}
									<span class="truncate text-xs text-fg-muted"
										>{gamePass.description}</span>
								{/if}
								<span class="font-mono text-2xs text-fg-faint"
									>{gamePass.isForSale
										? gamePass.priceInRobux === 0
											? "Free"
											: formatRobux(gamePass.priceInRobux, false)
										: "Off sale"}</span>
							</span>
						</button>
						{#if hasOwnership}
							<div class="flex shrink-0 items-center pr-3 text-xs">
								{#if isPending}
									<span
										class="inline-flex items-center gap-1 font-medium text-fg-muted"
										data-tooltip="Roblox is still delivering this game pass"
										data-tooltip-side="bottom-end">
										<Icon
											svg={clockIcon}
											size={14}
											aria-hidden="true" />
										Pending
									</span>
								{:else if gamePass.isOwned}
									<span
										class="inline-flex animate-pop items-center gap-1 font-medium text-success">
										<Icon
											svg={checkCircleIcon}
											size={14}
											aria-hidden="true" />
										Owned
									</span>
								{:else if gamePass.isForSale}
									<button
										class="button"
										type="button"
										aria-label={gamePass.priceInRobux === 0
											? `Get ${name}`
											: `Buy ${name} for ${formatRobux(gamePass.priceInRobux, false)}`}
										disabled={universeGamePasses.isBuying}
										onclick={() =>
											(shown = {
												gamePass,
												openedView: "purchase",
											})}>
										<Icon
											svg={shoppingCartIcon}
											size={16}
											aria-hidden="true" />
										{gamePass.priceInRobux === 0 ? "Get" : "Buy"}
									</button>
								{:else}
									<span class="text-fg-muted">Not owned</span>
								{/if}
							</div>
						{/if}
					</li>
				{/each}
			</ol>
		{/if}
	</div>
</section>

{#if shown}
	{@const gamePassId = shown.gamePass.gamePassId}
	<GamePassDialog
		{universeGamePasses}
		gamePass={shown.gamePass}
		gameName={store.displayName(place)}
		{account}
		openedView={shown.openedView}
		onClose={() => void closeGamePass(gamePassId)} />
{/if}

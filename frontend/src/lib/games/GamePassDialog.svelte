<script lang="ts">
	import calendarDotsIcon from "@phosphor-icons/core/regular/calendar-dots.svg?raw"
	import checkCircleFillIcon from "@phosphor-icons/core/fill/check-circle-fill.svg?raw"
	import clockCounterClockwiseIcon from "@phosphor-icons/core/regular/clock-counter-clockwise.svg?raw"
	import shoppingCartIcon from "@phosphor-icons/core/regular/shopping-cart.svg?raw"
	import storefrontIcon from "@phosphor-icons/core/regular/storefront.svg?raw"
	import ticketIcon from "@phosphor-icons/core/regular/ticket.svg?raw"
	import xIcon from "@phosphor-icons/core/regular/x.svg?raw"
	import { onMount, tick } from "svelte"
	import type { Account } from "../accounts/account-model"
	import type { GamePass } from "../backend/bridge"
	import { formatRobux } from "../inventory/inventory-prices"
	import PurchaseAccountActions from "../inventory/PurchaseAccountActions.svelte"
	import PurchaseItemSummary from "../inventory/PurchaseItemSummary.svelte"
	import Dialog from "../shared/Dialog.svelte"
	import Icon from "../shared/Icon.svelte"
	import ProfileImage from "../shared/ProfileImage.svelte"
	import Timestamp from "../shared/Timestamp.svelte"
	import { relativeTime } from "../shared/timestamp"
	import type { UniverseGamePasses } from "./game-passes-state.svelte"

	type GamePassView = "details" | "purchase"

	let {
		universeGamePasses,
		gamePass: openedGamePass,
		gameName,
		account,
		openedView,
		onClose,
	}: {
		universeGamePasses: UniverseGamePasses
		gamePass: GamePass
		gameName: string
		// account is the account that buys the game pass.
		account: Account
		// openedView is purchase when the Buy button of the list opened the
		// dialog, which then closes when the purchase ends.
		openedView: GamePassView
		onClose: () => void
	} = $props()

	const id = $props.id(),
		metaClass =
			"flex min-w-0 flex-wrap items-center gap-1 font-mono text-2xs wrap-anywhere text-fg-muted",
		sectionHeadingClass = "text-xs font-semibold text-fg-muted"
	let view = $state<GamePassView>("details"),
		card = $state<HTMLDivElement | undefined>(undefined)
	// The list replaces a pass when it is bought, so the dialog shows the
	// listed copy.
	const gamePass = $derived(
			universeGamePasses.gamePasses?.find(
				(listedGamePass) =>
					listedGamePass.gamePassId === openedGamePass.gamePassId,
			) ?? openedGamePass,
		),
		name = $derived(gamePass.name || "Unnamed game pass"),
		price = $derived(gamePass.priceInRobux),
		hasOwnership = $derived(
			account.id !== 0 && universeGamePasses.accountId === account.id,
		),
		isPending = $derived(
			universeGamePasses.pendingPurchaseGamePassIds.has(gamePass.gamePassId),
		),
		canBuy = $derived(hasOwnership && !gamePass.isOwned && gamePass.isForSale),
		priceLabel = $derived(
			gamePass.isForSale
				? price === 0
					? "Free"
					: formatRobux(price, false)
				: "Off sale",
		)

	onMount(() => {
		void showView(openedView)
	})

	// The dialog keeps its card while it switches between the details and the
	// purchase, so focus moves to the control that replaced the one the user
	// pressed. The balance is read each time the purchase opens, because Robux
	// can be bought outside the application.
	async function showView(shownView: GamePassView): Promise<void> {
		universeGamePasses.clearPurchaseError()
		view = shownView
		if (shownView === "purchase") void universeGamePasses.loadRobuxBalance()
		await tick()
		const focusTarget =
			card?.querySelector<HTMLButtonElement>(
				shownView === "purchase"
					? "[data-cancel-confirmation]"
					: "[data-begin-purchase]",
			) ?? card?.querySelector<HTMLButtonElement>("[data-close-game-pass]")
		focusTarget?.focus()
	}

	function close(): void {
		if (universeGamePasses.isBuying) return
		if (view === "purchase" && openedView === "details") void showView("details")
		else {
			universeGamePasses.clearPurchaseError()
			onClose()
		}
	}

	async function buy(): Promise<void> {
		if (!(await universeGamePasses.buy(gamePass))) return
		if (openedView === "purchase") onClose()
		else await showView("details")
	}
</script>

<svelte:window
	onkeydown={(event) => {
		if (event.key === "Escape") {
			event.preventDefault()
			close()
		}
	}} />

{#snippet dateFact(icon: string, label: string, valueMs: number)}
	<div>
		<dt><Icon svg={icon} size={16} aria-hidden="true" />{label}</dt>
		<dd>
			{#if valueMs}
				<Timestamp value={valueMs} />
				<small>{relativeTime(valueMs)}</small>
			{:else}
				<span class="text-fg-muted">Unavailable</span>
			{/if}
		</dd>
	</div>
{/snippet}

<Dialog
	class={view === "purchase" ? "max-w-md text-left" : "max-w-lg text-left"}
	role={view === "purchase" ? "alertdialog" : "dialog"}
	aria-labelledby={`${id}-${view}-title`}
	aria-describedby={view === "purchase" ? `${id}-purchase-description` : undefined}
	bind:card
	onBackdropClick={close}>
	{#if view === "purchase"}
		<div class="dialog-heading">
			<div class="min-w-0">
				<h2 id={`${id}-purchase-title`}>
					{price === 0 ? "Get game pass?" : "Buy game pass?"}
				</h2>
				<p id={`${id}-purchase-description`} class="wrap-anywhere">
					{#if price === 0}
						{name} is added to the account below for free.
					{:else}
						{name} is bought with the account below. Roblox doesn't refund Robux
						spent on game passes.
					{/if}
				</p>
			</div>
			<button
				type="button"
				aria-label="Close confirmation"
				disabled={universeGamePasses.isBuying}
				onclick={close}>
				<Icon svg={xIcon} size={14} aria-hidden="true" />
			</button>
		</div>
		<PurchaseItemSummary
			{name}
			detail={`Game pass - ${gameName}`}
			imageUrl={gamePass.iconUrl}
			fallbackIcon={ticketIcon}
			priceInRobux={price} />
		{#if universeGamePasses.purchaseError}
			<p class="mt-3 notice-error" role="alert">
				{universeGamePasses.purchaseError}
			</p>
		{/if}
		<PurchaseAccountActions
			{account}
			priceInRobux={price}
			robuxBalance={universeGamePasses.robuxBalance}
			isBuying={universeGamePasses.isBuying}
			getLabel="Get game pass"
			onCancel={close}
			onBuy={() => void buy()} />
	{:else}
		<div class="flex flex-col gap-4">
			<div class="flex items-start gap-4">
				<span
					class="grid size-28 shrink-0 place-items-center overflow-hidden rounded-lg border border-line-subtle bg-raised"
					aria-hidden="true">
					<ProfileImage url={gamePass.iconUrl} fallbackIcon={ticketIcon} />
				</span>
				<div class="flex min-w-0 flex-1 flex-col gap-1">
					<div class="flex items-start gap-3">
						<h2
							id={`${id}-details-title`}
							class="min-w-0 flex-1 leading-7 tracking-tight wrap-anywhere">
							{name}
						</h2>
						<div class="flex shrink-0 items-center gap-1">
							{#if canBuy}
								<button
									class="button-primary"
									type="button"
									data-begin-purchase
									onclick={() => void showView("purchase")}>
									<Icon
										svg={shoppingCartIcon}
										size={16}
										aria-hidden="true" />
									{price === 0 ? "Get" : "Buy"}
								</button>
							{/if}
							<button
								class="ml-1 button-close"
								type="button"
								data-close-game-pass
								aria-label="Close game pass details"
								onclick={close}>
								<Icon svg={xIcon} size={14} aria-hidden="true" />
							</button>
						</div>
					</div>
					<div class={metaClass}>
						<span class="select-text">ID: {gamePass.gamePassId}</span>
						<span class="text-fg-faint" aria-hidden="true">-</span>
						<span>Game pass</span>
					</div>
					<div class={metaClass}><span>{gameName}</span></div>
					<p
						class={[
							"inline-flex items-center gap-1 text-xs font-medium tabular-nums",
							gamePass.isForSale ? "text-fg" : "text-fg-muted",
						]}>
						<Icon svg={storefrontIcon} size={14} aria-hidden="true" />
						{priceLabel}
					</p>
					{#if hasOwnership && gamePass.isOwned}
						<ul
							class="mt-2 flex min-h-6 flex-wrap gap-1"
							aria-label="Status">
							<li class="tag-pill-success animate-pop">
								<Icon
									svg={checkCircleFillIcon}
									size={12}
									aria-hidden="true" />
								<span>@{account.username} owns this</span>
							</li>
						</ul>
					{/if}
					{#if isPending}
						<p class="text-xs text-fg-muted" role="status">
							Bought. It can take a minute for the game to see it.
						</p>
					{/if}
				</div>
			</div>

			<section class="flex flex-col gap-3" aria-labelledby={`${id}-details`}>
				<h3 id={`${id}-details`} class={sectionHeadingClass}>Details</h3>
				<dl class="facts">
					{@render dateFact(
						clockCounterClockwiseIcon,
						"Updated",
						gamePass.updatedAtMs,
					)}
					{@render dateFact(
						calendarDotsIcon,
						"Created",
						gamePass.createdAtMs,
					)}
				</dl>
			</section>

			<section class="flex flex-col gap-3" aria-labelledby={`${id}-description`}>
				<h3 id={`${id}-description`} class={sectionHeadingClass}>
					Description
				</h3>
				{#if gamePass.description}
					<p class="text-block" data-variant="prose">
						{gamePass.description}
					</p>
				{:else}
					<p class="text-xs text-fg-muted">No description provided.</p>
				{/if}
			</section>
		</div>
	{/if}
</Dialog>

<script lang="ts">
	import { notifyErrors } from "../notifications/error-notifications.svelte"
	import calendarDotsIcon from "@phosphor-icons/core/regular/calendar-dots.svg?raw"
	import checkCircleFillIcon from "@phosphor-icons/core/fill/check-circle-fill.svg?raw"
	import prohibitIcon from "@phosphor-icons/core/regular/prohibit.svg?raw"
	import clockCounterClockwiseIcon from "@phosphor-icons/core/regular/clock-counter-clockwise.svg?raw"
	import globeHemisphereWestIcon from "@phosphor-icons/core/regular/globe-hemisphere-west.svg?raw"
	import shoppingCartIcon from "@phosphor-icons/core/regular/shopping-cart.svg?raw"
	import storefrontIcon from "@phosphor-icons/core/regular/storefront.svg?raw"
	import ticketIcon from "@phosphor-icons/core/regular/ticket.svg?raw"
	import xIcon from "@phosphor-icons/core/regular/x.svg?raw"
	import { onMount, tick, untrack } from "svelte"
	import type { Account } from "../accounts/account-model"
	import type { GamePass } from "../backend/bridge"
	import { salePriceLabel } from "../inventory/inventory-prices"
	import PurchaseAccountActions from "../inventory/PurchaseAccountActions.svelte"
	import PurchaseItemSummary from "../inventory/PurchaseItemSummary.svelte"
	import Dialog from "../shared/Dialog.svelte"
	import Icon from "../shared/Icon.svelte"
	import ProfileImage from "../shared/ProfileImage.svelte"
	import {
		getRobloxPageOpener,
		openInBrowserUnavailableReason,
		robloxGamePassUrl,
	} from "../shared/roblox-pages"
	import Timestamp from "../shared/Timestamp.svelte"
	import { relativeTime } from "../shared/timestamp"
	import type { UniverseGamePasses } from "./game-passes-state.svelte"

	type GamePassView = "details" | "purchase"

	let {
		universeGamePasses,
		gamePass: openedGamePass,
		gameName,
		account,
		onClose,
	}: {
		universeGamePasses: UniverseGamePasses
		gamePass: GamePass
		gameName: string
		account: Account
		onClose: () => void
	} = $props()

	const id = $props.id(),
		metaClass =
			"flex min-w-0 flex-wrap items-center gap-1 font-mono text-2xs wrap-anywhere text-fg-muted",
		sectionHeadingClass = "text-xs font-semibold text-fg-muted"
	const robloxPages = getRobloxPageOpener()
	let view = $state<GamePassView>("details"),
		card = $state<HTMLDivElement | undefined>(undefined),
		// What the open confirmation buys, fixed while it is open.
		confirmation = $state.raw<{ gamePass: GamePass; accountId: number } | null>(
			null,
		),
		changeNotice = $state("")
	// The listed copy, which a refresh or purchase updates. null once a refresh
	// drops the pass.
	const listedGamePass = $derived(
			universeGamePasses.gamePasses?.find(
				(gamePass) => gamePass.gamePassId === openedGamePass.gamePassId,
			) ?? null,
		),
		gamePass = $derived(listedGamePass ?? openedGamePass),
		shownGamePass = $derived(
			view === "purchase" && confirmation ? confirmation.gamePass : gamePass,
		),
		name = $derived(shownGamePass.name || "Unnamed game pass"),
		price = $derived(shownGamePass.priceInRobux),
		hasOwnership = $derived(universeGamePasses.hasOwnershipFor(account.id)),
		isPending = $derived(
			universeGamePasses.pendingPurchaseGamePassIds.has(gamePass.gamePassId),
		),
		canBuy = $derived(
			hasOwnership &&
				!!listedGamePass &&
				!listedGamePass.isOwned &&
				listedGamePass.isForSale,
		),
		priceLabel = $derived(salePriceLabel(gamePass, false))

	onMount(() => {
		void showView("details")
	})

	$effect(() => {
		if (view === "purchase" && !universeGamePasses.isBuying) {
			endStaleConfirmation()
		}
	})

	// endStaleConfirmation returns to details when the account or the listed
	// pass no longer matches the confirmation, and reports whether it did.
	function endStaleConfirmation(): boolean {
		if (!confirmation) return true
		const confirmedGamePass = confirmation.gamePass
		let notice: string
		if (account.id !== confirmation.accountId || listedGamePass?.isOwned) {
			notice = ""
		} else if (!listedGamePass) {
			notice = "This pass is no longer listed."
		} else if (
			listedGamePass.priceInRobux !== confirmedGamePass.priceInRobux ||
			listedGamePass.isForSale !== confirmedGamePass.isForSale
		) {
			notice = "The price changed."
		} else {
			return false
		}
		untrack(() => {
			changeNotice = notice
			void showView("details")
		})
		return true
	}

	// Focus moves to the control that replaced the pressed one. Robux is reread
	// each time, since it can change outside the app.
	async function showView(shownView: GamePassView): Promise<void> {
		universeGamePasses.clearPurchaseError()
		if (shownView === "purchase") {
			confirmation = {
				gamePass: $state.snapshot(gamePass),
				accountId: account.id,
			}
			changeNotice = ""
			void universeGamePasses.loadRobuxBalance()
		}
		view = shownView
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
		if (view === "purchase") void showView("details")
		else {
			universeGamePasses.clearPurchaseError()
			onClose()
		}
	}

	async function buy(): Promise<void> {
		if (!confirmation || endStaleConfirmation()) return
		const { gamePass: confirmedGamePass, accountId } = confirmation
		if (!(await universeGamePasses.buy(confirmedGamePass, accountId))) return
		await showView("details")
	}

	notifyErrors("Game pass not bought", () => universeGamePasses.purchaseError)
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
						This game pass is free.
					{:else}
						Roblox does not refund game passes.
					{/if}
				</p>
			</div>
			<button
				type="button"
				aria-label="Close dialog"
				disabled={universeGamePasses.isBuying}
				onclick={close}>
				<Icon svg={xIcon} size={14} aria-hidden="true" />
			</button>
		</div>
		<PurchaseItemSummary
			{name}
			detail={`Game pass · ${gameName}`}
			imageUrl={shownGamePass.iconUrl}
			fallbackIcon={ticketIcon}
			priceInRobux={price} />
		<PurchaseAccountActions
			{account}
			priceInRobux={price}
			robuxBalance={universeGamePasses.robuxBalance}
			isBuying={universeGamePasses.isBuying}
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
								class="button-icon"
								type="button"
								aria-label="Open in browser"
								data-tooltip={robloxPages.canOpen
									? "Open in browser"
									: openInBrowserUnavailableReason}
								disabled={!robloxPages.canOpen}
								onclick={() =>
									robloxPages.open(
										robloxGamePassUrl(gamePass.gamePassId),
									)}>
								<Icon
									svg={globeHemisphereWestIcon}
									size={16}
									aria-hidden="true" />
							</button>
							<button
								class="ml-1 button-close"
								type="button"
								data-close-game-pass
								aria-label="Close dialog"
								onclick={close}>
								<Icon svg={xIcon} size={14} aria-hidden="true" />
							</button>
						</div>
					</div>
					<div class={metaClass}>
						<span class="select-text">ID: {gamePass.gamePassId}</span>
						<span class="text-fg-faint" aria-hidden="true">·</span>
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
					{#if changeNotice}
						<p class="mt-1 notice-warning" role="status">{changeNotice}</p>
					{/if}
					{#if (hasOwnership && gamePass.isOwned) || gamePass.isModerated}
						<ul
							class="mt-2 flex min-h-6 flex-wrap gap-1"
							aria-label="Status">
							{#if gamePass.isModerated}
								<li class="tag-pill-warning animate-pop">
									<Icon
										svg={prohibitIcon}
										size={12}
										aria-hidden="true" />
									<span>Deleted</span>
								</li>
							{/if}
							{#if hasOwnership && gamePass.isOwned}
								<li class="tag-pill-success animate-pop">
									<Icon
										svg={checkCircleFillIcon}
										size={12}
										aria-hidden="true" />
									<span>Owned</span>
								</li>
							{/if}
						</ul>
					{/if}
					{#if gamePass.isModerated}
						<p class="mt-1 text-xs text-fg-muted">
							Roblox moderated this pass and deleted its icon.
						</p>
					{/if}
					{#if isPending}
						<p class="text-xs text-fg-muted" role="status">
							Bought. It may take a minute to show up.
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

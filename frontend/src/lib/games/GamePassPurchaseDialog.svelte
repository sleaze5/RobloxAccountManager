<script lang="ts">
	import ticketIcon from "@phosphor-icons/core/regular/ticket.svg?raw"
	import xIcon from "@phosphor-icons/core/regular/x.svg?raw"
	import { onMount } from "svelte"
	import type { GamePass } from "../backend/bridge"
	import {
		formatRobux,
		robuxBalanceAfterPurchase,
	} from "../inventory/inventory-prices"
	import RobuxPurchaseSummary from "../inventory/RobuxPurchaseSummary.svelte"
	import Dialog from "../shared/Dialog.svelte"
	import Icon from "../shared/Icon.svelte"
	import type { UniverseGamePasses } from "./game-passes-state.svelte"

	let {
		universeGamePasses,
		gamePass,
		gameName,
		accountUsername,
		onClose,
	}: {
		universeGamePasses: UniverseGamePasses
		gamePass: GamePass
		gameName: string
		accountUsername: string
		onClose: (isBought: boolean) => void
	} = $props()

	const id = $props.id(),
		name = $derived(gamePass.name || "Unnamed game pass"),
		price = $derived(gamePass.priceInRobux),
		balanceAfterPurchase = $derived(
			robuxBalanceAfterPurchase(universeGamePasses.robuxBalance, price),
		),
		isShortOfRobux = $derived(
			price > 0 && balanceAfterPurchase !== null && balanceAfterPurchase < 0,
		)

	onMount(() => {
		universeGamePasses.clearPurchaseError()
		if (price > 0) void universeGamePasses.loadRobuxBalance()
	})

	function close(): void {
		if (universeGamePasses.isBuying) return
		universeGamePasses.clearPurchaseError()
		onClose(false)
	}

	async function buy(): Promise<void> {
		if (await universeGamePasses.buy(gamePass)) onClose(true)
	}
</script>

<svelte:window
	onkeydown={(event) => {
		if (event.key === "Escape") {
			event.preventDefault()
			close()
		}
	}} />

<Dialog
	class="max-w-md text-left"
	role="alertdialog"
	aria-labelledby={`${id}-title`}
	aria-describedby={`${id}-description`}
	onBackdropClick={close}>
	<div class="dialog-heading">
		<div class="min-w-0">
			{#if price === 0}
				<h2 id={`${id}-title`}>Get game pass?</h2>
				<p id={`${id}-description`} class="wrap-anywhere">
					@{accountUsername} gets {name} for free.
				</p>
			{:else}
				<h2 id={`${id}-title`}>Buy game pass?</h2>
				<p id={`${id}-description`} class="wrap-anywhere">
					@{accountUsername} buys {name}. Roblox doesn't refund Robux spent on
					game passes.
				</p>
			{/if}
		</div>
		<button
			type="button"
			aria-label="Close confirmation"
			disabled={universeGamePasses.isBuying}
			onclick={close}>
			<Icon svg={xIcon} size={14} aria-hidden="true" />
		</button>
	</div>
	{#if price > 0}
		<RobuxPurchaseSummary
			{name}
			detail={`Game pass - ${gameName}`}
			imageUrl={gamePass.iconUrl}
			fallbackIcon={ticketIcon}
			priceInRobux={price}
			{accountUsername}
			robuxBalance={universeGamePasses.robuxBalance} />
	{/if}
	{#if universeGamePasses.purchaseError}
		<p class="mt-3 notice-error" role="alert">
			{universeGamePasses.purchaseError}
		</p>
	{/if}
	<div class="dialog-actions">
		<button
			class="button"
			type="button"
			disabled={universeGamePasses.isBuying}
			onclick={close}>Cancel</button>
		<button
			class="button-primary"
			type="button"
			aria-busy={universeGamePasses.isBuying}
			disabled={universeGamePasses.isBuying || isShortOfRobux}
			onclick={() => void buy()}
			>{universeGamePasses.isBuying
				? "Buying…"
				: price === 0
					? "Get game pass"
					: `Buy for ${formatRobux(price, false)}`}</button>
	</div>
</Dialog>

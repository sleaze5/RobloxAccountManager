<script lang="ts">
	import arrowRightIcon from "@phosphor-icons/core/regular/arrow-right.svg?raw"
	import type { Account } from "../accounts/account-model"
	import AccountConfirmationActions from "../shared/AccountConfirmationActions.svelte"
	import Icon from "../shared/Icon.svelte"
	import Skeleton from "../shared/Skeleton.svelte"
	import { formatRobux } from "./inventory-prices"

	let {
		account,
		priceInRobux,
		robuxBalance,
		isBuying,
		onCancel,
		onBuy,
	}: {
		account: Account
		priceInRobux: number
		// undefined while loading, null if unavailable.
		robuxBalance: number | null | undefined
		isBuying: boolean
		onCancel: () => void
		onBuy: () => void
	} = $props()

	// Buy stays enabled on a low balance, which can change outside the app.
	// Roblox's refusal is shown instead.
</script>

<AccountConfirmationActions {account}>
	{#snippet details()}
		{#if robuxBalance === undefined}
			<Skeleton className="w-28 text-2xs" label="Loading Robux…" />
		{:else if robuxBalance === null}
			<span class="truncate text-2xs text-fg-muted">Robux unavailable</span>
		{:else}
			{@const balanceAfterPurchase = robuxBalance - priceInRobux}
			{@const balanceDescription =
				priceInRobux === 0
					? `${formatRobux(robuxBalance, false)} now`
					: `${formatRobux(robuxBalance, false)} now, ${formatRobux(balanceAfterPurchase, false)} after`}
			<span
				class="flex min-w-0 items-center gap-1 font-mono text-2xs text-fg-muted"
				data-tooltip={balanceDescription}>
				<span class="sr-only">{balanceDescription}</span>
				<span aria-hidden="true">{formatRobux(robuxBalance, false)}</span>
				{#if priceInRobux > 0}
					<Icon
						svg={arrowRightIcon}
						class="shrink-0 text-fg-faint"
						size={12}
						aria-hidden="true" />
					<span
						class={[
							"font-semibold",
							balanceAfterPurchase < 0 ? "text-danger-muted" : "text-fg",
						]}
						aria-hidden="true"
						>{formatRobux(balanceAfterPurchase, false)}</span>
				{/if}
			</span>
		{/if}
	{/snippet}
	<button
		class="button"
		type="button"
		data-cancel-confirmation
		disabled={isBuying}
		onclick={onCancel}>Cancel</button>
	<button
		class="button-primary"
		type="button"
		aria-busy={isBuying}
		disabled={isBuying}
		onclick={onBuy}
		>{isBuying
			? "Buying…"
			: priceInRobux === 0
				? "Get"
				: `Buy for ${formatRobux(priceInRobux, false)}`}</button>
</AccountConfirmationActions>

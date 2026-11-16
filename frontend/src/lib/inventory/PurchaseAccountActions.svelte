<script lang="ts">
	import arrowRightIcon from "@phosphor-icons/core/regular/arrow-right.svg?raw"
	import AccountAvatar from "../accounts/AccountAvatar.svelte"
	import type { Account } from "../accounts/account-model"
	import Icon from "../shared/Icon.svelte"
	import Skeleton from "../shared/Skeleton.svelte"
	import { formatRobux } from "./inventory-prices"

	let {
		account,
		priceInRobux,
		robuxBalance,
		isBuying,
		getLabel,
		onCancel,
		onBuy,
	}: {
		// account is the account that pays.
		account: Account
		priceInRobux: number
		// undefined while the balance loads, and null when it is unavailable.
		robuxBalance: number | null | undefined
		isBuying: boolean
		// getLabel names the action for a free item, such as Get item.
		getLabel: string
		onCancel: () => void
		onBuy: () => void
	} = $props()

	// Buy stays enabled when the balance looks too low, because the balance
	// can change outside the application. Roblox refuses the purchase then, and
	// the dialog shows why.
	const balanceAfterPurchase = $derived(
			typeof robuxBalance === "number" ? robuxBalance - priceInRobux : null,
		),
		balanceDescription = $derived(
			typeof robuxBalance !== "number" || balanceAfterPurchase === null
				? ""
				: priceInRobux === 0
					? `@${account.username} has ${formatRobux(robuxBalance, false)}`
					: `@${account.username} has ${formatRobux(robuxBalance, false)}, and ${formatRobux(balanceAfterPurchase, false)} after this purchase`,
		)
</script>

<div
	class="mt-3 flex items-center justify-between gap-3 border-t border-line-subtle pt-3">
	<div class="flex min-w-0 items-center gap-2">
		<AccountAvatar {account} className="size-7" />
		<div class="flex min-w-0 flex-col leading-tight">
			<span class="truncate text-xs font-medium text-fg"
				>@{account.username}</span>
			{#if robuxBalance === undefined}
				<Skeleton className="w-28 text-2xs" label="Loading the Robux balance" />
			{:else if robuxBalance === null || balanceAfterPurchase === null}
				<span class="truncate text-2xs text-fg-muted">Robux unavailable</span>
			{:else}
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
								balanceAfterPurchase < 0
									? "text-danger-muted"
									: "text-fg",
							]}
							aria-hidden="true"
							>{formatRobux(balanceAfterPurchase, false)}</span>
					{/if}
				</span>
			{/if}
		</div>
	</div>
	<div class="flex shrink-0 gap-2">
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
					? getLabel
					: `Buy for ${formatRobux(priceInRobux, false)}`}</button>
	</div>
</div>

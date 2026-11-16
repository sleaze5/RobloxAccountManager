<script lang="ts">
	import ProfileImage from "../shared/ProfileImage.svelte"
	import Skeleton from "../shared/Skeleton.svelte"
	import { formatRobux, robuxBalanceAfterPurchase } from "./inventory-prices"

	let {
		name,
		detail,
		imageUrl,
		fallbackIcon,
		priceInRobux,
		accountUsername,
		robuxBalance,
	}: {
		name: string
		detail: string
		imageUrl: string
		fallbackIcon: string
		priceInRobux: number
		accountUsername: string
		// undefined while the balance loads, and null when it is unavailable.
		robuxBalance: number | null | undefined
	} = $props()

	const balanceAfterPurchase = $derived(
			robuxBalanceAfterPurchase(robuxBalance, priceInRobux),
		),
		isShortOfRobux = $derived(
			balanceAfterPurchase !== null && balanceAfterPurchase < 0,
		)
</script>

<div class="flex items-center gap-3 rounded-lg border border-line-subtle bg-fill p-2">
	<span
		class="grid size-12 shrink-0 place-items-center overflow-hidden rounded-md bg-raised"
		aria-hidden="true">
		<ProfileImage url={imageUrl} {fallbackIcon} />
	</span>
	<div class="min-w-0">
		<p class="truncate text-sm font-medium text-fg">{name}</p>
		<p class="truncate text-xs text-fg-muted">{detail}</p>
	</div>
</div>
<dl class="mt-3 flex flex-col gap-1.5 px-1 text-xs">
	<div class="flex items-baseline justify-between gap-3">
		<dt class="text-fg-muted">Price</dt>
		<dd class="font-mono text-fg">{formatRobux(priceInRobux, false)}</dd>
	</div>
	<div class="flex items-baseline justify-between gap-3">
		<dt class="min-w-0 truncate text-fg-muted">
			@{accountUsername}'s Robux
		</dt>
		<dd class="font-mono text-fg">
			{#if robuxBalance === undefined}
				<Skeleton className="w-16" label="Loading the Robux balance" />
			{:else if robuxBalance === null}
				<span class="font-sans text-fg-muted">Unavailable</span>
			{:else}
				{formatRobux(robuxBalance, false)}
			{/if}
		</dd>
	</div>
	<div
		class="flex items-baseline justify-between gap-3 border-t border-line-subtle pt-1.5">
		<dt class="font-medium text-fg">After purchase</dt>
		<dd
			class={[
				"font-mono font-semibold",
				isShortOfRobux ? "text-danger-muted" : "text-fg",
			]}>
			{#if robuxBalance === undefined}
				<Skeleton className="w-16" label="Loading the Robux balance" />
			{:else if balanceAfterPurchase === null}
				<span class="font-sans font-normal text-fg-muted">Unavailable</span>
			{:else}
				{formatRobux(balanceAfterPurchase, false)}
			{/if}
		</dd>
	</div>
</dl>
{#if isShortOfRobux && balanceAfterPurchase !== null}
	<p class="mt-3 notice-error" role="alert">
		@{accountUsername} needs {formatRobux(-balanceAfterPurchase, false)}
		more to buy this item.
	</p>
{/if}

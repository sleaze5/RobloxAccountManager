<script lang="ts">
	import shoppingBagIcon from "@phosphor-icons/core/regular/shopping-bag.svg?raw"
	import storefrontIcon from "@phosphor-icons/core/regular/storefront.svg?raw"
	import xIcon from "@phosphor-icons/core/regular/x.svg?raw"
	import type { DeveloperProduct } from "../backend/bridge"
	import { salePriceLabel } from "../inventory/inventory-prices"
	import Dialog from "../shared/Dialog.svelte"
	import Icon from "../shared/Icon.svelte"
	import ProfileImage from "../shared/ProfileImage.svelte"

	let {
		developerProduct,
		gameName,
		onClose,
	}: {
		developerProduct: DeveloperProduct
		gameName: string
		onClose: () => void
	} = $props()

	const id = $props.id(),
		metaClass =
			"flex min-w-0 flex-wrap items-center gap-1 font-mono text-2xs wrap-anywhere text-fg-muted",
		name = $derived(developerProduct.name || "Unnamed developer product")
</script>

<svelte:window
	onkeydown={(event) => {
		if (event.key === "Escape") {
			event.preventDefault()
			onClose()
		}
	}} />

<Dialog
	class="max-w-lg text-left"
	aria-labelledby={`${id}-title`}
	onBackdropClick={onClose}>
	<div class="flex flex-col gap-4">
		<div class="flex items-start gap-4">
			<span
				class="grid size-28 shrink-0 place-items-center overflow-hidden rounded-lg border border-line-subtle bg-raised"
				aria-hidden="true">
				<ProfileImage
					url={developerProduct.iconUrl}
					fallbackIcon={shoppingBagIcon} />
			</span>
			<div class="flex min-w-0 flex-1 flex-col gap-1">
				<div class="flex items-start gap-3">
					<h2
						id={`${id}-title`}
						class="min-w-0 flex-1 leading-7 tracking-tight wrap-anywhere">
						{name}
					</h2>
					<button
						class="button-close"
						type="button"
						aria-label="Close dialog"
						onclick={onClose}>
						<Icon svg={xIcon} size={14} aria-hidden="true" />
					</button>
				</div>
				<div class={metaClass}>
					<span class="select-text"
						>ID: {developerProduct.developerProductId}</span>
					<span class="text-fg-faint" aria-hidden="true">·</span>
					<span>Developer product</span>
				</div>
				<div class={metaClass}><span>{gameName}</span></div>
				<p
					class={[
						"inline-flex items-center gap-1 text-xs font-medium tabular-nums",
						developerProduct.isForSale ? "text-fg" : "text-fg-muted",
					]}>
					<Icon svg={storefrontIcon} size={14} aria-hidden="true" />
					{salePriceLabel(developerProduct, false)}
				</p>
				<p class="text-xs text-fg-muted">Players buy it inside the game.</p>
			</div>
		</div>

		<section class="flex flex-col gap-3" aria-labelledby={`${id}-description`}>
			<h3 id={`${id}-description`} class="text-xs font-semibold text-fg-muted">
				Description
			</h3>
			{#if developerProduct.description}
				<p class="text-block" data-variant="prose">
					{developerProduct.description}
				</p>
			{:else}
				<p class="text-xs text-fg-muted">No description provided.</p>
			{/if}
		</section>
	</div>
</Dialog>

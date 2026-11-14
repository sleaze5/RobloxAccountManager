<script lang="ts">
	import type { InventoryItem } from "../backend/bridge"
	import ProfileImage from "../shared/ProfileImage.svelte"
	import { inventoryTypeLabel } from "./inventory-categories"
	import type { InventoryCategoryOption } from "./inventory-categories"

	let {
		item,
		category,
		onOpen,
	}: {
		item: InventoryItem
		category: InventoryCategoryOption
		onOpen: (item: InventoryItem) => void
	} = $props()

	const exactNumbers = new Intl.NumberFormat("en-US"),
		compactNumbers = new Intl.NumberFormat("en-US", {
			notation: "compact",
			maximumFractionDigits: 1,
		}),
		name = $derived(item.name || "Unnamed item"),
		typeLabel = $derived(inventoryTypeLabel(item.typeName))
</script>

<button
	class="group/tile flex w-full min-w-0 flex-col gap-1 rounded-lg border border-line-subtle bg-surface p-1 text-left transition-colors duration-100 hover:border-line hover:bg-hover"
	type="button"
	data-tooltip={`Open ${name} on Roblox`}
	data-tooltip-strong={name}
	onclick={() => onOpen(item)}>
	<span
		class="relative grid aspect-square w-full place-items-center overflow-hidden rounded-sm bg-raised">
		<span
			class="grid size-full place-items-center transition-transform duration-150 ease-out motion-safe:group-hover/tile:scale-105"
			aria-hidden="true">
			<ProfileImage url={item.imageUrl} fallbackIcon={category.icon} />
		</span>
		{#if item.serialNumber !== null}
			<span
				class="absolute top-1 left-1 rounded-sm border border-line-strong bg-overlay px-1 font-mono text-2xs leading-4 font-medium text-fg"
				><span class="sr-only">Serial number </span>#{exactNumbers.format(
					item.serialNumber,
				)}</span>
		{/if}
	</span>
	<span class="flex min-w-0 flex-col px-1 pb-0.5">
		<span class="truncate text-xs font-medium text-fg">{name}</span>
		{#if category.itemDetail === "price"}
			<span class="truncate font-mono text-2xs text-fg-muted"
				>{item.recentAveragePrice !== null
					? `RAP R$ ${compactNumbers.format(item.recentAveragePrice)}`
					: "No RAP"}</span>
		{:else if category.itemDetail === "type"}
			<span class="truncate text-2xs text-fg-muted">{typeLabel || " "}</span>
		{/if}
	</span>
</button>

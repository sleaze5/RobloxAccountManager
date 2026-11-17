<script lang="ts">
	import type { AvatarAsset } from "../backend/bridge"
	import { inventoryTypeLabel } from "../inventory/inventory-categories"
	import ProfileImage from "../shared/ProfileImage.svelte"
	import { avatarItemCategory } from "./avatar-categories"

	let {
		asset,
		isMarked = false,
	}: {
		asset: AvatarAsset
		// isMarked puts an accent dot on the image, such as for an adjusted
		// accessory.
		isMarked?: boolean
	} = $props()

	const typeLabel = $derived(inventoryTypeLabel(asset.item.typeName))
</script>

<span class="flex min-w-0 flex-1 items-center gap-3">
	<span class="relative size-9 shrink-0" aria-hidden="true">
		<span
			class="grid size-full place-items-center overflow-hidden rounded-md bg-raised">
			<ProfileImage
				url={asset.item.imageUrl}
				fallbackIcon={avatarItemCategory(asset.category).icon} />
		</span>
		{#if isMarked}
			<span
				class="absolute -top-0.5 -right-0.5 size-2.5 animate-bloom rounded-full border-2 border-surface bg-accent"
			></span>
		{/if}
	</span>
	<span class="flex min-w-0 flex-col">
		<span class="truncate text-sm font-medium text-fg"
			>{asset.item.name || "Unnamed item"}</span>
		{#if typeLabel}
			<span class="truncate text-xs text-fg-muted">{typeLabel}</span>
		{/if}
	</span>
</span>

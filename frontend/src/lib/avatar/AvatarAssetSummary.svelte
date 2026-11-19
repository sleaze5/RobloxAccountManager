<script lang="ts">
	import type { InventoryCategory, InventoryItem } from "../backend/bridge"
	import { inventoryTypeLabel } from "../inventory/inventory-categories"
	import ProfileImage from "../shared/ProfileImage.svelte"
	import { avatarItemCategory } from "./avatar-categories"

	// AvatarAssetSummary shows an avatar item with its image, name, and type,
	// such as a worn accessory or the pose of a thumbnail. category picks the
	// icon of an item without an image.
	let { item, category }: { item: InventoryItem; category: InventoryCategory } =
		$props()

	const typeLabel = $derived(inventoryTypeLabel(item.typeName))
</script>

<span class="flex min-w-0 flex-1 items-center gap-2">
	<span
		class="grid size-9 shrink-0 place-items-center overflow-hidden rounded-md bg-raised"
		aria-hidden="true">
		<ProfileImage
			url={item.imageUrl}
			fallbackIcon={avatarItemCategory(category).icon} />
	</span>
	<span class="flex min-w-0 flex-col">
		<span class="truncate text-sm font-medium text-fg"
			>{item.name || "Unnamed item"}</span>
		{#if typeLabel}
			<span
				class="truncate text-xs text-fg-faint group-hover/row:text-fg-muted group-data-open/row:text-fg-muted"
				>{typeLabel}</span>
		{/if}
	</span>
</span>

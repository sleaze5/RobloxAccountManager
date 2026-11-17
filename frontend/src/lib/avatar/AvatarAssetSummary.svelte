<script lang="ts">
	import type { AvatarAsset } from "../backend/bridge"
	import { inventoryTypeLabel } from "../inventory/inventory-categories"
	import ProfileImage from "../shared/ProfileImage.svelte"
	import { avatarItemCategory } from "./avatar-categories"

	let { asset, detail }: { asset: AvatarAsset; detail: string } = $props()

	const typeLabel = $derived(inventoryTypeLabel(asset.item.typeName))
</script>

<span class="flex min-w-0 flex-1 items-center gap-3">
	<span
		class="grid size-9 shrink-0 place-items-center overflow-hidden rounded-md bg-raised"
		aria-hidden="true">
		<ProfileImage
			url={asset.item.imageUrl}
			fallbackIcon={avatarItemCategory(asset.category).icon} />
	</span>
	<span class="flex min-w-0 flex-col">
		<span class="truncate text-sm font-medium text-fg"
			>{asset.item.name || "Unnamed item"}</span>
		<span class="truncate text-xs text-fg-muted"
			>{[typeLabel, detail].filter(Boolean).join(" - ")}</span>
	</span>
</span>

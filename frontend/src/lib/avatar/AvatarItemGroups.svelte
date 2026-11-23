<script lang="ts">
	import slidersHorizontalIcon from "@phosphor-icons/core/regular/sliders-horizontal.svg?raw"
	import type { Snippet } from "svelte"
	import type { AvatarAsset } from "../backend/bridge"
	import InventoryItemTile from "../inventory/InventoryItemTile.svelte"
	import type { ItemTileBadge } from "../shared/ItemTile.svelte"
	import AvatarCountedHeading from "./AvatarCountedHeading.svelte"
	import { groupWornAssets } from "./avatar-categories"
	import { isDefaultAdjustment } from "./avatar-draft"

	let {
		assets,
		headingIdPrefix,
		isOwned,
		badgesOf = () => [],
		onOpenItem,
		itemControls,
	}: {
		assets: AvatarAsset[]
		// headingIdPrefix keeps the heading IDs unique while the avatar page
		// and an outfit dialog both show groups.
		headingIdPrefix: string
		// Whether the account that the page acts for owns the asset.
		isOwned: (asset: AvatarAsset) => boolean
		// badgesOf adds the marks that only the caller knows to each tile.
		badgesOf?: (asset: AvatarAsset) => ItemTileBadge[]
		onOpenItem: (asset: AvatarAsset) => void
		// itemControls renders its buttons at the end of a row along the top of
		// each tile.
		itemControls?: Snippet<[AvatarAsset]>
	} = $props()

	const groups = $derived(groupWornAssets(assets)),
		adjustedBadge: ItemTileBadge = {
			label: "Adjusted placement",
			icon: slidersHorizontalIcon,
			tone: "text-fg",
		}
</script>

{#each groups as group, groupIndex (group.category.label)}
	<section
		class="flex flex-col gap-2"
		aria-labelledby={`${headingIdPrefix}-${groupIndex}`}>
		<AvatarCountedHeading
			id={`${headingIdPrefix}-${groupIndex}`}
			label={group.category.label}
			count={group.assets.length}
			icon={group.category.icon} />
		<ul
			class="grid grid-cols-3 gap-2 @2xl:grid-cols-4 @4xl:grid-cols-5 @6xl:grid-cols-7">
			{#each group.assets as asset (asset.item.itemId)}
				<li class="relative min-w-0 animate-enter">
					<InventoryItemTile
						item={asset.item}
						category={group.category}
						isWorn={false}
						isOwned={isOwned(asset)}
						extraBadges={[
							...badgesOf(asset),
							...(isDefaultAdjustment(asset.adjustment)
								? []
								: [adjustedBadge]),
						]}
						actionLabel="Show details"
						onOpen={() => onOpenItem(asset)} />
					{#if itemControls}
						<span
							class="pointer-events-none absolute inset-x-2 top-2 flex h-6 items-center justify-end gap-1">
							{@render itemControls(asset)}
						</span>
					{/if}
				</li>
			{/each}
		</ul>
	</section>
{/each}

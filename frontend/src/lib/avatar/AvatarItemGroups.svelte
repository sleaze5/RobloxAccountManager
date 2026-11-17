<script lang="ts">
	import type { Snippet } from "svelte"
	import type { AvatarAsset } from "../backend/bridge"
	import InventoryItemTile from "../inventory/InventoryItemTile.svelte"
	import Icon from "../shared/Icon.svelte"
	import { groupWornAssets } from "./avatar-categories"
	import { isDefaultAdjustment } from "./avatar-draft"

	let {
		assets,
		headingIdPrefix,
		onOpenItem,
		itemControls,
	}: {
		assets: AvatarAsset[]
		// headingIdPrefix keeps the heading IDs unique while the avatar page
		// and an outfit dialog both show groups.
		headingIdPrefix: string
		onOpenItem: (asset: AvatarAsset) => void
		// itemControls renders in a row along the top of each tile, with its
		// tags first and its buttons at the end.
		itemControls?: Snippet<[AvatarAsset]>
	} = $props()

	const groups = $derived(groupWornAssets(assets))
</script>

{#each groups as group, groupIndex (group.category.label)}
	<section
		class="flex flex-col gap-2"
		aria-labelledby={`${headingIdPrefix}-${groupIndex}`}>
		<h3
			id={`${headingIdPrefix}-${groupIndex}`}
			class="flex items-center gap-2 text-xs font-semibold text-fg-muted">
			<Icon
				svg={group.category.icon}
				class="text-fg-faint"
				size={14}
				aria-hidden="true" />
			{group.category.label}
			<span class="font-mono text-2xs font-normal text-fg-faint"
				>{group.assets.length}</span>
		</h3>
		<ul
			class="grid grid-cols-3 gap-2 @lg:grid-cols-4 @2xl:grid-cols-5 @4xl:grid-cols-6 @6xl:grid-cols-8">
			{#each group.assets as asset (asset.item.itemId)}
				<li class="relative min-w-0 animate-enter">
					<InventoryItemTile
						item={asset.item}
						category={group.category}
						isWorn={false}
						actionLabel="Show details"
						onOpen={() => onOpenItem(asset)} />
					<!-- The tags and the controls share one row, so that they center
					     on the same line. -->
					<span
						class="pointer-events-none absolute inset-x-2 top-2 flex h-6 items-center gap-1">
						{#if !isDefaultAdjustment(asset.adjustment)}
							<span
								class="rounded-sm border border-line-strong bg-overlay px-1 text-2xs leading-4 font-medium text-fg"
								>Adjusted</span>
						{/if}
						{@render itemControls?.(asset)}
					</span>
				</li>
			{/each}
		</ul>
	</section>
{/each}

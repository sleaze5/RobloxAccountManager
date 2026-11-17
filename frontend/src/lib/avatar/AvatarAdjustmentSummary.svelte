<script lang="ts">
	import type { AvatarAsset } from "../backend/bridge"
	import AvatarAdjustmentFacts from "./AvatarAdjustmentFacts.svelte"
	import AvatarAdjustmentRow from "./AvatarAdjustmentRow.svelte"
	import AvatarClothingLayers from "./AvatarClothingLayers.svelte"
	import AvatarCountedHeading from "./AvatarCountedHeading.svelte"
	import { isDefaultAdjustment } from "./avatar-draft"

	let {
		assets,
		headingIdPrefix,
	}: {
		assets: AvatarAsset[]
		// headingIdPrefix keeps the IDs unique while the avatar page and an
		// outfit dialog both show adjustments.
		headingIdPrefix: string
	} = $props()

	// The accessory whose values are open, or 0.
	let expandedAssetId = $state(0)

	const adjustedAssets = $derived(
		assets.filter((asset) => !isDefaultAdjustment(asset.adjustment)),
	)
</script>

{#if adjustedAssets.length > 0}
	<section
		class="flex flex-col gap-2"
		aria-labelledby={`${headingIdPrefix}-adjusted`}>
		<AvatarCountedHeading
			id={`${headingIdPrefix}-adjusted`}
			label="Adjusted accessories"
			count={adjustedAssets.length} />
		<ul
			class="flex flex-col overflow-hidden rounded-lg border border-line-subtle bg-surface">
			{#each adjustedAssets as asset (asset.item.itemId)}
				{@const assetId = asset.item.itemId}
				<AvatarAdjustmentRow
					{asset}
					controlsId={`${headingIdPrefix}-${assetId}`}
					isExpanded={expandedAssetId === assetId}
					onToggle={() =>
						(expandedAssetId = expandedAssetId === assetId ? 0 : assetId)}>
					{#if asset.adjustment}
						<AvatarAdjustmentFacts adjustment={asset.adjustment} />
					{/if}
				</AvatarAdjustmentRow>
			{/each}
		</ul>
	</section>
{/if}
<AvatarClothingLayers {assets} headingId={`${headingIdPrefix}-layers`} />

<script lang="ts">
	import type { AvatarAsset } from "../backend/bridge"
	import AvatarAdjustmentFacts from "./AvatarAdjustmentFacts.svelte"
	import AvatarAssetSummary from "./AvatarAssetSummary.svelte"
	import AvatarClothingLayers from "./AvatarClothingLayers.svelte"
	import { isDefaultAdjustment } from "./avatar-draft"

	let {
		assets,
		headingIdPrefix,
	}: {
		assets: AvatarAsset[]
		// headingIdPrefix keeps the heading IDs unique while the avatar page
		// and an outfit dialog both show adjustments.
		headingIdPrefix: string
	} = $props()

	const adjustedAssets = $derived(
		assets.filter((asset) => !isDefaultAdjustment(asset.adjustment)),
	)
</script>

{#if adjustedAssets.length > 0}
	<section
		class="flex flex-col gap-2"
		aria-labelledby={`${headingIdPrefix}-adjusted`}>
		<h3
			id={`${headingIdPrefix}-adjusted`}
			class="text-xs font-semibold text-fg-muted">
			Adjusted accessories
		</h3>
		<ul class="flex flex-col gap-2">
			{#each adjustedAssets as asset (asset.item.itemId)}
				{#if asset.adjustment}
					<li
						class="flex flex-col gap-1 rounded-lg border border-line-subtle bg-surface px-3 py-2">
						<AvatarAssetSummary {asset} detail="Adjusted" />
						<AvatarAdjustmentFacts adjustment={asset.adjustment} />
					</li>
				{/if}
			{/each}
		</ul>
	</section>
{/if}
<AvatarClothingLayers {assets} headingId={`${headingIdPrefix}-layers`} />

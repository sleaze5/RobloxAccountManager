<script lang="ts">
	import type { AvatarAsset } from "../backend/bridge"
	import AvatarAssetSummary from "./AvatarAssetSummary.svelte"
	import AvatarCountedHeading from "./AvatarCountedHeading.svelte"
	import { formatPuffiness } from "./adjustment-format"

	let {
		assets,
		headingId,
	}: {
		// assets may include items without a layer, which the list leaves out.
		assets: AvatarAsset[]
		headingId: string
	} = $props()

	const exactNumbers = new Intl.NumberFormat("en-US"),
		// The outermost layer is listed first, as the avatar shows it.
		layeredAssets = $derived(
			assets
				.filter((asset) => asset.layer)
				.sort(
					(first, second) =>
						(second.layer?.order ?? 0) - (first.layer?.order ?? 0),
				),
		),
		hasPuffiness = $derived(
			layeredAssets.some((asset) => asset.layer?.puffiness !== null),
		)
</script>

{#if layeredAssets.length > 0}
	<section class="flex flex-col gap-2" aria-labelledby={headingId}>
		<div class="flex flex-col gap-0.5">
			<AvatarCountedHeading
				id={headingId}
				label="Layered clothing"
				count={layeredAssets.length} />
			<p class="text-xs text-fg-muted">
				Outer layers come first.{hasPuffiness
					? " Puffiness comes from older avatar editors and does not change."
					: ""}
			</p>
		</div>
		<ul
			class="flex flex-col overflow-hidden rounded-lg border border-line-subtle bg-surface">
			{#each layeredAssets as asset (asset.item.itemId)}
				{@const layer = asset.layer}
				{#if layer}
					{@const puffiness = formatPuffiness(layer)}
					<!-- The right padding keeps the layer in line with the status of
					     the accessory rows above, which end in a caret. -->
					<li
						class="flex h-12 min-w-0 items-center gap-2 border-line-subtle pr-9 pl-3 not-first:border-t">
						<AvatarAssetSummary
							item={asset.item}
							category={asset.category} />
						<span class="flex shrink-0 flex-col items-end">
							<span class="text-xs text-fg"
								>Layer <span class="font-mono tabular-nums"
									>{exactNumbers.format(layer.order)}</span
								></span>
							{#if puffiness}
								<span class="text-2xs text-fg-muted"
									>Puffiness <span class="font-mono">{puffiness}</span
									></span>
							{/if}
						</span>
					</li>
				{/if}
			{/each}
		</ul>
	</section>
{/if}

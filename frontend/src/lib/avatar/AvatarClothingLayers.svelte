<script lang="ts">
	import type { AvatarAsset } from "../backend/bridge"
	import AvatarAssetSummary from "./AvatarAssetSummary.svelte"
	import { formatPuffiness } from "./adjustment-format"

	let {
		assets,
		headingId,
	}: {
		// assets lists the layered items, which each have a layer.
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
		<h3 id={headingId} class="text-xs font-semibold text-fg-muted">
			Layered clothing
		</h3>
		<p class="text-xs text-fg-muted">
			Roblox puts higher layers over lower ones.{hasPuffiness
				? " Puffiness is kept from older avatar editors."
				: ""}
		</p>
		<ul class="flex flex-col rounded-lg border border-line-subtle">
			{#each layeredAssets as asset (asset.item.itemId)}
				{@const layer = asset.layer}
				{#if layer}
					{@const puffiness = formatPuffiness(layer)}
					<li
						class="flex min-w-0 items-center gap-3 border-line-subtle px-3 py-2 not-first:border-t">
						<AvatarAssetSummary {asset} detail="" />
						<span class="flex shrink-0 flex-col items-end text-xs">
							<span class="font-mono text-fg tabular-nums"
								>Layer {exactNumbers.format(layer.order)}</span>
							{#if puffiness}
								<span class="font-mono text-2xs text-fg-muted"
									>Puffiness {puffiness}</span>
							{/if}
						</span>
					</li>
				{/if}
			{/each}
		</ul>
	</section>
{/if}

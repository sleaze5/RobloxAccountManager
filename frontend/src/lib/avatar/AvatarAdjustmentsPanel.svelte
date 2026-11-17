<script lang="ts">
	import slidersHorizontalIcon from "@phosphor-icons/core/regular/sliders-horizontal.svg?raw"
	import { onMount } from "svelte"
	import Icon from "../shared/Icon.svelte"
	import AvatarAccessoryAdjuster from "./AvatarAccessoryAdjuster.svelte"
	import AvatarAdjustmentFacts from "./AvatarAdjustmentFacts.svelte"
	import AvatarAdjustmentRow from "./AvatarAdjustmentRow.svelte"
	import AvatarAdjustmentSummary from "./AvatarAdjustmentSummary.svelte"
	import AvatarClothingLayers from "./AvatarClothingLayers.svelte"
	import { isDefaultAdjustment, isSameAdjustment } from "./avatar-draft"
	import type { AvatarState } from "./avatar-state.svelte"

	let {
		avatar,
		canEdit,
		username,
		expandedAssetId = $bindable(),
	}: {
		avatar: AvatarState
		canEdit: boolean
		username: string
		// expandedAssetId is the accessory whose controls are open, or 0. The
		// Wearing tab sets it to open an accessory here.
		expandedAssetId: number
	} = $props()

	const assets = $derived(avatar.draft?.assets ?? []),
		savedAssetById = $derived(
			new Map(
				(avatar.avatar?.assets ?? []).map((asset) => [
					asset.item.itemId,
					asset,
				]),
			),
		),
		// The editor lists every accessory that the account may adjust, and
		// any other accessory that Roblox lists with an adjustment.
		adjustableAssets = $derived(
			assets.filter(
				(asset) =>
					!!avatar.adjustmentLimitsOf(asset.item.typeName) ||
					!isDefaultAdjustment(asset.adjustment),
			),
		),
		hasLayers = $derived(assets.some((asset) => asset.layer)),
		hasAdjustments = $derived(
			assets.some((asset) => !isDefaultAdjustment(asset.adjustment)),
		),
		isEmpty = $derived(
			canEdit
				? adjustableAssets.length === 0 && !hasLayers
				: !hasAdjustments && !hasLayers,
		)

	// The accessory that the Wearing tab opened scrolls into view and takes
	// focus, because the button that opened it left with the Wearing tab.
	onMount(() => {
		if (!expandedAssetId) return
		const row = document.getElementById(`avatar-adjust-row-${expandedAssetId}`)
		row?.scrollIntoView({ block: "nearest" })
		row?.querySelector<HTMLButtonElement>("button[aria-expanded]")?.focus({
			preventScroll: true,
		})
	})
</script>

<div class="flex flex-col gap-5 p-3">
	<p class="flex min-h-7 items-center text-xs text-fg-muted">
		{canEdit
			? "Move, turn, and resize accessories. The image previews your changes."
			: `How @${username} placed accessories and stacked layered clothing.`}
	</p>

	{#if isEmpty}
		<div class="empty-state py-10" role="status">
			<Icon svg={slidersHorizontalIcon} size={28} aria-hidden="true" />
			<h2>{canEdit ? "No adjustable accessories" : "No adjustments"}</h2>
			<p>
				{canEdit
					? "Put on a hat or another rigid accessory to adjust where it sits."
					: `@${username} wears every accessory where it attaches by default.`}
			</p>
		</div>
	{:else if canEdit}
		{#if adjustableAssets.length > 0}
			<section
				class="flex flex-col gap-2"
				aria-labelledby="avatar-adjust-accessories">
				<h3
					id="avatar-adjust-accessories"
					class="flex items-center gap-2 text-xs font-semibold text-fg-muted">
					Accessories
					<span class="font-mono text-2xs font-normal text-fg-faint"
						>{adjustableAssets.length}</span>
				</h3>
				<ul
					class="flex flex-col overflow-hidden rounded-lg border border-line-subtle bg-surface">
					{#each adjustableAssets as asset (asset.item.itemId)}
						{@const assetId = asset.item.itemId}
						{@const limits = avatar.adjustmentLimitsOf(asset.item.typeName)}
						{@const savedAsset = savedAssetById.get(assetId)}
						<AvatarAdjustmentRow
							{asset}
							rowId={`avatar-adjust-row-${assetId}`}
							controlsId={`avatar-adjust-${assetId}`}
							isExpanded={expandedAssetId === assetId}
							isEdited={!!savedAsset &&
								!isSameAdjustment(
									savedAsset.adjustment,
									asset.adjustment,
								)}
							unavailableReason={!savedAsset
								? "Save to adjust"
								: !limits
									? "Can't be adjusted"
									: ""}
							onToggle={() =>
								(expandedAssetId =
									expandedAssetId === assetId ? 0 : assetId)}>
							{#if limits && savedAsset}
								<AvatarAccessoryAdjuster {avatar} {asset} {limits} />
							{:else if asset.adjustment}
								<AvatarAdjustmentFacts adjustment={asset.adjustment} />
							{/if}
						</AvatarAdjustmentRow>
					{/each}
				</ul>
			</section>
		{/if}
		<AvatarClothingLayers {assets} headingId="avatar-adjust-layers" />
	{:else}
		<AvatarAdjustmentSummary {assets} headingIdPrefix="avatar-adjust" />
	{/if}
</div>

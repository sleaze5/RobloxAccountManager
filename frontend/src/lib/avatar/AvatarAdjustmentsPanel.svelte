<script lang="ts">
	import caretDownIcon from "@phosphor-icons/core/regular/caret-down.svg?raw"
	import slidersHorizontalIcon from "@phosphor-icons/core/regular/sliders-horizontal.svg?raw"
	import Icon from "../shared/Icon.svelte"
	import AvatarAccessoryAdjuster from "./AvatarAccessoryAdjuster.svelte"
	import AvatarAdjustmentFacts from "./AvatarAdjustmentFacts.svelte"
	import AvatarAdjustmentSummary from "./AvatarAdjustmentSummary.svelte"
	import AvatarAssetSummary from "./AvatarAssetSummary.svelte"
	import AvatarClothingLayers from "./AvatarClothingLayers.svelte"
	import { isDefaultAdjustment } from "./avatar-draft"
	import type { AvatarState } from "./avatar-state.svelte"

	let {
		avatar,
		canEdit,
		username,
	}: {
		avatar: AvatarState
		canEdit: boolean
		username: string
	} = $props()

	// The accessory whose controls are open, or 0.
	let expandedAssetId = $state(0)

	const assets = $derived(avatar.draft?.assets ?? []),
		savedAssetIds = $derived(
			new Set((avatar.avatar?.assets ?? []).map((asset) => asset.item.itemId)),
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
</script>

<div class="flex flex-col gap-4 p-3">
	<p class="flex min-h-7 items-center text-xs text-fg-muted">
		{canEdit
			? "Move, turn, and resize accessories within the limits that Roblox allows. The image previews your changes."
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
					class="text-xs font-semibold text-fg-muted">
					Accessories
				</h3>
				<ul class="flex flex-col gap-2">
					{#each adjustableAssets as asset (asset.item.itemId)}
						{@const assetId = asset.item.itemId}
						{@const limits = avatar.adjustmentLimitsOf(asset.item.typeName)}
						{@const isSaved = savedAssetIds.has(assetId)}
						{@const isExpanded = expandedAssetId === assetId}
						{@const canAdjust = !!limits && isSaved}
						{@const detail = !isSaved
							? "Save the avatar to adjust it"
							: !limits
								? "Roblox doesn't let this account adjust it"
								: isDefaultAdjustment(asset.adjustment)
									? "Default placement"
									: "Adjusted"}
						<li
							class="flex animate-enter flex-col rounded-lg border border-line-subtle bg-surface">
							<button
								class="group flex min-w-0 items-center gap-3 rounded-lg px-3 py-2 text-left transition-colors duration-100 hover:bg-hover disabled:hover:bg-transparent"
								type="button"
								aria-expanded={isExpanded}
								aria-controls={`avatar-adjust-${assetId}`}
								disabled={!canAdjust &&
									isDefaultAdjustment(asset.adjustment)}
								onclick={() =>
									(expandedAssetId = isExpanded ? 0 : assetId)}>
								<AvatarAssetSummary {asset} {detail} />
								<Icon
									svg={caretDownIcon}
									class="shrink-0 text-fg-muted transition-transform duration-100 group-aria-expanded:rotate-180"
									size={16}
									aria-hidden="true" />
							</button>
							{#if isExpanded}
								<div
									id={`avatar-adjust-${assetId}`}
									class="animate-enter border-t border-line-subtle px-3 py-3">
									{#if limits && canAdjust}
										<AvatarAccessoryAdjuster
											{avatar}
											{asset}
											{limits} />
									{:else if asset.adjustment}
										<AvatarAdjustmentFacts
											adjustment={asset.adjustment} />
									{/if}
								</div>
							{/if}
						</li>
					{/each}
				</ul>
			</section>
		{/if}
		<AvatarClothingLayers {assets} headingId="avatar-adjust-layers" />
	{:else}
		<AvatarAdjustmentSummary {assets} headingIdPrefix="avatar-adjust" />
	{/if}
</div>

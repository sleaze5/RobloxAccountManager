<script lang="ts">
	import plusIcon from "@phosphor-icons/core/regular/plus.svg?raw"
	import asteriskIcon from "@phosphor-icons/core/regular/asterisk.svg?raw"
	import slidersHorizontalIcon from "@phosphor-icons/core/regular/sliders-horizontal.svg?raw"
	import tShirtIcon from "@phosphor-icons/core/regular/t-shirt.svg?raw"
	import xIcon from "@phosphor-icons/core/regular/x.svg?raw"
	import type { AvatarAsset } from "../backend/bridge"
	import Icon from "../shared/Icon.svelte"
	import AvatarItemGroups from "./AvatarItemGroups.svelte"
	import type { AvatarState } from "./avatar-state.svelte"

	let {
		avatar,
		canEdit,
		username,
		isOwned,
		onOpenItem,
		onAddItems,
		onAdjustItem,
	}: {
		avatar: AvatarState
		canEdit: boolean
		username: string
		isOwned: (asset: AvatarAsset) => boolean
		onOpenItem: (asset: AvatarAsset) => void
		onAddItems: () => void
		onAdjustItem: (assetId: number) => void
	} = $props()
	const elementIdPrefix = $props.id()

	const exactNumbers = new Intl.NumberFormat("en-US"),
		assets = $derived(avatar.draft?.assets ?? []),
		savedAssetIds = $derived(
			new Set((avatar.avatar?.assets ?? []).map((asset) => asset.item.itemId)),
		)
</script>

<div class="flex flex-col gap-4 p-3">
	<div class="flex min-h-7 items-center gap-2">
		<p class="min-w-0 flex-1 text-xs text-fg-muted" aria-live="polite">
			{assets.length === 0
				? "Nothing worn"
				: `${exactNumbers.format(assets.length)} ${assets.length === 1 ? "item" : "items"} worn, including body parts and animations`}
		</p>
		{#if canEdit}
			<button
				class="button"
				type="button"
				disabled={avatar.isDraftLocked}
				onclick={onAddItems}>
				<Icon svg={plusIcon} size={16} aria-hidden="true" />
				Add items
			</button>
		{/if}
	</div>

	{#if assets.length === 0}
		<div class="empty-state py-10" role="status">
			<Icon svg={tShirtIcon} size={28} aria-hidden="true" />
			<h2>Nothing worn</h2>
			<p>
				{canEdit
					? "Add items from the inventory, then save the avatar."
					: `@${username} wears only the default body.`}
			</p>
		</div>
	{/if}

	<AvatarItemGroups
		{assets}
		headingIdPrefix={`${elementIdPrefix}-avatar-group`}
		{isOwned}
		badgesOf={(asset) =>
			canEdit && !savedAssetIds.has(asset.item.itemId)
				? [{ label: "New, not saved yet", icon: asteriskIcon, tone: "text-fg" }]
				: []}
		{onOpenItem}>
		{#snippet itemControls(asset)}
			{#if canEdit}
				<span class="pointer-events-auto flex gap-1">
					{#if savedAssetIds.has(asset.item.itemId) && avatar.adjustmentLimitsOf(asset.item.typeName)}
						<button
							class="button-icon size-6 bg-overlay"
							type="button"
							aria-label={`Adjust ${asset.item.name || "item"}`}
							data-tooltip="Adjust placement"
							onclick={() => onAdjustItem(asset.item.itemId)}>
							<Icon
								svg={slidersHorizontalIcon}
								size={14}
								aria-hidden="true" />
						</button>
					{/if}
					<button
						class="button-icon size-6 bg-overlay"
						type="button"
						aria-label={`Take off ${asset.item.name || "item"}`}
						data-tooltip="Take off"
						disabled={avatar.isDraftLocked}
						onclick={() => avatar.takeOff(asset.item.itemId)}>
						<Icon svg={xIcon} size={14} aria-hidden="true" />
					</button>
				</span>
			{/if}
		{/snippet}
	</AvatarItemGroups>
</div>

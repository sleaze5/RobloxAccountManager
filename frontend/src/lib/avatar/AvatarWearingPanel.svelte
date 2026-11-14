<script lang="ts">
	import plusIcon from "@phosphor-icons/core/regular/plus.svg?raw"
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
		onOpenItem,
		onAddItems,
	}: {
		avatar: AvatarState
		canEdit: boolean
		username: string
		onOpenItem: (asset: AvatarAsset) => void
		onAddItems: () => void
	} = $props()

	const exactNumbers = new Intl.NumberFormat("en-US"),
		assets = $derived(avatar.draft?.assets ?? []),
		savedAssetIds = $derived(
			new Set((avatar.avatar?.assets ?? []).map((asset) => asset.item.itemId)),
		)
</script>

<div class="flex flex-col gap-4 p-3">
	<div class="flex h-7 items-center gap-2">
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
			<h2>Wearing nothing</h2>
			<p>
				{canEdit
					? "Add items from the inventory, then save the avatar."
					: `@${username} wears only the default body.`}
			</p>
		</div>
	{/if}

	<AvatarItemGroups {assets} headingIdPrefix="avatar-group" {onOpenItem}>
		{#snippet itemControls(asset)}
			{#if canEdit && !savedAssetIds.has(asset.item.itemId)}
				<span
					class="pointer-events-none absolute top-2 left-2 animate-pop rounded-sm border border-line-strong bg-overlay px-1 text-2xs leading-4 font-medium text-fg"
					>New</span>
			{/if}
			{#if canEdit}
				<button
					class="absolute top-2 right-2 button-icon size-6 bg-overlay"
					type="button"
					aria-label={`Take off ${asset.item.name || "item"}`}
					data-tooltip="Take off"
					disabled={avatar.isDraftLocked}
					onclick={() => avatar.takeOff(asset.item.itemId)}>
					<Icon svg={xIcon} size={14} aria-hidden="true" />
				</button>
			{/if}
		{/snippet}
	</AvatarItemGroups>
</div>

<script lang="ts">
	import plusIcon from "@phosphor-icons/core/regular/plus.svg?raw"
	import xIcon from "@phosphor-icons/core/regular/x.svg?raw"
	import type { AvatarEmote } from "../backend/bridge"
	import InventoryItemTile from "../inventory/InventoryItemTile.svelte"
	import Icon from "../shared/Icon.svelte"
	import { emoteCategory } from "./avatar-categories"
	import { emoteSlotCount } from "./avatar-draft"
	import type { AvatarState } from "./avatar-state.svelte"

	let {
		avatar,
		canEdit,
		onOpenEmote,
		onChooseEmote,
	}: {
		avatar: AvatarState
		canEdit: boolean
		onOpenEmote: (emote: AvatarEmote) => void
		onChooseEmote: (slot: number) => void
	} = $props()

	const slots = Array.from({ length: emoteSlotCount }, (_unused, index) => index + 1),
		emoteBySlot = $derived(
			new Map((avatar.draft?.emotes ?? []).map((emote) => [emote.slot, emote])),
		)
</script>

<div class="flex flex-col gap-4 p-3">
	<p class="flex h-7 items-center text-xs text-fg-muted">
		{canEdit
			? "Emotes play from the emote wheel in experiences. Choose a slot to change it."
			: "The emotes on the emote wheel in experiences."}
	</p>
	<ol class="grid grid-cols-4 gap-2 @2xl:grid-cols-8" aria-label="Emote slots">
		{#each slots as slot (slot)}
			{@const emote = emoteBySlot.get(slot)}
			<li class="relative min-w-0">
				{#if emote}
					<InventoryItemTile
						item={emote.item}
						category={emoteCategory}
						isWorn={false}
						actionLabel={`Slot ${slot}. Show details`}
						onOpen={() => onOpenEmote(emote)} />
					{#if canEdit}
						<button
							class="absolute top-2 right-2 button-icon size-6 bg-overlay"
							type="button"
							aria-label={`Clear slot ${slot}`}
							data-tooltip="Clear slot"
							disabled={avatar.isDraftLocked}
							onclick={() => avatar.clearEmote(slot)}>
							<Icon svg={xIcon} size={14} aria-hidden="true" />
						</button>
					{/if}
				{:else if canEdit}
					<button
						class="flex aspect-square w-full flex-col items-center justify-center gap-1 rounded-lg border border-dashed border-line-strong text-xs text-fg-muted transition-colors duration-100 hover:border-line-focus hover:bg-hover hover:text-fg"
						type="button"
						aria-label={`Add an emote to slot ${slot}`}
						disabled={avatar.isDraftLocked}
						onclick={() => onChooseEmote(slot)}>
						<Icon svg={plusIcon} size={16} aria-hidden="true" />
						Add emote
					</button>
				{:else}
					<span
						class="flex aspect-square w-full items-center justify-center rounded-lg border border-dashed border-line text-xs text-fg-faint"
						>Empty</span>
				{/if}
				<span
					class="pointer-events-none absolute top-2 left-2 rounded-sm border border-line-strong bg-overlay px-1 font-mono text-2xs leading-4 font-medium text-fg-muted"
					aria-hidden="true">{slot}</span>
			</li>
		{/each}
	</ol>
</div>

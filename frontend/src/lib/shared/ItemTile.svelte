<script lang="ts" module>
	export interface ItemTileBadge {
		// label names the mark in its tooltip and for screen readers.
		label: string
		icon: string
		tone: "text-success" | "text-warning" | "text-fg" | "text-fg-muted"
	}
</script>

<script lang="ts">
	import type { Snippet } from "svelte"
	import Icon from "./Icon.svelte"
	import ProfileImage from "./ProfileImage.svelte"

	let {
		name,
		imageUrl,
		fallbackIcon,
		badges = [],
		cornerLabel,
		limitedLabel = "",
		detail = "",
		priceLabel = "",
		isPriceMuted = false,
		isSelected = false,
		isDisabled = false,
		actionLabel,
		onOpen,
	}: {
		name: string
		imageUrl: string
		fallbackIcon: string
		// badges are the marks of the item, in priority order. The last ones
		// clip first when the tile is narrow.
		badges?: ItemTileBadge[]
		// cornerLabel fills the top left corner of the image, such as a serial
		// number.
		cornerLabel?: Snippet
		// limitedLabel marks a limited item in the bottom left corner of its
		// image, as Roblox does.
		limitedLabel?: string
		detail?: string
		priceLabel?: string
		isPriceMuted?: boolean
		// isSelected marks the item that another page opened.
		isSelected?: boolean
		// isDisabled keeps the tile focusable, so that its tooltip can say why
		// the action is unavailable.
		isDisabled?: boolean
		actionLabel: string
		onOpen: () => void
	} = $props()
</script>

<button
	class="group/tile flex h-full w-full min-w-0 flex-col gap-1 rounded-lg border border-line-subtle bg-surface p-1 text-left transition-colors duration-100 hover:border-line hover:bg-hover aria-disabled:cursor-not-allowed aria-disabled:opacity-45 aria-disabled:hover:border-line-subtle aria-disabled:hover:bg-surface data-selected:border-accent"
	type="button"
	aria-disabled={isDisabled || undefined}
	data-selected={isSelected || undefined}
	data-tooltip={actionLabel}
	data-tooltip-strong={name}
	onclick={() => {
		if (!isDisabled) onOpen()
	}}>
	<span
		class="relative grid aspect-square w-full place-items-center overflow-hidden rounded-sm bg-raised">
		<span
			class="grid size-full place-items-center transition-transform duration-150 ease-out motion-safe:group-hover/tile:scale-105"
			aria-hidden="true">
			<ProfileImage url={imageUrl} {fallbackIcon} />
		</span>
		{#if cornerLabel}
			<span
				class="absolute top-1 left-1 max-w-full truncate rounded-sm border border-line-strong bg-overlay px-1 font-mono text-2xs leading-4 font-medium text-fg"
				>{@render cornerLabel()}</span>
		{/if}
		{#if limitedLabel || badges.length > 0}
			<span class="absolute inset-x-1 bottom-1 flex items-end gap-1">
				{#if limitedLabel}
					<span
						class="shrink-0 rounded-sm bg-success px-1 text-2xs leading-4 font-semibold text-canvas"
						>{limitedLabel}</span>
				{/if}
				{#if badges.length > 0}
					<!-- Reversed, so that the first badge sits at the right edge
					     and the last ones clip first on a narrow tile. -->
					<span
						class="flex min-w-0 flex-1 flex-row-reverse gap-0.5 overflow-hidden">
						{#each badges as badge (badge.label)}
							<span
								class={[
									"grid size-5 shrink-0 animate-pop place-items-center rounded-sm border border-line-strong bg-overlay",
									badge.tone,
								]}
								data-tooltip={badge.label}
								><Icon
									svg={badge.icon}
									size={12}
									aria-hidden="true" /><span class="sr-only"
									>{badge.label}</span
								></span>
						{/each}
					</span>
				{/if}
			</span>
		{/if}
	</span>
	<span class="flex min-w-0 flex-col px-1 pb-0.5">
		<span class="truncate text-xs font-medium text-fg">{name}</span>
		{#if detail || priceLabel}
			<span class="flex min-w-0 items-baseline justify-between gap-2 text-2xs">
				<span class="truncate text-fg-muted">{detail}</span>
				{#if priceLabel}
					<span
						class={[
							"shrink-0 font-mono",
							isPriceMuted ? "text-fg-faint" : "text-fg",
						]}>{priceLabel}</span>
				{/if}
			</span>
		{/if}
	</span>
</button>

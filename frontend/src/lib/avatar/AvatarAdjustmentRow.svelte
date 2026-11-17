<script lang="ts">
	import caretDownIcon from "@phosphor-icons/core/regular/caret-down.svg?raw"
	import type { Snippet } from "svelte"
	import type { AvatarAsset } from "../backend/bridge"
	import Icon from "../shared/Icon.svelte"
	import AvatarAssetSummary from "./AvatarAssetSummary.svelte"
	import { describeAdjustment } from "./adjustment-format"
	import { isDefaultAdjustment } from "./avatar-draft"

	let {
		asset,
		rowId,
		controlsId,
		isExpanded,
		isEdited = false,
		unavailableReason = "",
		onToggle,
		children,
	}: {
		asset: AvatarAsset
		rowId?: string
		controlsId: string
		isExpanded: boolean
		// isEdited marks a placement that differs from the saved one.
		isEdited?: boolean
		// unavailableReason replaces the placement when the row cannot open,
		// such as for an item that is not saved yet.
		unavailableReason?: string
		onToggle: () => void
		children: Snippet
	} = $props()

	const isAdjusted = $derived(!isDefaultAdjustment(asset.adjustment)),
		isExpandable = $derived(!unavailableReason || isAdjusted)
</script>

{#snippet rowContent()}
	<AvatarAssetSummary {asset} />
	{#if isEdited}
		<span
			class="shrink-0 animate-pop rounded-sm border border-line-strong bg-overlay px-1 text-2xs leading-4 font-medium text-fg"
			>Edited</span>
	{/if}
	<span
		class={[
			"max-w-48 shrink-0 truncate text-right text-xs",
			isAdjusted && !unavailableReason ? "text-fg" : "text-fg-muted",
		]}>{unavailableReason || describeAdjustment(asset.adjustment)}</span>
{/snippet}

<li id={rowId} class="flex flex-col border-line-subtle not-first:border-t">
	{#if isExpandable}
		<button
			class="group/row list-row flex h-12 min-w-0 items-center gap-2 px-3 text-left"
			type="button"
			data-open={isExpanded || undefined}
			aria-expanded={isExpanded}
			aria-controls={controlsId}
			onclick={onToggle}>
			{@render rowContent()}
			<Icon
				svg={caretDownIcon}
				class="shrink-0 text-fg-muted transition-transform duration-100 group-data-open/row:rotate-180"
				size={16}
				aria-hidden="true" />
		</button>
	{:else}
		<!-- The right padding keeps the status in line with the rows that end
		     in a caret: 12px of padding, a 16px caret, and an 8px gap. -->
		<div class="flex h-12 min-w-0 items-center gap-2 pr-9 pl-3">
			{@render rowContent()}
		</div>
	{/if}
	{#if isExpanded && isExpandable}
		<div
			id={controlsId}
			class="animate-enter border-t border-line-subtle bg-canvas px-3 py-3">
			{@render children()}
		</div>
	{/if}
</li>

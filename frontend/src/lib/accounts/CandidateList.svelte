<script lang="ts">
	import Check from "@lucide/svelte/icons/check"
	import Minus from "@lucide/svelte/icons/minus"
	import ProfileImage from "../shared/ProfileImage.svelte"
	import type { CandidateListItem } from "./candidate-model"

	let {
		items,
		selected,
		disabled = false,
		onToggle,
		onToggleAll,
		selectionTotal,
	}: {
		items: CandidateListItem[]
		selected: string[]
		disabled?: boolean
		onToggle: (id: string, selected: boolean) => void
		onToggleAll: () => void
		selectionTotal?: number
	} = $props()

	const selectableCount = $derived(
			selectionTotal ?? items.filter((item) => item.selectable).length,
		),
		allSelected = $derived(
			selectableCount > 0 && selected.length === selectableCount,
		),
		someSelected = $derived(selected.length > 0 && !allSelected)
</script>

<div class="cookie-import-results" aria-live="polite">
	<div class="cookie-results-toolbar">
		<label class="cookie-select-all">
			<span class="cookie-selection">
				<input
					type="checkbox"
					checked={allSelected}
					indeterminate={someSelected}
					disabled={disabled || selectableCount === 0}
					onchange={onToggleAll} />
				<span aria-hidden="true">
					{#if someSelected}<Minus size={10} strokeWidth={2.4} />{:else}<Check
							size={10}
							strokeWidth={2.4} />{/if}
				</span>
			</span>
			<span>{allSelected ? "Unselect all" : "Select all"}</span>
		</label>
		<span class="cookie-selection-count"
			>{selected.length} of {selectableCount} selected</span>
	</div>
	<ol>
		{#each items as item (item.id)}
			<li
				class:cookie-result-failed={!!item.error}
				class:cookie-result-duplicate={!!item.duplicateText}>
				<label class="cookie-selection">
					<input
						type="checkbox"
						checked={selected.includes(item.id)}
						disabled={disabled || !item.selectable}
						aria-label={item.selectionLabel}
						onchange={(event) =>
							onToggle(item.id, event.currentTarget.checked)} />
					<span aria-hidden="true"
						><Check size={10} strokeWidth={2.4} /></span>
				</label>
				{#if item.indexLabel}<span class="cookie-result-index"
						>{item.indexLabel}</span
					>{/if}
				{#if item.duplicateText}
					<span class="cookie-result-warning">{item.duplicateText}</span>
				{:else if item.error}
					<span class="cookie-result-error">{item.error}</span>
				{:else}
					<span class="cookie-result-avatar" aria-hidden="true">
						<ProfileImage url={item.avatarUrl ?? ""} />
					</span>
					<span class="cookie-result-account">
						<span>{item.displayName}</span>
						<span class="cookie-result-username">@{item.username}</span>
						<span class="cookie-result-id">- ID: {item.robloxUserId}</span>
					</span>
					{#if item.status}<span class="cookie-result-status"
							>{item.status}</span
						>{/if}
				{/if}
			</li>
		{/each}
	</ol>
</div>

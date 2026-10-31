<script lang="ts">
	import CheckIcon from "phosphor-svelte/lib/CheckIcon"
	import MinusIcon from "phosphor-svelte/lib/MinusIcon"
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

<div
	class="mt-3 overflow-hidden rounded-md border border-line bg-fill"
	aria-live="polite">
	<div
		class="flex min-h-8 items-center justify-between gap-2 border-b border-line-subtle bg-fill px-2 text-xs text-fg-muted">
		<label class="flex cursor-pointer items-center gap-1 font-medium">
			<span class="checkbox">
				<input
					type="checkbox"
					checked={allSelected}
					indeterminate={someSelected}
					disabled={disabled || selectableCount === 0}
					onchange={onToggleAll} />
				<span aria-hidden="true">
					{#if someSelected}<MinusIcon
							size={11}
							weight="bold"
							aria-hidden="true" />{:else}<CheckIcon
							size={11}
							weight="bold"
							aria-hidden="true" />{/if}
				</span>
			</span>
			<span>{allSelected ? "Unselect all" : "Select all"}</span>
		</label>
		<span class="font-mono text-fg-faint"
			>{selected.length} of {selectableCount} selected</span>
	</div>
	<ol class="flex max-h-44 flex-col overflow-x-hidden overflow-y-auto">
		{#each items as item (item.id)}
			<li
				class={[
					"flex min-h-9 items-center gap-2 px-2 py-1 text-xs leading-snug not-first:border-t not-first:border-line-subtle",
					item.error ? "text-danger-muted" : "text-fg-muted",
				]}>
				<label class="checkbox">
					<input
						type="checkbox"
						checked={selected.includes(item.id)}
						disabled={disabled || !item.selectable}
						aria-label={item.selectionLabel}
						onchange={(event) =>
							onToggle(item.id, event.currentTarget.checked)} />
					<span aria-hidden="true"
						><CheckIcon size={11} weight="bold" aria-hidden="true" /></span>
				</label>
				{#if item.indexLabel}<span
						class="w-5 shrink-0 text-right font-mono text-fg-faint"
						>{item.indexLabel}</span
					>{/if}
				{#if item.duplicateText}
					<span class="min-w-0 flex-1 wrap-anywhere text-warning"
						>{item.duplicateText}</span>
				{:else if item.error}
					<span class="min-w-0 flex-1 wrap-anywhere">{item.error}</span>
				{:else}
					<span
						class="relative grid size-6 shrink-0 place-items-center overflow-hidden rounded-md border border-line bg-raised text-fg-faint"
						aria-hidden="true">
						<ProfileImage url={item.avatarUrl ?? ""} />
					</span>
					<span
						class="flex min-w-0 flex-1 items-baseline gap-1 overflow-hidden whitespace-nowrap">
						<span class="min-w-0 truncate font-medium text-fg"
							>{item.displayName}</span>
						<span class="shrink-0 text-fg-faint">@{item.username}</span>
						<span class="shrink-0 font-mono text-fg-faint"
							>- ID: {item.robloxUserId}</span>
					</span>
					{#if item.status}<span
							class="rounded-sm border border-warning/30 px-1 py-0.5 text-2xs whitespace-nowrap text-warning"
							>{item.status}</span
						>{/if}
				{/if}
			</li>
		{/each}
	</ol>
</div>

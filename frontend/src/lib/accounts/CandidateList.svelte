<script lang="ts">
	import CheckboxMark from "../shared/CheckboxMark.svelte"
	import ProfileImage from "../shared/ProfileImage.svelte"
	import type { CandidateListItem } from "./candidate-model"
	import { moderationStatus } from "./moderation.svelte"

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
			selectionTotal ?? items.filter((candidate) => candidate.selectable).length,
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
				<CheckboxMark isIndeterminate={someSelected} />
			</span>
			<span>{allSelected ? "Clear selection" : "Select all"}</span>
		</label>
		<span class="font-mono text-fg-faint"
			>{selected.length} of {selectableCount} selected</span>
	</div>
	<ol class="flex max-h-44 flex-col overflow-x-hidden overflow-y-auto">
		{#each items as candidate (candidate.id)}
			<li
				class={[
					"flex min-h-9 items-center gap-2 px-2 py-1 text-xs leading-snug not-first:border-t not-first:border-line-subtle",
					candidate.error ? "text-danger-muted" : "text-fg-muted",
				]}>
				<label class="checkbox">
					<input
						type="checkbox"
						checked={selected.includes(candidate.id)}
						disabled={disabled || !candidate.selectable}
						aria-label={candidate.selectionLabel}
						onchange={(event) =>
							onToggle(candidate.id, event.currentTarget.checked)} />
					<CheckboxMark />
				</label>
				{#if candidate.indexLabel}<span
						class="w-5 shrink-0 text-right font-mono text-fg-faint"
						>{candidate.indexLabel}</span
					>{/if}
				{#if candidate.duplicateText}
					<span class="min-w-0 flex-1 wrap-anywhere text-warning"
						>{candidate.duplicateText}</span>
				{:else if candidate.error}
					<span class="min-w-0 flex-1 wrap-anywhere">{candidate.error}</span>
				{:else}
					<span
						class="relative grid size-6 shrink-0 place-items-center overflow-hidden rounded-md border border-line bg-raised text-fg-faint"
						aria-hidden="true">
						<ProfileImage url={candidate.avatarUrl ?? ""} />
					</span>
					<span
						class="flex min-w-0 flex-1 items-baseline gap-1 overflow-hidden whitespace-nowrap">
						<span class="min-w-0 truncate font-medium text-fg"
							>{candidate.displayName}</span>
						<span class="shrink-0 text-fg-faint"
							>@{candidate.username}</span>
						<span class="shrink-0 font-mono text-fg-faint"
							>- ID: {candidate.robloxUserId}</span>
					</span>
					{#if candidate.moderation}
						{@const moderation = moderationStatus(
							candidate.moderation,
							Date.now(),
						)}
						<span
							class={[
								"rounded-sm border px-1 py-0.5 text-2xs whitespace-nowrap tabular-nums",
								moderation.tone === "warning"
									? "border-warning/30 text-warning"
									: "border-danger/40 text-danger-muted",
							]}
							>{moderation.label}{moderation.detail
								? ` ${moderation.detail}`
								: ""}</span>
					{/if}
					{#if candidate.status}<span
							class="rounded-sm border border-warning/30 px-1 py-0.5 text-2xs whitespace-nowrap text-warning"
							>{candidate.status}</span
						>{/if}
				{/if}
			</li>
		{/each}
	</ol>
</div>

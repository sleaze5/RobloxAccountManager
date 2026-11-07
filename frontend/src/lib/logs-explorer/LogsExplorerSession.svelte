<script lang="ts">
	import sortAscendingIcon from "@phosphor-icons/core/regular/sort-ascending.svg?raw"
	import sortDescendingIcon from "@phosphor-icons/core/regular/sort-descending.svg?raw"
	import warningIcon from "@phosphor-icons/core/regular/warning.svg?raw"
	import Icon from "../shared/Icon.svelte"
	import type { LogsExplorerSession, LogsExplorerVisit } from "../backend/bridge"
	import LogsExplorerTimeline from "./LogsExplorerTimeline.svelte"
	import LogsExplorerLogDetails from "./LogsExplorerLogDetails.svelte"

	let {
		session,
		newestFirst = $bindable(),
		refreshing,
		onRefresh,
		onFillLaunch,
		onSearchPlace,
	}: {
		session: LogsExplorerSession
		newestFirst: boolean
		refreshing: boolean
		onRefresh: () => void
		onFillLaunch: (visit: LogsExplorerVisit) => void
		onSearchPlace: (placeId: number) => void
	} = $props()
	const visits = $derived(session.visits ?? [])
</script>

<section
	class="grid min-h-0 min-w-0 animate-enter grid-rows-1"
	aria-label="Selected launch session"
	aria-busy={refreshing}>
	<div class="min-h-0 overflow-y-auto">
		<div class="flex flex-col gap-3 p-3">
			<LogsExplorerLogDetails {session} {refreshing} {onRefresh} />
			{#if session.issue}
				<p class="flex items-start gap-2 notice-warning" role="status">
					<Icon
						svg={warningIcon}
						class="shrink-0"
						size={16}
						aria-hidden="true" />{session.issue}
				</p>
			{/if}
			<div class="flex items-center gap-2 border-b border-line pb-2">
				<h2 class="text-base font-semibold text-fg">Timeline</h2>
				<button
					class="button-icon"
					type="button"
					aria-label="Newest visits first"
					aria-pressed={newestFirst}
					data-tooltip={newestFirst
						? "Newest first. Switch to oldest first"
						: "Oldest first. Switch to newest first"}
					onclick={() => (newestFirst = !newestFirst)}>
					{#if newestFirst}
						<Icon svg={sortDescendingIcon} size={16} aria-hidden="true" />
					{:else}
						<Icon svg={sortAscendingIcon} size={16} aria-hidden="true" />
					{/if}
				</button>
			</div>
			{#if visits.length}
				<LogsExplorerTimeline
					{visits}
					logStartedAtMs={session.startedAtMs}
					{newestFirst}
					{onFillLaunch}
					{onSearchPlace} />
			{:else}
				<p class="text-xs text-fg-muted">No game joins recorded.</p>
			{/if}
		</div>
	</div>
</section>

<script lang="ts">
	import WarningIcon from "phosphor-svelte/lib/WarningIcon"
	import SortAscendingIcon from "phosphor-svelte/lib/SortAscendingIcon"
	import SortDescendingIcon from "phosphor-svelte/lib/SortDescendingIcon"
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
	class="logs-explorer-session"
	aria-label="Selected launch session"
	aria-busy={refreshing}>
	<div class="logs-explorer-session-scroll">
		<div class="logs-explorer-session-content">
			<LogsExplorerLogDetails {session} {refreshing} {onRefresh} />
			{#if session.issue}
				<p class="logs-explorer-warning" role="status">
					<WarningIcon size={16} aria-hidden="true" />{session.issue}
				</p>
			{/if}
			<div class="logs-explorer-timeline-heading">
				<h2>Timeline</h2>
				<button
					class="icon-action bordered"
					type="button"
					aria-label="Newest visits first"
					aria-pressed={newestFirst}
					data-tooltip={newestFirst
						? "Newest first. Switch to oldest first"
						: "Oldest first. Switch to newest first"}
					onclick={() => (newestFirst = !newestFirst)}>
					{#if newestFirst}
						<SortDescendingIcon size={16} aria-hidden="true" />
					{:else}
						<SortAscendingIcon size={16} aria-hidden="true" />
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
				<p class="logs-explorer-empty-timeline">No game joins recorded.</p>
			{/if}
		</div>
	</div>
</section>

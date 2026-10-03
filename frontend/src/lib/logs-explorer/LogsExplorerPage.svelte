<script lang="ts">
	import { onMount } from "svelte"
	import AlertTriangle from "@lucide/svelte/icons/triangle-alert"
	import ChevronLeft from "@lucide/svelte/icons/chevron-left"
	import ChevronRight from "@lucide/svelte/icons/chevron-right"
	import FileSearch from "@lucide/svelte/icons/file-search"
	import FileText from "@lucide/svelte/icons/file-text"
	import LoaderCircle from "@lucide/svelte/icons/loader-circle"
	import RefreshCw from "@lucide/svelte/icons/refresh-cw"
	import Search from "@lucide/svelte/icons/search"
	import X from "@lucide/svelte/icons/x"
	import { appSettings } from "../settings/settings-store.svelte"
	import type { LogsExplorerVisit } from "../backend/bridge"
	import SidebarResizer from "../layout/SidebarResizer.svelte"
	import OperationError from "../shared/OperationError.svelte"
	import { formatTimestamp } from "../shared/timestamp"
	import LogsExplorerSession from "./LogsExplorerSession.svelte"
	import type { LogsExplorerStore } from "./logs-explorer-store.svelte"

	let {
		store,
		onFillLaunch,
	}: { store: LogsExplorerStore; onFillLaunch: (visit: LogsExplorerVisit) => void } =
		$props()
	const sessions = $derived(store.filteredSessions),
		selected = $derived(store.selectedSession)

	onMount(() => store.initialize())
</script>

<main class="logs-explorer-page" aria-label="Logs Explorer">
	<OperationError {store} />
	<div
		class="workbench"
		style={`--sidebar-width: ${store.sidebarCollapsed ? 42 : store.sidebarWidth}px;`}>
		<aside
			class="account-explorer"
			class:collapsed={store.sidebarCollapsed}
			aria-label="Launch sessions">
			{#if store.sidebarCollapsed}
				<div class="collapsed-sidebar">
					<button
						type="button"
						aria-label="Expand logs sidebar"
						data-tooltip="Expand the logs sidebar"
						data-tooltip-side="right"
						onclick={() => (store.sidebarCollapsed = false)}
						><ChevronRight size={16} /></button>
					<FileSearch size={16} aria-hidden="true" />
					<span>{store.sessions.length}</span>
				</div>
			{:else}
				<div class="explorer-controls">
					<div class="account-title">
						<h1>Logs <span>({store.sessions.length})</span></h1>
						<div class="account-title-actions">
							<button
								type="button"
								disabled={store.refreshing}
								aria-busy={store.refreshing}
								onclick={() => void store.refreshAll()}
								aria-label={store.refreshing
									? "Refreshing all logs"
									: "Refresh all logs"}
								data-tooltip="Refresh all logs"
								data-tooltip-side="bottom-end">
								{#if store.refreshing}<LoaderCircle
										class="spinner"
										size={14}
										aria-hidden="true" />{:else}<RefreshCw
										size={14}
										aria-hidden="true" />{/if}
							</button>
							<button
								type="button"
								aria-label="Collapse logs sidebar"
								data-tooltip="Collapse the logs sidebar"
								data-tooltip-side="bottom-end"
								onclick={() => (store.sidebarCollapsed = true)}
								><ChevronLeft size={15} /></button>
						</div>
					</div>
					<div class="account-filters">
						<div class="search-field">
							<Search size={14} aria-hidden="true" /><input
								type="text"
								placeholder="Search logs or IDs"
								aria-label="Search logs, user IDs, place IDs, or job IDs"
								bind:value={store.query} />{#if store.query}<button
									class="search-clear"
									type="button"
									aria-label="Clear logs search"
									onclick={() => (store.query = "")}
									><X size={12} /></button
								>{/if}
						</div>
					</div>
				</div>
				<div class="logs-explorer-session-list" aria-busy={store.refreshing}>
					{#each sessions as session (session.fileName)}
						{@const visitCount = session.visits?.length ?? 0}
						<button
							class="logs-explorer-session-row"
							class:active={store.selectedFile === session.fileName}
							type="button"
							aria-current={store.selectedFile === session.fileName
								? "true"
								: undefined}
							onclick={() => (store.selectedFile = session.fileName)}
							data-tooltip={session.fileName}
							data-tooltip-side="right">
							<span class="logs-explorer-row-title"
								><FileText size={14} aria-hidden="true" /><span
									>{session.startedAtMs
										? formatTimestamp(
												session.startedAtMs,
												appSettings.timestampFormat,
											)
										: "Unknown start time"}</span
								>{#if session.issue}<AlertTriangle
										class="logs-explorer-issue-icon"
										size={13}
										aria-label="Incomplete log" />{/if}</span>
							<span class="logs-explorer-row-meta"
								>{visitCount}
								{visitCount === 1 ? "visit" : "visits"}</span>
						</button>
					{:else}
						{#if store.refreshing}
							<p class="logs-explorer-list-empty">Loading sessions…</p>
						{:else if store.query}
							<p class="logs-explorer-list-empty">
								No matching sessions.
							</p>
						{/if}
					{/each}
				</div>
			{/if}
		</aside>
		<SidebarResizer
			bind:width={store.sidebarWidth}
			collapsed={store.sidebarCollapsed}
			name="logs" />
		{#if selected}
			{#key selected.fileName}
				<LogsExplorerSession
					session={selected}
					bind:newestFirst={store.timelineNewestFirst}
					refreshing={store.refreshingFiles.has(selected.fileName)}
					{onFillLaunch}
					onRefresh={() => void store.refreshSession(selected.fileName)} />
			{/key}
		{:else}
			<div class="logs-explorer-empty" role="status">
				{#if store.refreshing}<LoaderCircle
						class="spinner"
						size={24}
						aria-hidden="true" />
					<h2>Reading Roblox logs…</h2>
				{:else if store.error && !store.loaded}<AlertTriangle
						size={24}
						aria-hidden="true" />
					<h2>Logs are unavailable</h2>
					<p>Refresh all to try reading the logs again.</p>
				{:else if !store.sessions.length}<FileSearch
						size={24}
						aria-hidden="true" />
					<h2>No Roblox sessions found</h2>
				{:else if !sessions.length}<Search size={24} aria-hidden="true" />
					<h2>No matching sessions</h2>
					<p>Try a different file name, user ID, or place ID.</p>
				{:else}<FileSearch size={24} aria-hidden="true" />
					<h2>Select a launch session</h2>
					<p>Choose a log to explore its game timeline.</p>{/if}
			</div>
		{/if}
	</div>
	<footer class="status-bar" aria-label="Log details">
		<span>User ID: {selected?.userId || "Not recorded"}</span>
		<div>
			<span>Channel: {selected?.channel || "Not recorded"}</span>
			<span>Client version: {selected?.version || "Not recorded"}</span>
		</div>
	</footer>
</main>

<script lang="ts">
	import { onMount } from "svelte"
	import WarningIcon from "phosphor-svelte/lib/WarningIcon"
	import CaretLeftIcon from "phosphor-svelte/lib/CaretLeftIcon"
	import CaretRightIcon from "phosphor-svelte/lib/CaretRightIcon"
	import FileMagnifyingGlassIcon from "phosphor-svelte/lib/FileMagnifyingGlassIcon"
	import FileTextIcon from "phosphor-svelte/lib/FileTextIcon"
	import CircleNotchIcon from "phosphor-svelte/lib/CircleNotchIcon"
	import ArrowsClockwiseIcon from "phosphor-svelte/lib/ArrowsClockwiseIcon"
	import MagnifyingGlassIcon from "phosphor-svelte/lib/MagnifyingGlassIcon"
	import XIcon from "phosphor-svelte/lib/XIcon"
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
		onSearchPlace,
	}: {
		store: LogsExplorerStore
		onFillLaunch: (visit: LogsExplorerVisit) => void
		onSearchPlace: (placeId: number) => void
	} = $props()
	const sessions = $derived(store.filteredSessions),
		selected = $derived(store.selectedSession)

	onMount(() => store.initialize())
</script>

<main class="flex min-h-0 flex-1 flex-col bg-canvas" aria-label="Logs Explorer">
	<OperationError {store} />
	<div
		class="flex min-h-0 flex-1"
		style={`--sidebar-width: ${store.sidebarCollapsed ? 44 : store.sidebarWidth}px;`}>
		<aside
			class="flex min-h-0 w-(--sidebar-width) shrink-0 flex-col bg-surface"
			aria-label="Launch sessions">
			{#if store.sidebarCollapsed}
				<div class="flex h-full flex-col items-center gap-3 pt-2 text-fg-faint">
					<button
						class="btn-ghost"
						type="button"
						aria-label="Expand logs sidebar"
						data-tooltip="Expand the logs sidebar"
						data-tooltip-side="right"
						onclick={() => (store.sidebarCollapsed = false)}
						><CaretRightIcon
							data-nudge="right"
							size={16}
							aria-hidden="true" /></button>
					<FileMagnifyingGlassIcon size={16} aria-hidden="true" />
					<span class="font-mono text-2xs">{store.sessions.length}</span>
				</div>
			{:else}
				<div class="flex shrink-0 flex-col gap-1 px-3 pt-1 pb-2">
					<div class="relative flex h-7 items-center">
						<h1>
							Logs <span
								class="font-mono text-2xs font-normal text-fg-faint"
								>({store.sessions.length})</span>
						</h1>
						<div class="ml-auto flex items-center gap-px">
							<button
								class="btn-ghost"
								type="button"
								disabled={store.refreshing}
								aria-busy={store.refreshing}
								onclick={() => void store.refreshAll()}
								aria-label={store.refreshing
									? "Refreshing all logs"
									: "Refresh all logs"}
								data-tooltip="Refresh all logs"
								data-tooltip-side="bottom-end">
								{#if store.refreshing}<CircleNotchIcon
										class="spinner"
										size={16}
										aria-hidden="true" />{:else}<ArrowsClockwiseIcon
										data-turn
										size={16}
										aria-hidden="true" />{/if}
							</button>
							<button
								class="btn-ghost"
								type="button"
								aria-label="Collapse logs sidebar"
								data-tooltip="Collapse the logs sidebar"
								data-tooltip-side="bottom-end"
								onclick={() => (store.sidebarCollapsed = true)}
								><CaretLeftIcon
									data-nudge="left"
									size={16}
									aria-hidden="true" /></button>
						</div>
					</div>
					<div class="flex items-center gap-1">
						<div class="search-field">
							<MagnifyingGlassIcon size={16} aria-hidden="true" /><input
								type="text"
								placeholder="Search logs or IDs"
								aria-label="Search logs, user IDs, place IDs, or job IDs"
								bind:value={store.query} />{#if store.query}<button
									type="button"
									aria-label="Clear logs search"
									onclick={() => (store.query = "")}
									><XIcon size={14} aria-hidden="true" /></button
								>{/if}
						</div>
					</div>
				</div>
				<div
					class="min-h-0 flex-1 scrollbar-stable overflow-y-auto"
					aria-busy={store.refreshing}>
					{#each sessions as session (session.fileName)}
						{@const visitCount = session.visits?.length ?? 0}
						<button
							class="list-row flex min-h-12 w-full flex-col justify-center border-b border-line-subtle px-3 py-2 text-left leading-snug"
							data-selected={store.selectedFile === session.fileName ||
								undefined}
							type="button"
							aria-current={store.selectedFile === session.fileName
								? "true"
								: undefined}
							onclick={() => (store.selectedFile = session.fileName)}
							data-tooltip={session.fileName}
							data-tooltip-side="right">
							<span
								class="flex w-full items-center gap-2 text-xs font-semibold text-fg"
								><FileTextIcon
									class="shrink-0 text-fg-muted"
									size={16}
									aria-hidden="true" /><span class="truncate"
									>{session.startedAtMs
										? formatTimestamp(
												session.startedAtMs,
												appSettings.timestampFormat,
											)
										: "Unknown start time"}</span
								>{#if session.issue}<WarningIcon
										class="shrink-0 text-warning"
										size={14}
										aria-label="Incomplete log" />{/if}</span>
							<span
								class="flex w-full items-center gap-2 pl-6 text-2xs text-fg-muted"
								>{visitCount}
								{visitCount === 1 ? "visit" : "visits"}</span>
						</button>
					{:else}
						{#if store.refreshing}
							<p class="p-3 text-2xs text-fg-muted">Loading sessions…</p>
						{:else if store.query}
							<p class="p-3 text-2xs text-fg-muted">
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
		<div class="grid min-w-0 flex-1 grid-rows-1">
			{#if selected}
				{#key selected.fileName}
					<LogsExplorerSession
						session={selected}
						bind:newestFirst={store.timelineNewestFirst}
						refreshing={store.refreshingFiles.has(selected.fileName)}
						{onFillLaunch}
						{onSearchPlace}
						onRefresh={() =>
							void store.refreshSession(selected.fileName)} />
				{/key}
			{:else}
				<div class="empty-state" role="status">
					{#if store.refreshing}<CircleNotchIcon
							class="spinner"
							size={28}
							aria-hidden="true" />
						<h2>Reading Roblox logs…</h2>
					{:else if store.error && !store.loaded}<WarningIcon
							size={28}
							aria-hidden="true" />
						<h2>Logs are unavailable</h2>
						<p>Refresh all to try reading the logs again.</p>
					{:else if !store.sessions.length}<FileMagnifyingGlassIcon
							size={28}
							aria-hidden="true" />
						<h2>No Roblox sessions found</h2>
					{:else if !sessions.length}<MagnifyingGlassIcon
							size={28}
							aria-hidden="true" />
						<h2>No matching sessions</h2>
						<p>Try a different file name, user ID, or place ID.</p>
					{:else}<FileMagnifyingGlassIcon size={28} aria-hidden="true" />
						<h2>Select a launch session</h2>
						<p>Choose a log to explore its game timeline.</p>{/if}
				</div>
			{/if}
		</div>
	</div>
	<footer
		class="flex h-6 shrink-0 items-center justify-between border-t border-line-subtle bg-surface pl-3 font-mono text-2xs text-fg-faint"
		aria-label="Log details">
		{#if selected}
			<span>User ID: {selected.userId || "Not recorded"}</span>
			<div class="flex items-center gap-4 pr-3">
				<span>Channel: {selected.channel || "Not recorded"}</span>
				<span
					>Client version: {selected.versionHash && selected.version
						? `${selected.versionHash} (${selected.version})`
						: selected.version || "Not recorded"}</span>
			</div>
		{:else}
			<span>Session: None</span>
		{/if}
	</footer>
</main>

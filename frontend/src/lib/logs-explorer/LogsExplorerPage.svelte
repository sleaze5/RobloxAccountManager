<script lang="ts">
	import { onMount } from "svelte"
	import arrowsClockwiseIcon from "@phosphor-icons/core/regular/arrows-clockwise.svg?raw"
	import circleNotchIcon from "@phosphor-icons/core/regular/circle-notch.svg?raw"
	import fileMagnifyingGlassIcon from "@phosphor-icons/core/regular/file-magnifying-glass.svg?raw"
	import fileTextIcon from "@phosphor-icons/core/regular/file-text.svg?raw"
	import magnifyingGlassIcon from "@phosphor-icons/core/regular/magnifying-glass.svg?raw"
	import warningIcon from "@phosphor-icons/core/regular/warning.svg?raw"
	import xIcon from "@phosphor-icons/core/regular/x.svg?raw"
	import Icon from "../shared/Icon.svelte"
	import SkeletonList from "../shared/SkeletonList.svelte"
	import RefreshIcon from "../shared/RefreshIcon.svelte"
	import { moveListFocus } from "../shared/move-shortcut"
	import { appSettings } from "../settings/settings-store.svelte"
	import type { LogsExplorerVisit } from "../backend/bridge"
	import WorkbenchSidebar from "../layout/WorkbenchSidebar.svelte"
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
		<WorkbenchSidebar
			label="Launch sessions"
			name="logs"
			icon={fileMagnifyingGlassIcon}
			count={store.sessions.length}
			bind:collapsed={store.sidebarCollapsed}
			bind:width={store.sidebarWidth}>
			{#snippet title()}Logs <span
					class="font-mono text-2xs font-normal text-fg-faint"
					>({store.sessions.length})</span
				>{/snippet}
			{#snippet actions()}
				<button
					class="button-ghost"
					type="button"
					disabled={store.refreshing}
					aria-busy={store.refreshing}
					onclick={() => void store.refreshAll()}
					aria-label={store.refreshing
						? "Refreshing all logs"
						: "Refresh all logs"}
					data-tooltip="Refresh all logs"
					data-tooltip-side="bottom-end">
					<RefreshIcon isRefreshing={store.refreshing} />
				</button>
			{/snippet}
			{#snippet tools()}
				<div class="flex items-center gap-1">
					<div class="search-field">
						<Icon
							svg={magnifyingGlassIcon}
							size={16}
							aria-hidden="true" /><input
							type="text"
							placeholder="Search logs or IDs"
							aria-label="Search logs, user IDs, place IDs, or job IDs"
							bind:value={store.query} />{#if store.query}<button
								type="button"
								aria-label="Clear logs search"
								onclick={() => (store.query = "")}
								><Icon
									svg={xIcon}
									size={14}
									aria-hidden="true" /></button
							>{/if}
					</div>
				</div>
			{/snippet}
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
						data-list-item
						onclick={() => (store.selectedFile = session.fileName)}
						onkeydown={moveListFocus}
						data-tooltip={session.fileName}
						data-tooltip-side="right">
						<span
							class="flex w-full items-center gap-2 text-xs font-semibold text-fg"
							><Icon
								svg={fileTextIcon}
								class="shrink-0 text-fg-muted"
								size={16}
								aria-hidden="true" /><span class="truncate"
								>{session.startedAtMs
									? formatTimestamp(
											session.startedAtMs,
											appSettings.timestampFormat,
										)
									: "Unknown start time"}</span
							>{#if session.issue}<Icon
									svg={warningIcon}
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
						<SkeletonList label="Loading sessions">
							{#snippet row(index)}
								<div
									class="flex h-12 flex-col justify-center gap-1 border-b border-line-subtle px-3">
									<span class="text-xs"
										><span
											class={[
												"skeleton-line",
												index % 2 ? "w-32" : "w-40",
											]}></span
										></span>
									<span class="pl-6 text-2xs"
										><span class="skeleton-line w-14"></span></span>
								</div>
							{/snippet}
						</SkeletonList>
					{:else if store.query}
						<p class="p-3 text-2xs text-fg-muted">No matching sessions.</p>
					{/if}
				{/each}
			</div>
		</WorkbenchSidebar>
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
					{#if store.refreshing}<Icon
							svg={circleNotchIcon}
							class="spinner"
							size={28}
							aria-hidden="true" />
						<h2>Reading Roblox logs…</h2>
					{:else if store.error && !store.loaded}<Icon
							svg={warningIcon}
							size={28}
							aria-hidden="true" />
						<h2>Logs are unavailable</h2>
						<p>Refresh the logs to try reading them again.</p>
						<button
							class="button"
							type="button"
							onclick={() => void store.refreshAll()}
							><Icon
								svg={arrowsClockwiseIcon}
								data-turn
								size={16}
								aria-hidden="true" />Refresh logs</button>
					{:else if !store.sessions.length}<Icon
							svg={fileMagnifyingGlassIcon}
							size={28}
							aria-hidden="true" />
						<h2>No Roblox sessions yet</h2>
						<p>Launch Roblox once, then refresh to read its logs.</p>
						<button
							class="button"
							type="button"
							onclick={() => void store.refreshAll()}
							><Icon
								svg={arrowsClockwiseIcon}
								data-turn
								size={16}
								aria-hidden="true" />Refresh logs</button>
					{:else if !sessions.length}<Icon
							svg={magnifyingGlassIcon}
							size={28}
							aria-hidden="true" />
						<h2>No matching sessions</h2>
						<p>Try a different file name, user ID, or place ID.</p>
					{:else}<Icon
							svg={fileMagnifyingGlassIcon}
							size={28}
							aria-hidden="true" />
						<h2>Select a launch session</h2>
						<p>Choose a log to explore its game timeline.</p>{/if}
				</div>
			{/if}
		</div>
	</div>
	<footer
		class="flex h-6 shrink-0 items-center justify-between border-t border-line-subtle bg-surface pl-3 text-2xs text-fg-faint"
		aria-label="Log details">
		{#if selected}
			<span
				>User ID: <span class="font-mono"
					>{selected.userId || "Not recorded"}</span
				></span>
			<div class="flex items-center gap-4 pr-3">
				<span
					>Channel: <span class="font-mono"
						>{selected.channel || "Not recorded"}</span
					></span>
				<span
					>Client version: <span class="font-mono"
						>{selected.version || "Not recorded"}</span
					></span>
			</div>
		{/if}
	</footer>
</main>

<script lang="ts">
	import { notifyErrors } from "../notifications/error-notifications.svelte"
	import LoadMore from "../shared/LoadMore.svelte"
	import hardDrivesIcon from "@phosphor-icons/core/regular/hard-drives.svg?raw"
	import warningIcon from "@phosphor-icons/core/regular/warning.svg?raw"
	import Icon from "../shared/Icon.svelte"
	import ServerTileSkeleton from "./ServerTileSkeleton.svelte"
	import { untrack } from "svelte"
	import { GameServerRecordOrder } from "../backend/bridge"
	import Select from "../shared/Select.svelte"
	import ServerTile from "./ServerTile.svelte"
	import type { ServerMenuOpener } from "./server-row-menu"
	import {
		type RecordedServers,
		type RecordPageSize,
		regionLabel,
		type ServerDetailsLoader,
		type ServerStatsLoader,
	} from "./game-servers-state.svelte"

	let {
		servers,
		stats,
		details,
		activeServerId,
		menuID,
		onMenu,
	}: {
		servers: RecordedServers
		stats: ServerStatsLoader
		details: ServerDetailsLoader
		activeServerId: string | undefined
		menuID: string
		onMenu: ServerMenuOpener
	} = $props()

	const numbers = new Intl.NumberFormat()
	let list = $state<HTMLElement | undefined>(undefined)
	const shown = $derived(servers.page?.servers ?? []),
		newestVersion = $derived(stats.stats?.newestVersion ?? 0),
		regionOptions = $derived([
			{
				value: "",
				label: stats.loading
					? "Loading regions…"
					: stats.failed
						? "Regions unavailable"
						: "Any region",
			},
			...(stats.stats?.regions ?? []).map((region) => ({
				value: region.code,
				label: `${regionLabel(region)} (${numbers.format(region.servers)})`,
			})),
		])

	$effect(() => {
		void servers.listVersion
		untrack(() => list?.scrollTo({ top: 0 }))
	})

	notifyErrors(
		"RoValra servers not loaded",
		() => servers.error || servers.loadMoreError,
	)
</script>

<div class="flex min-h-0 flex-1 flex-col">
	<div
		class="flex flex-wrap items-end gap-x-3 gap-y-2 border-b border-line-subtle p-3"
		role="group"
		aria-label="Server filters">
		<div class="flex flex-col gap-1 text-2xs text-fg-muted">
			<span>Region</span>
			<Select
				className="w-56"
				label="Server region"
				value={servers.region}
				options={regionOptions}
				onChange={(region) => servers.setOptions({ region })} />
		</div>
		<div
			class="flex flex-col gap-1 text-2xs text-fg-muted"
			data-tooltip={servers.region
				? "RoValra lists a region's servers newest first"
				: undefined}>
			<span>Sort</span>
			<Select
				label="Sort servers"
				value={servers.order}
				disabled={!!servers.region}
				options={[
					{
						value: GameServerRecordOrder.RecordOrderNewest,
						label: "Newest first",
					},
					{
						value: GameServerRecordOrder.RecordOrderOldest,
						label: "Oldest first",
					},
				]}
				onChange={(order) => servers.setOptions({ order })} />
		</div>
		<div class="flex flex-col gap-1 text-2xs text-fg-muted">
			<span>Per page</span>
			<Select
				className="w-20"
				label="Servers per page"
				value={servers.limit}
				options={[10, 50, 100].map((value) => ({
					value: value as RecordPageSize,
					label: String(value),
				}))}
				onChange={(value) => servers.setOptions({ limit: value })} />
		</div>
	</div>

	{#if servers.error}
		<div class="empty-state">
			<Icon svg={warningIcon} size={28} aria-hidden="true" />
			<h2>RoValra servers unavailable</h2>
			<button class="button" type="button" onclick={() => void servers.load()}
				>Try again</button>
		</div>
	{:else if !servers.page}
		<ServerTileSkeleton label="Loading RoValra servers…" isRecorded />
	{:else if !shown.length}
		<div class="empty-state" role="status">
			<Icon svg={hardDrivesIcon} size={28} aria-hidden="true" />
			{#if servers.region}
				<h2>No servers in this region</h2>
				<p>RoValra has not seen a server of this place in this region.</p>
				<button
					class="button"
					type="button"
					onclick={() => servers.setOptions({ region: "" })}
					>Any region</button>
			{:else}
				<h2>No servers recorded</h2>
				<p>RoValra has not seen a server of this place recently.</p>
			{/if}
		</div>
	{:else}
		<p class="flex gap-1 px-3 pt-2 font-mono text-2xs text-fg-faint" role="status">
			<span
				>{numbers.format(shown.length)}{servers.hasNext ? "+" : ""}
				{shown.length === 1 ? "server" : "servers"}</span>
			{#if stats.stats?.totalServers}
				<span aria-hidden="true">·</span>
				<span>{numbers.format(stats.stats.totalServers)} recorded</span>
			{/if}
			<span aria-hidden="true">·</span>
			<span>Some may have closed</span>
		</p>
		<div class="server-tiles" data-load-more-root bind:this={list}>
			<h2 class="sr-only">Servers</h2>
			<ol
				aria-label="Servers"
				aria-busy={servers.loading}
				data-loading={servers.loading || undefined}>
				{#each shown as server (server.serverId)}
					<ServerTile
						serverId={server.serverId}
						record={server}
						details={details.get(server.serverId)}
						{newestVersion}
						isMenuOpen={activeServerId === server.serverId}
						{menuID}
						{onMenu}
						onVisible={() =>
							details.request(servers.placeId, server.serverId)} />
				{/each}
			</ol>
			{#if servers.hasNext}
				<LoadMore
					pageKey={servers.page?.servers?.length}
					label="Load more servers"
					loadingLabel="Loading more servers…"
					isLoadingMore={servers.loadingMore}
					hasFailed={!!servers.loadMoreError}
					onLoadMore={() => void servers.loadMore()} />
			{/if}
		</div>
	{/if}
</div>

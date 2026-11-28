<script lang="ts">
	import { notifyErrors } from "../notifications/error-notifications.svelte"
	import LoadMore from "../shared/LoadMore.svelte"
	import hardDrivesIcon from "@phosphor-icons/core/regular/hard-drives.svg?raw"
	import warningIcon from "@phosphor-icons/core/regular/warning.svg?raw"
	import Icon from "../shared/Icon.svelte"
	import ServerTileSkeleton from "./ServerTileSkeleton.svelte"
	import { untrack } from "svelte"
	import { GameServerOrder } from "../backend/bridge"
	import Select from "../shared/Select.svelte"
	import FriendsInGameList from "./FriendsInGameList.svelte"
	import ServerTile from "./ServerTile.svelte"
	import type { ServerMenuOpener } from "./server-row-menu"
	import {
		type FriendsInGameLoader,
		type GameServers,
		serverPlace,
		serverPlaceVersion,
		type ServerDetailsLoader,
		type ServerPageSize,
	} from "./game-servers-state.svelte"

	let {
		servers,
		details,
		friendsInGame,
		records,
		newestVersion: newestRecordedVersion,
		activeServerId,
		menuID,
		onMenu,
	}: {
		servers: GameServers
		details: ServerDetailsLoader
		// friendsInGame is null when no account is shown.
		friendsInGame: FriendsInGameLoader | null
		records: boolean
		newestVersion: number
		activeServerId: string | undefined
		menuID: string
		onMenu: ServerMenuOpener
	} = $props()

	const filterID = $props.id(),
		numbers = new Intl.NumberFormat(),
		decimals = new Intl.NumberFormat(undefined, { maximumFractionDigits: 1 })
	let list = $state<HTMLElement | undefined>(undefined)
	const loaded = $derived(servers.page?.servers ?? []),
		listedServerById = $derived(
			new Map(loaded.map((server) => [server.serverId, server])),
		),
		placeByServerId = $derived(
			new Map(
				loaded.map((server) => [
					server.serverId,
					serverPlace(server.record, details.get(server.serverId)),
				]),
			),
		),
		filtered = $derived(servers.servers),
		shown = $derived(
			servers.region
				? filtered.filter(
						(server) =>
							placeByServerId.get(server.serverId) === servers.region,
					)
				: filtered,
		),
		players = $derived(shown.reduce((total, server) => total + server.playing, 0)),
		openSlots = $derived(
			shown.reduce(
				(total, server) =>
					total + Math.max(server.maxPlayers - server.playing, 0),
				0,
			),
		),
		averageFps = $derived(
			shown.length
				? shown.reduce((total, server) => total + server.fps, 0) / shown.length
				: 0,
		),
		versions = $derived(
			loaded
				.map((server) =>
					serverPlaceVersion(server.record, details.get(server.serverId)),
				)
				.filter(Boolean),
		),
		// The newest version that RoValra or a loaded server runs, so that a
		// server on an older version shows as outdated also without RoValra.
		newestVersion = $derived(Math.max(newestRecordedVersion, ...versions, 0)),
		outdatedCount = $derived(
			versions.filter((version) => version < newestVersion).length,
		),
		regions = $derived(regionSummaries()),
		regionOptions = $derived([
			{ value: "", label: "Any region" },
			...regions.map((region) => ({
				value: region.place,
				label: `${region.place} · ${region.servers} ${region.servers === 1 ? "server" : "servers"} · ${region.occupancyPercent}% full · ${numbers.format(region.openSlots)} open`,
			})),
			...(servers.region &&
			!regions.some((region) => region.place === servers.region)
				? [{ value: servers.region, label: servers.region }]
				: []),
		])

	// regionSummaries counts the loaded servers of each place, with their
	// average occupancy and open slots, from the most servers to the fewest.
	function regionSummaries(): {
		place: string
		servers: number
		occupancyPercent: number
		openSlots: number
	}[] {
		const summaryByPlace = new Map<
			string,
			{ servers: number; occupancy: number; openSlots: number }
		>()
		for (const server of loaded) {
			const place = placeByServerId.get(server.serverId)
			if (!place) continue
			const summary = summaryByPlace.get(place) ?? {
				servers: 0,
				occupancy: 0,
				openSlots: 0,
			}
			summary.servers++
			summary.occupancy += server.maxPlayers
				? server.playing / server.maxPlayers
				: 0
			summary.openSlots += Math.max(server.maxPlayers - server.playing, 0)
			summaryByPlace.set(place, summary)
		}
		return [...summaryByPlace]
			.map(([place, summary]) => ({
				place,
				servers: summary.servers,
				occupancyPercent: Math.round(
					(summary.occupancy / summary.servers) * 100,
				),
				openSlots: summary.openSlots,
			}))
			.sort(
				(left, right) =>
					right.servers - left.servers ||
					left.place.localeCompare(right.place),
			)
	}

	function clearFilters(): void {
		servers.maxPing = 0
		servers.minFps = 0
		servers.region = ""
	}

	$effect(() => {
		void servers.listVersion
		untrack(() => list?.scrollTo({ top: 0 }))
	})

	notifyErrors("Servers not loaded", () => servers.error || servers.loadMoreError)
</script>

<div class="server-tiles" data-load-more-root bind:this={list}>
	{#if friendsInGame}
		<FriendsInGameList
			{friendsInGame}
			{details}
			placeId={servers.placeId}
			{listedServerById}
			{newestVersion}
			activeKey={activeServerId}
			{menuID}
			{onMenu} />
	{/if}
	<div
		class="sticky top-0 z-10 flex flex-wrap items-end gap-x-3 gap-y-2 border-b border-line-subtle bg-canvas p-3"
		role="group"
		aria-label="Server filters">
		<div class="flex flex-col gap-1 text-2xs text-fg-muted">
			<span>Sort</span>
			<Select
				label="Sort servers"
				value={servers.order}
				options={[
					{
						value: GameServerOrder.ServerOrderRecommended,
						label: "Recommended",
					},
					{ value: GameServerOrder.ServerOrderBestPing, label: "Best ping" },
					{
						value: GameServerOrder.ServerOrderMostPlayers,
						label: "Most players",
					},
					{
						value: GameServerOrder.ServerOrderFewest,
						label: "Fewest players",
					},
				]}
				onChange={(order) => servers.setOptions({ order })} />
		</div>
		{#if records || regions.length}
			<div class="flex flex-col gap-1 text-2xs text-fg-muted">
				<span>Region</span>
				<Select
					className="w-40"
					label="Server region"
					value={servers.region}
					options={regionOptions}
					onChange={(region) => (servers.region = region)} />
			</div>
		{/if}
		<div class="flex flex-col gap-1 text-2xs text-fg-muted">
			<span>Ping</span>
			<Select
				className="w-32"
				label="Maximum ping"
				value={servers.maxPing}
				options={[
					{ value: 0, label: "Any ping" },
					{ value: 100, label: "Up to 100 ms" },
					{ value: 200, label: "Up to 200 ms" },
					{ value: 300, label: "Up to 300 ms" },
				]}
				onChange={(value) => (servers.maxPing = value)} />
		</div>
		<div class="flex flex-col gap-1 text-2xs text-fg-muted">
			<span>FPS</span>
			<Select
				className="w-32"
				label="Minimum FPS"
				value={servers.minFps}
				options={[
					{ value: 0, label: "Any FPS" },
					{ value: 25, label: "25 or more" },
					{ value: 45, label: "45 or more" },
					{ value: 55, label: "55 or more" },
				]}
				onChange={(value) => (servers.minFps = value)} />
		</div>
		<div class="flex flex-col gap-1 text-2xs text-fg-muted">
			<span>Per page</span>
			<Select
				className="w-20"
				label="Servers per page"
				value={servers.limit}
				options={[10, 25, 50, 100].map((value) => ({
					value: value as ServerPageSize,
					label: String(value),
				}))}
				onChange={(value) => servers.setOptions({ limit: value })} />
		</div>
		<div class="flex flex-col gap-1 text-2xs text-fg-muted">
			<span id={`${filterID}-full`}>Hide full servers</span>
			<label class="switch">
				<input
					type="checkbox"
					role="switch"
					aria-labelledby={`${filterID}-full`}
					checked={servers.excludeFull}
					onchange={(event) =>
						servers.setOptions({
							excludeFull: event.currentTarget.checked,
						})} />
				<span></span>
			</label>
		</div>
	</div>

	{#if servers.error}
		<div class="empty-state flex-1">
			<Icon svg={warningIcon} size={28} aria-hidden="true" />
			<h2>Servers unavailable</h2>
			<button class="button" type="button" onclick={() => void servers.load()}
				>Try again</button>
		</div>
	{:else if !servers.page}
		<ServerTileSkeleton label="Loading servers…" />
	{:else if !loaded.length}
		<div class="empty-state flex-1" role="status">
			<Icon svg={hardDrivesIcon} size={28} aria-hidden="true" />
			{#if servers.excludeFull}
				<h2>No servers have open slots</h2>
				<p>Every running server is full. Show full servers to see them.</p>
			{:else}
				<h2>No servers are running</h2>
				<p>Nobody is playing this place.</p>
			{/if}
		</div>
	{:else}
		<p
			class="flex flex-wrap gap-x-1 px-3 pt-2 font-mono text-2xs text-fg-faint"
			role="status">
			<span
				>{shown.length === loaded.length
					? numbers.format(loaded.length)
					: `${numbers.format(shown.length)} of ${numbers.format(loaded.length)}`}{servers.hasNext
					? "+"
					: ""}
				{loaded.length === 1 ? "server" : "servers"}</span>
			<span aria-hidden="true">·</span>
			<span
				>{numbers.format(players)} {players === 1 ? "player" : "players"}</span>
			<span aria-hidden="true">·</span>
			<span
				>{numbers.format(openSlots)} open {openSlots === 1
					? "slot"
					: "slots"}</span>
			{#if shown.length}
				<span aria-hidden="true">·</span>
				<span>{decimals.format(averageFps)} FPS on average</span>
			{/if}
			{#if regions.length}
				<span aria-hidden="true">·</span>
				<span
					>{regions.length}
					{regions.length === 1 ? "region" : "regions"}</span>
			{/if}
			{#if outdatedCount}
				<span aria-hidden="true">·</span>
				<span>{outdatedCount} on an older version</span>
			{/if}
		</p>
		{#if shown.length}
			<h2 class="sr-only">Servers</h2>
			<ol
				aria-label="Servers"
				aria-busy={servers.loading}
				data-loading={servers.loading || undefined}>
				{#each shown as server (server.serverId)}
					<ServerTile
						serverId={server.serverId}
						{server}
						record={server.record}
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
		{:else}
			<div class="empty-state flex-1" role="status">
				<Icon svg={hardDrivesIcon} size={28} aria-hidden="true" />
				<h2>No loaded servers match</h2>
				<p>No loaded server matches the region, ping, and FPS filters.</p>
				<button class="button" type="button" onclick={clearFilters}
					>Clear filters</button>
			</div>
			{#if servers.hasNext}
				<LoadMore
					pageKey={servers.page?.servers?.length}
					label="Load more servers"
					loadingLabel="Loading more servers…"
					isLoadingMore={servers.loadingMore}
					hasFailed={!!servers.loadMoreError}
					onLoadMore={() => void servers.loadMore()} />
			{/if}
		{/if}
	{/if}
</div>

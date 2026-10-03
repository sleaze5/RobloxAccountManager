<script lang="ts">
	import AlertTriangle from "@lucide/svelte/icons/triangle-alert"
	import LoaderCircle from "@lucide/svelte/icons/loader-circle"
	import Server from "@lucide/svelte/icons/server"
	import { untrack } from "svelte"
	import { GameServerOrder } from "../backend/bridge"
	import ProfileImage from "../shared/ProfileImage.svelte"
	import Select from "../shared/Select.svelte"
	import ServerActionsButton from "./ServerActionsButton.svelte"
	import ServerPagerNav from "./ServerPagerNav.svelte"
	import ServerUptime from "./ServerUptime.svelte"
	import ServerVersion from "./ServerVersion.svelte"
	import {
		recordLocation,
		recordPlace,
		type GameServers,
		type ServerPageSize,
	} from "./game-servers-state.svelte"

	let {
		servers,
		records,
		newestVersion,
		activeJobId,
		menuID,
		onMenu,
	}: {
		servers: GameServers
		// records reports whether the RoValra integration adds region, version, and uptime columns.
		records: boolean
		newestVersion: number
		activeJobId: string | undefined
		menuID: string
		onMenu: (jobId: string, trigger: HTMLElement, toggle: boolean) => void
	} = $props()

	const filterID = $props.id(),
		numbers = new Intl.NumberFormat()
	let list = $state<HTMLElement | undefined>(undefined)
	const loaded = $derived(servers.page?.servers ?? []),
		shown = $derived(servers.servers),
		players = $derived(shown.reduce((total, server) => total + server.playing, 0))

	$effect(() => {
		void servers.page
		untrack(() => list?.scrollTo({ top: 0 }))
	})
</script>

<div class="game-servers-body">
	<div class="game-servers-filters" role="group" aria-label="Server filters">
		<div class="game-servers-filter">
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
		<div class="game-servers-filter">
			<span>Ping</span>
			<Select
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
		<div class="game-servers-filter compact">
			<span>Per page</span>
			<Select
				label="Servers per page"
				value={servers.limit}
				options={[10, 25, 50, 100].map((value) => ({
					value: value as ServerPageSize,
					label: String(value),
				}))}
				onChange={(value) => servers.setOptions({ limit: value })} />
		</div>
		<div class="game-servers-filter">
			<span id={`${filterID}-full`}>Hide full servers</span>
			<label class="settings-toggle">
				<input
					type="checkbox"
					aria-labelledby={`${filterID}-full`}
					checked={servers.excludeFull}
					onchange={(event) =>
						servers.setOptions({
							excludeFull: event.currentTarget.checked,
						})} />
				<span></span>
			</label>
		</div>
		<ServerPagerNav pager={servers} />
	</div>

	{#if servers.error}
		<div class="games-empty" role="alert">
			<AlertTriangle size={24} aria-hidden="true" />
			<h2>Servers are unavailable</h2>
			<p>{servers.error}</p>
			<button
				class="control-button games-text-button"
				type="button"
				onclick={() => void servers.load()}>Try again</button>
		</div>
	{:else if !servers.page}
		<div class="games-empty" role="status">
			<LoaderCircle class="spinner" size={24} aria-hidden="true" />
			<h2>Loading servers…</h2>
		</div>
	{:else if !loaded.length}
		<div class="games-empty" role="status">
			<Server size={24} aria-hidden="true" />
			{#if servers.pageNumber > 1}
				<h2>No more servers</h2>
				<p>The servers on this page closed. Go back to the first page.</p>
				<button
					class="control-button games-text-button"
					type="button"
					onclick={() => servers.first()}>First page</button>
			{:else if servers.excludeFull}
				<h2>No servers have open slots</h2>
				<p>Every running server is full. Show full servers to see them.</p>
			{:else}
				<h2>No servers are running</h2>
				<p>Nobody is playing this place right now.</p>
			{/if}
		</div>
	{:else}
		<p class="game-servers-summary" role="status">
			<span>Page {servers.pageNumber}</span>
			<span aria-hidden="true">·</span>
			<span
				>{shown.length === loaded.length
					? numbers.format(loaded.length)
					: `${numbers.format(shown.length)} of ${numbers.format(loaded.length)}`}
				{loaded.length === 1 ? "server" : "servers"}</span>
			<span aria-hidden="true">·</span>
			<span
				>{numbers.format(players)}
				{players === 1 ? "player" : "players"}</span>
		</p>
		{#if shown.length}
			<div
				class="game-servers-table"
				class:records
				class:loading={servers.loading}
				bind:this={list}>
				<div class="game-servers-columns" aria-hidden="true">
					<span class="numeric">Players</span>
					<span>In server</span>
					<span class="numeric">Ping</span>
					<span class="numeric">FPS</span>
					<span class="numeric">Language</span>
					<span class="numeric">Friends</span>
					{#if records}
						<span>Region</span>
						<span class="numeric">Version</span>
						<span class="numeric">Uptime</span>
					{/if}
					<span>Job ID</span>
				</div>
				<ol aria-label="Servers" aria-busy={servers.loading}>
					{#each shown as server (server.jobId)}
						{@const full = server.playing >= server.maxPlayers}
						{@const fill = server.maxPlayers
							? Math.min(server.playing / server.maxPlayers, 1)
							: 0}
						{@const images = server.playerImages ?? []}
						{@const hidden = server.playing - images.length}
						<li
							class="game-server-row"
							class:menu-open={activeJobId === server.jobId}
							oncontextmenu={(event) => {
								const trigger =
									event.currentTarget.querySelector<HTMLElement>(
										".game-server-actions",
									)
								if (!trigger) return
								event.preventDefault()
								onMenu(server.jobId, trigger, true)
							}}>
							<div
								class="game-server-capacity"
								aria-label={`${server.playing} of ${server.maxPlayers} players${full ? ", full" : ""}`}>
								<span class="game-server-count"
									>{#if full}<span class="game-server-tag">Full</span
										>{/if}<strong>{server.playing}</strong
									>/{server.maxPlayers}</span>
								<span class="game-server-bar" aria-hidden="true"
									><span style:transform={`scaleX(${fill})`}></span
									></span>
							</div>
							<div
								class="game-server-players"
								aria-label={images.length
									? `${images.length} player avatars shown`
									: "Player avatars unavailable"}>
								{#each images as url, index (index)}
									<span class="game-server-avatar"
										><ProfileImage {url} /></span>
								{/each}
								{#if images.length && hidden > 0}
									<span class="game-server-more">+{hidden}</span>
								{:else if !images.length}
									<span class="game-server-faint">—</span>
								{/if}
							</div>
							{#if server.pingMs === null}
								<span
									class="game-server-faint numeric"
									aria-label="Ping unavailable">—</span>
							{:else}
								<span
									class="game-server-number numeric"
									aria-label={`Ping ${server.pingMs} milliseconds`}>
									{server.pingMs} ms
								</span>
							{/if}
							<span
								class="game-server-number numeric"
								aria-label={`${Math.round(server.fps)} frames per second`}
								>{Math.round(server.fps)}</span>
							{#if server.languageMatches === null}
								<span
									class="game-server-faint numeric"
									aria-label="Language matches unavailable">—</span>
							{:else}
								{@const speakers = `${server.languageMatches} ${server.languageMatches === 1 ? "player speaks" : "players speak"} your language`}
								<span
									class="game-server-number numeric"
									aria-label={speakers}
									data-tooltip={speakers}
									>{server.languageMatches}</span>
							{/if}
							{#if server.friends === null}
								<span
									class="game-server-faint numeric"
									aria-label="Friends unavailable">—</span>
							{:else}
								<span
									class="game-server-number numeric"
									aria-label={`${server.friends} ${server.friends === 1 ? "friend" : "friends"}`}
									>{server.friends}</span>
							{/if}
							{#if records}
								{#if server.record}
									<span
										class="game-server-place"
										data-tooltip={recordLocation(server.record)}
										>{recordPlace(server.record)}</span>
								{:else}
									<span
										class="game-server-faint"
										data-tooltip="RoValra has not seen this server yet"
										>—</span>
								{/if}
								<ServerVersion
									version={server.record?.placeVersion ?? 0}
									{newestVersion} />
								<ServerUptime
									firstSeenMs={server.record?.firstSeenMs ?? 0} />
							{/if}
							<span class="game-server-job" data-tooltip={server.jobId}
								>{server.jobId}</span>
							<ServerActionsButton
								open={activeJobId === server.jobId}
								{menuID}
								onMenu={(trigger, toggle) =>
									onMenu(server.jobId, trigger, toggle)} />
						</li>
					{/each}
				</ol>
			</div>
		{:else}
			<div class="games-empty" role="status">
				<Server size={24} aria-hidden="true" />
				<h2>No servers on this page match</h2>
				<p>
					No server on this page reports a ping of {servers.maxPing} ms or less.
				</p>
				<button
					class="control-button games-text-button"
					type="button"
					onclick={() => (servers.maxPing = 0)}>Show any ping</button>
			</div>
		{/if}
	{/if}
</div>

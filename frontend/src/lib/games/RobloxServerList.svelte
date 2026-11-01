<script lang="ts">
	import circleNotchIcon from "@phosphor-icons/core/regular/circle-notch.svg?raw"
	import hardDrivesIcon from "@phosphor-icons/core/regular/hard-drives.svg?raw"
	import warningIcon from "@phosphor-icons/core/regular/warning.svg?raw"
	import Icon from "../shared/Icon.svelte"
	import { untrack } from "svelte"
	import { GameServerOrder } from "../backend/bridge"
	import ProfileImage from "../shared/ProfileImage.svelte"
	import Select from "../shared/Select.svelte"
	import ServerActionsButton from "./ServerActionsButton.svelte"
	import ServerPagerNav from "./ServerPagerNav.svelte"
	import ServerUptime from "./ServerUptime.svelte"
	import ServerVersion from "./ServerVersion.svelte"
	import {
		type GameServers,
		recordLocation,
		recordPlace,
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

<div class="flex min-h-0 flex-1 flex-col">
	<div
		class="flex flex-wrap items-end gap-x-3 gap-y-2 border-b border-line-subtle p-3"
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
		<div class="flex flex-col gap-1 text-2xs text-fg-muted">
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
		<ServerPagerNav pager={servers} />
	</div>

	{#if servers.error}
		<div class="empty-state" role="alert">
			<Icon svg={warningIcon} size={28} aria-hidden="true" />
			<h2>Servers are unavailable</h2>
			<p>{servers.error}</p>
			<button class="btn" type="button" onclick={() => void servers.load()}
				>Try again</button>
		</div>
	{:else if !servers.page}
		<div class="empty-state" role="status">
			<Icon svg={circleNotchIcon} class="spinner" size={28} aria-hidden="true" />
			<h2>Loading servers…</h2>
		</div>
	{:else if !loaded.length}
		<div class="empty-state" role="status">
			<Icon svg={hardDrivesIcon} size={28} aria-hidden="true" />
			{#if servers.pageNumber > 1}
				<h2>No more servers</h2>
				<p>The servers on this page closed. Go back to the first page.</p>
				<button class="btn" type="button" onclick={() => servers.first()}
					>First page</button>
			{:else if servers.excludeFull}
				<h2>No servers have open slots</h2>
				<p>Every running server is full. Show full servers to see them.</p>
			{:else}
				<h2>No servers are running</h2>
				<p>Nobody is playing this place right now.</p>
			{/if}
		</div>
	{:else}
		<p class="flex gap-1 px-3 pt-2 font-mono text-2xs text-fg-faint" role="status">
			<span>Page {servers.pageNumber}</span>
			<span aria-hidden="true">-</span>
			<span
				>{shown.length === loaded.length
					? numbers.format(loaded.length)
					: `${numbers.format(shown.length)} of ${numbers.format(loaded.length)}`}
				{loaded.length === 1 ? "server" : "servers"}</span>
			<span aria-hidden="true">-</span>
			<span
				>{numbers.format(players)}
				{players === 1 ? "player" : "players"}</span>
		</p>
		{#if shown.length}
			<div
				class="server-table"
				data-variant={records ? "records" : undefined}
				data-loading={servers.loading || undefined}
				bind:this={list}>
				<div class="server-columns" aria-hidden="true">
					<span class="justify-self-end text-right">Players</span>
					<span>In server</span>
					<span class="justify-self-end text-right">Ping</span>
					<span class="justify-self-end text-right">FPS</span>
					<span class="justify-self-end text-right">Language</span>
					<span class="justify-self-end text-right">Friends</span>
					{#if records}
						<span>Region</span>
						<span class="justify-self-end text-right">Version</span>
						<span class="justify-self-end text-right">Uptime</span>
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
							class="server-row"
							data-open={activeJobId === server.jobId || undefined}
							oncontextmenu={(event) => {
								const trigger =
									event.currentTarget.querySelector<HTMLElement>(
										"[data-server-actions]",
									)
								if (!trigger) return
								event.preventDefault()
								onMenu(server.jobId, trigger, true)
							}}>
							<div
								class="flex flex-col gap-1"
								aria-label={`${server.playing} of ${server.maxPlayers} players${full ? ", full" : ""}`}>
								<span
									class="flex items-center justify-end gap-0.5 font-mono text-2xs text-fg-muted"
									>{#if full}<span class="mr-1 server-tag">Full</span
										>{/if}<strong class="font-semibold text-fg"
										>{server.playing}</strong
									>/{server.maxPlayers}</span>
								<span
									class="block h-1 overflow-hidden rounded-full bg-line-strong"
									aria-hidden="true"
									><span
										class="block h-full origin-left bg-fg-muted transition-transform duration-150"
										style:transform={`scaleX(${fill})`}></span
									></span>
							</div>
							<div
								class="flex min-w-0 items-center"
								aria-label={images.length
									? `${images.length} player avatars shown`
									: "Player avatars unavailable"}>
								{#each images as url, index (index)}
									<span
										class="grid size-6 shrink-0 place-items-center overflow-hidden rounded-full border border-canvas bg-raised text-fg-faint not-first:-ml-1.5"
										><ProfileImage {url} /></span>
								{/each}
								{#if images.length && hidden > 0}
									<span class="ml-1 font-mono text-2xs text-fg-muted"
										>+{hidden}</span>
								{:else if !images.length}
									<span class="text-fg-faint">-</span>
								{/if}
							</div>
							{#if server.pingMs === null}
								<span
									class="justify-self-end text-right text-fg-faint"
									aria-label="Ping unavailable">-</span>
							{:else}
								<span
									class="server-number justify-self-end text-right"
									aria-label={`Ping ${server.pingMs} milliseconds`}>
									{server.pingMs} ms
								</span>
							{/if}
							<span
								class="server-number justify-self-end text-right"
								aria-label={`${Math.round(server.fps)} frames per second`}
								>{Math.round(server.fps)}</span>
							{#if server.languageMatches === null}
								<span
									class="justify-self-end text-right text-fg-faint"
									aria-label="Language matches unavailable">-</span>
							{:else}
								{@const speakers = `${server.languageMatches} ${server.languageMatches === 1 ? "player speaks" : "players speak"} your language`}
								<span
									class="server-number justify-self-end text-right"
									aria-label={speakers}
									data-tooltip={speakers}
									>{server.languageMatches}</span>
							{/if}
							{#if server.friends === null}
								<span
									class="justify-self-end text-right text-fg-faint"
									aria-label="Friends unavailable">-</span>
							{:else}
								<span
									class="server-number justify-self-end text-right"
									aria-label={`${server.friends} ${server.friends === 1 ? "friend" : "friends"}`}
									>{server.friends}</span>
							{/if}
							{#if records}
								{#if server.record}
									<span
										class="truncate text-xs text-fg"
										data-tooltip={recordLocation(server.record)}
										>{recordPlace(server.record)}</span>
								{:else}
									<span
										class="text-fg-faint"
										data-tooltip="RoValra has not seen this server yet"
										>-</span>
								{/if}
								<ServerVersion
									version={server.record?.placeVersion ?? 0}
									{newestVersion} />
								<ServerUptime
									firstSeenMs={server.record?.firstSeenMs ?? 0} />
							{/if}
							<span
								class="truncate font-mono text-2xs text-fg-faint select-text"
								data-tooltip={server.jobId}>{server.jobId}</span>
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
			<div class="empty-state" role="status">
				<Icon svg={hardDrivesIcon} size={28} aria-hidden="true" />
				<h2>No servers on this page match</h2>
				<p>
					No server on this page reports a ping of {servers.maxPing} ms or less.
				</p>
				<button class="btn" type="button" onclick={() => (servers.maxPing = 0)}
					>Show any ping</button>
			</div>
		{/if}
	{/if}
</div>

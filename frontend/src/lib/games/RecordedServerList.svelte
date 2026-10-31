<script lang="ts">
	import WarningIcon from "phosphor-svelte/lib/WarningIcon"
	import CircleNotchIcon from "phosphor-svelte/lib/CircleNotchIcon"
	import HardDrivesIcon from "phosphor-svelte/lib/HardDrivesIcon"
	import { untrack } from "svelte"
	import { GameServerRecordOrder } from "../backend/bridge"
	import Select from "../shared/Select.svelte"
	import ServerActionsButton from "./ServerActionsButton.svelte"
	import ServerPagerNav from "./ServerPagerNav.svelte"
	import ServerUptime from "./ServerUptime.svelte"
	import ServerVersion from "./ServerVersion.svelte"
	import {
		type RecordedServers,
		recordLocation,
		type RecordPageSize,
		recordPlace,
		regionLabel,
		type ServerStatsLoader,
	} from "./game-servers-state.svelte"

	let {
		servers,
		stats,
		activeJobId,
		menuID,
		onMenu,
	}: {
		servers: RecordedServers
		stats: ServerStatsLoader
		activeJobId: string | undefined
		menuID: string
		onMenu: (jobId: string, trigger: HTMLElement, toggle: boolean) => void
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
		<ServerPagerNav pager={servers} />
	</div>

	{#if servers.error}
		<div class="empty-state" role="alert">
			<WarningIcon size={28} aria-hidden="true" />
			<h2>RoValra servers are unavailable</h2>
			<p>{servers.error}</p>
			<button class="btn" type="button" onclick={() => void servers.load()}
				>Try again</button>
		</div>
	{:else if !servers.page}
		<div class="empty-state" role="status">
			<CircleNotchIcon class="spinner" size={28} aria-hidden="true" />
			<h2>Loading servers…</h2>
		</div>
	{:else if !shown.length}
		<div class="empty-state" role="status">
			<HardDrivesIcon size={28} aria-hidden="true" />
			{#if servers.pageNumber > 1}
				<h2>No more servers</h2>
				<p>RoValra has no more servers for this list.</p>
				<button class="btn" type="button" onclick={() => servers.first()}
					>First page</button>
			{:else if servers.region}
				<h2>No servers in this region</h2>
				<p>RoValra has not seen a server of this place in this region.</p>
				<button
					class="btn"
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
			<span>Page {servers.pageNumber}</span>
			<span aria-hidden="true">-</span>
			<span
				>{numbers.format(shown.length)}
				{shown.length === 1 ? "server" : "servers"}</span>
			{#if stats.stats?.totalServers}
				<span aria-hidden="true">-</span>
				<span>{numbers.format(stats.stats.totalServers)} recorded</span>
			{/if}
			<span aria-hidden="true">-</span>
			<span>Some may have closed</span>
		</p>
		<div
			class="server-table"
			data-variant="recorded"
			data-loading={servers.loading || undefined}
			bind:this={list}>
			<div class="server-columns" aria-hidden="true">
				<span>Region</span>
				<span class="justify-self-end text-right">Datacenter</span>
				<span>IP address</span>
				<span class="justify-self-end text-right">Version</span>
				<span class="justify-self-end text-right">Uptime</span>
				<span>Job ID</span>
			</div>
			<ol aria-label="Servers" aria-busy={servers.loading}>
				{#each shown as server (server.jobId)}
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
						<span
							class="truncate text-xs text-fg"
							data-tooltip={recordLocation(server) || undefined}
							>{recordPlace(server) || "Unknown location"}</span>
						{#if server.datacenterId}
							<span class="server-number justify-self-end text-right"
								>{server.datacenterId}</span>
						{:else}
							<span
								class="justify-self-end text-right text-fg-faint"
								aria-label="Datacenter unknown">-</span>
						{/if}
						{#if server.address}
							<span
								class="truncate font-mono text-2xs text-fg-faint select-text"
								>{server.address}</span>
						{:else}
							<span class="text-fg-faint" aria-label="IP address unknown"
								>-</span>
						{/if}
						<ServerVersion version={server.placeVersion} {newestVersion} />
						<ServerUptime firstSeenMs={server.firstSeenMs} />
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
	{/if}
</div>

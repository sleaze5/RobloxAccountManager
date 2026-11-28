<script lang="ts">
	import caretDownIcon from "@phosphor-icons/core/regular/caret-down.svg?raw"
	import cellSignalHighIcon from "@phosphor-icons/core/regular/cell-signal-high.svg?raw"
	import cellSignalLowIcon from "@phosphor-icons/core/regular/cell-signal-low.svg?raw"
	import cellSignalMediumIcon from "@phosphor-icons/core/regular/cell-signal-medium.svg?raw"
	import { untrack } from "svelte"
	import { GameServerConnection, JoinOutcome, PingSignal } from "../backend/bridge"
	import type {
		FriendInServer,
		GameServer,
		GameServerRecord,
	} from "../backend/bridge"
	import Icon from "../shared/Icon.svelte"
	import ProfileImage from "../shared/ProfileImage.svelte"
	import Timestamp from "../shared/Timestamp.svelte"
	import ServerActionsButton from "./ServerActionsButton.svelte"
	import ServerFriend from "./ServerFriend.svelte"
	import ServerPlayersSparkline from "./ServerPlayersSparkline.svelte"
	import { openServerMenuFromRow } from "./server-row-menu"
	import type { ServerMenuOpener } from "./server-row-menu"
	import {
		datacenterLocation,
		formatDuration,
		formatUptime,
		outdated,
		recordLocation,
		serverHealth,
		serverPlace,
		serverPlaceVersion,
		type ServerDetailsState,
	} from "./game-servers-state.svelte"

	// ServerTile shows one server with everything that the app knows about it:
	// the Roblox server list, RoValra's record, what Roblox's join service
	// tells the shown account, and what this app observed while listing it.
	// friends are the shown account's friends in the server. menuKey keys the
	// server's menu, and is the server ID unless another list shows the same
	// server.
	let {
		serverId,
		menuKey = serverId,
		headingLevel = 3,
		friends = [],
		server,
		record,
		details: detailsState,
		newestVersion,
		isMenuOpen,
		menuID,
		onMenu,
		onVisible,
	}: {
		serverId: string
		menuKey?: string
		// headingLevel is the level of the tile's title. Its details sections
		// are one level below.
		headingLevel?: 3 | 4
		friends?: FriendInServer[]
		server?: GameServer
		record: GameServerRecord | null
		details: ServerDetailsState | undefined
		newestVersion: number
		isMenuOpen: boolean
		menuID: string
		onMenu: ServerMenuOpener
		onVisible: () => void
	} = $props()

	const tileID = $props.id(),
		numbers = new Intl.NumberFormat(),
		decimals = new Intl.NumberFormat(undefined, {
			minimumFractionDigits: 1,
			maximumFractionDigits: 1,
		}),
		signalIcons: Record<string, string> = {
			[PingSignal.PingSignalLow]: cellSignalHighIcon,
			[PingSignal.PingSignalMedium]: cellSignalMediumIcon,
			[PingSignal.PingSignalHigh]: cellSignalLowIcon,
		}
	let isVisible = $state(false),
		isExpanded = $state(false)

	const details = $derived(
			detailsState?.status === "loaded" ? detailsState.details : null,
		),
		isDetailsLoading = $derived(detailsState?.status === "loading"),
		hasDetailsFailed = $derived(detailsState?.status === "failed"),
		datacenter = $derived(details?.datacenter ?? null),
		datacenterId = $derived(details?.datacenterId || record?.datacenterId || 0),
		place = $derived(serverPlace(record, detailsState)),
		title = $derived(
			place || (datacenterId ? `Datacenter ${datacenterId}` : "Unknown location"),
		),
		titleTooltip = $derived(
			datacenter
				? `${datacenterLocation(datacenter)}. RoValra places datacenter ${datacenterId} here`
				: record && recordLocation(record)
					? `${recordLocation(record)}. Seen there by RoValra`
					: undefined,
		),
		version = $derived(serverPlaceVersion(record, detailsState)),
		isOutdated = $derived(outdated(version, newestVersion)),
		health = $derived(server ? serverHealth(server.fps) : null),
		openSlots = $derived(
			server ? Math.max(server.maxPlayers - server.playing, 0) : 0,
		),
		occupancy = $derived(
			server?.maxPlayers ? Math.min(server.playing / server.maxPlayers, 1) : 0,
		),
		isFull = $derived(!!server && server.playing >= server.maxPlayers),
		images = $derived(server?.playerImages ?? []),
		hiddenPlayers = $derived(server ? server.playing - images.length : 0),
		observation = $derived(server?.observation ?? null),
		history = $derived(observation?.playingHistory ?? []),
		restriction = $derived(
			details?.join.outcome === JoinOutcome.JoinOutcomeRestricted
				? details.join.message
				: "",
		),
		connectionLabel = $derived(
			details?.connection === GameServerConnection.ServerConnectionRelay
				? "UDP relay"
				: details?.connection === GameServerConnection.ServerConnectionDirect
					? "Direct"
					: "",
		),
		metaParts = $derived(
			[
				datacenterId ? `DC ${datacenterId}` : "",
				details?.engineVersion ? `Engine ${details.engineVersion}` : "",
				details?.releaseChannel ? `Channel ${details.releaseChannel}` : "",
				connectionLabel,
			].filter(Boolean),
		)

	$effect(() => {
		if (isVisible && !detailsState) untrack(onVisible)
	})

	function observeVisibility(element: HTMLElement): () => void {
		const observer = new IntersectionObserver(
			(entries) => {
				isVisible = entries.some((entry) => entry.isIntersecting)
			},
			{ rootMargin: "160px" },
		)
		observer.observe(element)
		return () => observer.disconnect()
	}

	function signed(change: number): string {
		return change > 0 ? `+${change}` : String(change)
	}

	function formatRange(
		range: {
			minimum: number
			average: number
			maximum: number
			standardDeviation: number
		},
		unit: string,
	): string {
		return `${Math.round(range.minimum)}–${Math.round(range.maximum)}${unit}, average ${decimals.format(range.average)}, ±${decimals.format(range.standardDeviation)}`
	}
</script>

<li
	class="server-tile"
	data-open={isMenuOpen || undefined}
	aria-labelledby={`${tileID}-title`}
	{@attach observeVisibility}
	oncontextmenu={(event) => openServerMenuFromRow(event, menuKey, onMenu)}>
	<div class="flex min-w-0 items-center gap-3">
		<div class="flex min-w-0 flex-1 items-center gap-3">
			<div class="flex min-w-0 shrink-0 items-center gap-2">
				<svelte:element
					this={`h${headingLevel}`}
					id={`${tileID}-title`}
					class={[
						"truncate text-sm font-semibold",
						title === "Unknown location" ? "text-fg-muted" : "text-fg",
					]}
					data-tooltip={titleTooltip}>
					{title}
				</svelte:element>
				<div class="flex min-w-0 shrink items-center gap-1 overflow-hidden">
					{#if isFull}<span class="server-tag">Full</span>{/if}
					{#if isOutdated}
						<span
							class="server-tag"
							data-tone="warning"
							data-tooltip={`The newest running place version is ${newestVersion}`}
							>Outdated</span>
					{/if}
					{#if restriction}
						<span
							class="truncate server-tag"
							data-tone="danger"
							data-tooltip={restriction}>Can't join</span>
					{:else if details && details.queuePosition > 0}
						<span class="server-tag">Queue {details.queuePosition}</span>
					{:else if !server && details?.hasJoinScript}
						<span
							class="server-tag"
							data-tone="success"
							data-tooltip="Roblox's join service still lists this server"
							>Running</span>
					{/if}
				</div>
			</div>
			<p
				class="flex min-w-0 flex-1 items-center font-mono text-2xs text-fg-faint">
				{#if metaParts.length}
					<span class="truncate">{metaParts.join(" · ")}</span>
				{:else if isDetailsLoading}
					<span class="skeleton-line w-40" aria-label="Loading server details"
					></span>
				{:else if hasDetailsFailed}
					<span class="font-sans">Server details unavailable</span>
				{/if}
			</p>
		</div>
		{#if health}
			<span class="server-health" data-tone={health.tone}>{health.label}</span>
		{/if}
		<div class="flex shrink-0 items-center gap-0.5">
			<button
				class="button-ghost"
				type="button"
				aria-label={isExpanded ? "Hide server details" : "Show server details"}
				aria-expanded={isExpanded}
				aria-controls={`${tileID}-details`}
				data-tooltip={isExpanded ? "Hide details" : "Show details"}
				data-tooltip-side="bottom-end"
				onclick={() => (isExpanded = !isExpanded)}>
				<Icon
					svg={caretDownIcon}
					size={16}
					class={[
						"transition-transform duration-150 motion-reduce:transition-none",
						isExpanded && "rotate-180",
					]}
					aria-hidden="true" />
			</button>
			<ServerActionsButton
				open={isMenuOpen}
				{menuID}
				onMenu={(trigger, toggle) => onMenu(menuKey, trigger, toggle)} />
		</div>
	</div>

	{#if friends.length}
		<ul
			class="flex min-w-0 flex-wrap gap-1.5"
			aria-label={`${friends.length} ${friends.length === 1 ? "friend" : "friends"} in this server`}>
			{#each friends as friend (friend.userId)}
				<li class="min-w-0"><ServerFriend {friend} /></li>
			{/each}
		</ul>
	{/if}

	<dl class="server-stats">
		{#if server}
			<div class="col-span-2">
				<dt>Players</dt>
				<dd>
					<div class="flex w-full flex-col gap-1">
						<span
							aria-label={`${server.playing} of ${server.maxPlayers} players, ${openSlots} open`}>
							<strong class="font-semibold text-fg-bright"
								>{server.playing}</strong
							><span class="text-fg-muted">/{server.maxPlayers}</span>
							<span class="ml-1 font-sans text-2xs text-fg-muted"
								>{isFull ? "No open slots" : `${openSlots} open`}</span>
						</span>
						<span
							class="block h-1 overflow-hidden rounded-full bg-line-strong"
							aria-hidden="true"
							><span
								class="block h-full origin-left bg-fg-muted transition-transform duration-150"
								style:transform={`scaleX(${occupancy})`}></span
							></span>
					</div>
				</dd>
			</div>
			<div
				data-tone={server.pingSignal === PingSignal.PingSignalHigh
					? "warning"
					: undefined}>
				<dt>Ping</dt>
				<dd>
					{#if server.pingMs === null}
						<span class="text-fg-muted">Unknown</span>
					{:else}
						<span>{server.pingMs} ms</span>
						{#if signalIcons[server.pingSignal]}
							<span
								class="text-fg-muted"
								aria-label={`Roblox rates this ping ${server.pingSignal.toLowerCase()}`}
								data-tooltip={`Roblox rates this ping ${server.pingSignal.toLowerCase()}`}
								><Icon
									svg={signalIcons[server.pingSignal]}
									size={14}
									aria-hidden="true" /></span>
						{/if}
					{/if}
				</dd>
			</div>
			<div
				data-tone={health && health.tone !== "success"
					? health.tone
					: undefined}>
				<dt>FPS</dt>
				<dd>
					<span>{decimals.format(server.fps)}</span>
					{#if server.fps > 0}
						<span class="text-2xs text-fg-muted"
							>{Math.round(1000 / server.fps)} ms</span>
					{/if}
				</dd>
			</div>
		{/if}
		<div>
			<dt>Uptime</dt>
			<dd>
				{#if details?.startedMs}
					<span
						data-tooltip="Since Roblox started the server"
						aria-label={`Up ${formatUptime(details.startedMs)}`}
						>{formatUptime(details.startedMs)}</span>
				{:else if record?.firstSeenMs}
					<span
						data-tooltip="Estimated from when RoValra first saw the server"
						aria-label={`Up about ${formatUptime(record.firstSeenMs)}`}
						>~{formatUptime(record.firstSeenMs)}</span>
				{:else if isDetailsLoading}
					<span class="skeleton-line w-12" aria-label="Loading uptime"></span>
				{:else}
					<span class="text-fg-muted">Unknown</span>
				{/if}
			</dd>
		</div>
		<div data-tone={isOutdated ? "warning" : undefined}>
			<dt>Version</dt>
			<dd>
				{#if version}
					<span>{version}</span>
				{:else if isDetailsLoading}
					<span class="skeleton-line w-10" aria-label="Loading place version"
					></span>
				{:else}
					<span class="text-fg-muted">Unknown</span>
				{/if}
			</dd>
		</div>
		{#if server}
			<div>
				<dt>Language</dt>
				<dd>
					{#if server.languageMatches === null}
						<span class="text-fg-muted">Unknown</span>
					{:else}
						<span
							aria-label={`${server.languageMatches} ${server.languageMatches === 1 ? "player speaks" : "players speak"} your language`}
							data-tooltip="Players who speak your language">
							{server.languageMatches}<span class="text-fg-muted"
								>/{server.playing}</span>
						</span>
					{/if}
				</dd>
			</div>
			<div>
				<dt>Friends</dt>
				<dd>
					{#if server.friends === null}
						<span class="text-fg-muted">Unknown</span>
					{:else}
						<span>{server.friends}</span>
					{/if}
				</dd>
			</div>
		{:else}
			{#if record}
				<div class="col-span-2">
					<dt>IP address</dt>
					<dd>
						{#if record.address}
							<span class="truncate select-text">{record.address}</span>
						{:else}
							<span class="text-fg-muted">Unknown</span>
						{/if}
					</dd>
				</div>
			{/if}
			<div class="col-span-3">
				<dt>Server ID</dt>
				<dd class="text-2xs text-fg-muted">
					<span class="truncate select-text">{serverId}</span>
				</dd>
			</div>
		{/if}
	</dl>

	{#if server}
		<div
			class="flex min-h-6 min-w-0 items-center gap-3 border-t border-line-subtle pt-2">
			<div
				class="flex shrink-0 items-center"
				aria-label={images.length
					? `${images.length} player avatars shown`
					: "Player avatars unavailable"}>
				{#each images as url, index (index)}
					<span
						class="grid size-5 shrink-0 place-items-center overflow-hidden rounded-full border border-surface bg-raised text-fg-faint not-first:-ml-1.5"
						><ProfileImage {url} /></span>
				{/each}
				{#if images.length && hiddenPlayers > 0}
					<span class="ml-1.5 font-mono text-2xs text-fg-muted"
						>+{numbers.format(hiddenPlayers)}</span>
				{/if}
			</div>
			{#if history.length > 1 && observation}
				<div class="flex min-w-0 items-center gap-2 text-2xs text-fg-muted">
					<ServerPlayersSparkline {history} maxPlayers={server.maxPlayers} />
					<span class="truncate">
						<span class="font-mono text-fg"
							>{signed(observation.playingChange)}</span>
						over {formatDuration(
							observation.lastSeenMs - observation.firstSeenMs,
						)}
					</span>
				</div>
			{/if}
			<span
				class="ml-auto min-w-0 truncate font-mono text-2xs text-fg-faint select-text"
				data-tooltip="Server ID">{serverId}</span>
		</div>
	{/if}

	{#if isExpanded}
		<div
			id={`${tileID}-details`}
			class="flex flex-col gap-3 border-t border-line-subtle pt-3">
			<section class="flex flex-col gap-1.5" aria-labelledby={`${tileID}-roblox`}>
				<svelte:element
					this={`h${headingLevel + 1}`}
					id={`${tileID}-roblox`}
					class="text-2xs font-semibold text-fg-muted">
					From Roblox's join service
				</svelte:element>
				{#if details?.hasJoinScript}
					<dl class="server-stats" data-density="compact">
						<div class="col-span-2">
							<dt>Started</dt>
							<dd data-font="sans">
								{#if details.startedMs}<Timestamp
										value={details.startedMs} />{:else}<span
										class="text-fg-muted">Unknown</span
									>{/if}
							</dd>
						</div>
						<div>
							<dt>Datacenter</dt>
							<dd>{details.datacenterId || "Unknown"}</dd>
						</div>
						<div class="col-span-2">
							<dt>Engine version</dt>
							<dd>{details.engineVersion || "Unknown"}</dd>
						</div>
						<div>
							<dt>Channel</dt>
							<dd data-font="sans">
								{details.releaseChannel || "Production"}
							</dd>
						</div>
						<div>
							<dt>Locale</dt>
							<dd>{details.locale || "Unknown"}</dd>
						</div>
						<div>
							<dt>Connection</dt>
							<dd data-font="sans">{connectionLabel || "Unknown"}</dd>
						</div>
						<div class="col-span-2">
							<dt>Address</dt>
							<dd>{details.address || "Unknown"}</dd>
						</div>
						<div>
							<dt>Ping interval</dt>
							<dd>
								{details.pingIntervalSeconds
									? `${details.pingIntervalSeconds} s`
									: "Unknown"}
							</dd>
						</div>
					</dl>
				{:else if details}
					<dl class="server-stats" data-density="compact">
						<div class="col-span-4">
							<dt>Join check</dt>
							<dd data-font="sans">
								<span class="whitespace-normal">
									{restriction ||
										(details.join.outcome ===
										JoinOutcome.JoinOutcomeWaiting
											? "Full or starting. A join may work in a moment"
											: "Roblox did not say")}
								</span>
							</dd>
						</div>
						<div>
							<dt>Queue position</dt>
							<dd>{details.queuePosition || "None"}</dd>
						</div>
					</dl>
				{:else if isDetailsLoading}
					<span class="text-xs"
						><span
							class="skeleton-line w-40"
							aria-label="Loading server details"></span
						></span>
				{:else}
					<p class="text-xs text-fg-muted">Server details unavailable</p>
				{/if}
			</section>
			<section
				class="flex flex-col gap-1.5"
				aria-labelledby={`${tileID}-rovalra`}>
				<svelte:element
					this={`h${headingLevel + 1}`}
					id={`${tileID}-rovalra`}
					class="text-2xs font-semibold text-fg-muted">
					From RoValra
				</svelte:element>
				{#if record}
					<dl class="server-stats" data-density="compact">
						<div class="col-span-2">
							<dt>Location</dt>
							<dd data-font="sans">
								<span class="truncate"
									>{recordLocation(record) || "Unknown"}</span>
							</dd>
						</div>
						<div class="col-span-2">
							<dt>First seen</dt>
							<dd data-font="sans">
								{#if record.firstSeenMs}<Timestamp
										value={record.firstSeenMs} />{:else}<span
										class="text-fg-muted">Unknown</span
									>{/if}
							</dd>
						</div>
						<div>
							<dt>Datacenter</dt>
							<dd>{record.datacenterId || "Unknown"}</dd>
						</div>
						<div class="col-span-2">
							<dt>IP address</dt>
							<dd>{record.address || "Unknown"}</dd>
						</div>
						<div>
							<dt>Place version</dt>
							<dd>{record.placeVersion || "Unknown"}</dd>
						</div>
					</dl>
				{:else}
					<p class="text-xs text-fg-muted">
						{datacenter
							? `RoValra places datacenter ${datacenterId} in ${datacenterLocation(datacenter)}, but has not recorded this server`
							: "RoValra has not recorded this server"}
					</p>
				{/if}
			</section>
			{#if server}
				<section
					class="flex flex-col gap-1.5"
					aria-labelledby={`${tileID}-observed`}>
					<svelte:element
						this={`h${headingLevel + 1}`}
						id={`${tileID}-observed`}
						class="text-2xs font-semibold text-fg-muted">
						Observed by this app and calculated
					</svelte:element>
					<dl class="server-stats" data-density="compact">
						<div>
							<dt>Occupancy</dt>
							<dd>{Math.round(occupancy * 100)}%</dd>
						</div>
						<div>
							<dt>Frame time</dt>
							<dd>
								{server.fps > 0
									? `${decimals.format(1000 / server.fps)} ms`
									: "Unknown"}
							</dd>
						</div>
						{#if observation}
							<div class="col-span-2">
								<dt>Listed</dt>
								<dd data-font="sans">
									{observation.listingCount === 1
										? "Once"
										: `${observation.listingCount} times over ${formatDuration(observation.lastSeenMs - observation.firstSeenMs)}`}
								</dd>
							</div>
							<div class="col-span-2">
								<dt>First listed</dt>
								<dd data-font="sans">
									<Timestamp value={observation.firstSeenMs} />
								</dd>
							</div>
							{#if observation.listingCount > 1}
								<div>
									<dt>Net change</dt>
									<dd>{signed(observation.playingChange)}</dd>
								</div>
								<div class="col-span-3">
									<dt>FPS range</dt>
									<dd>{formatRange(observation.fps, "")}</dd>
								</div>
								{#if observation.pingMs}
									<div class="col-span-3">
										<dt>Reported ping range</dt>
										<dd>
											{formatRange(observation.pingMs, " ms")}
										</dd>
									</div>
								{/if}
							{/if}
						{/if}
					</dl>
				</section>
			{/if}
		</div>
	{/if}
</li>

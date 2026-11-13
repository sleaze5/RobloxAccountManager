<script lang="ts">
	import checkIcon from "@phosphor-icons/core/regular/check.svg?raw"
	import circleNotchIcon from "@phosphor-icons/core/regular/circle-notch.svg?raw"
	import minusIcon from "@phosphor-icons/core/regular/minus.svg?raw"
	import skipForwardIcon from "@phosphor-icons/core/regular/skip-forward.svg?raw"
	import xIcon from "@phosphor-icons/core/regular/x.svg?raw"
	import Icon from "../shared/Icon.svelte"
	import { appSettings } from "../settings/settings-store.svelte"
	import type {
		NearestServerSearch,
		RegionCheck,
	} from "./nearest-server-search.svelte"

	type LabelAlignment = "start" | "center" | "end"

	let { search }: { search: NearestServerSearch } = $props()

	// A region lists at most this many servers at a time, so the placeholders
	// take the place of a full batch while it is listed.
	const placeholderServerIndexes = Array.from(
			{ length: 5 },
			(_unused, index) => index,
		),
		numberFormat = new Intl.NumberFormat()

	const previousCheck = $derived(
			search.activeRegionIndex > 0
				? search.regionChecks.at(search.activeRegionIndex - 1)
				: undefined,
		),
		currentCheck = $derived(search.regionChecks.at(search.activeRegionIndex)),
		nextCheck = $derived(search.regionChecks.at(search.activeRegionIndex + 1)),
		hiddenAfterCount = $derived(
			Math.max(search.regionChecks.length - search.activeRegionIndex - 2, 0),
		),
		foundServerNumber = $derived(
			(currentCheck?.servers.findIndex((server) => server.status === "found") ??
				-1) + 1,
		)

	function distanceLabel(check: RegionCheck): string {
		return check.nearbyRegion.region.code === appSettings.roValraRegion
			? "Preferred"
			: `${numberFormat.format(Math.round(check.nearbyRegion.distanceKilometers))} km`
	}
</script>

{#snippet regionNode(check: RegionCheck, isCurrent: boolean, alignment: LabelAlignment)}
	<span class="relative grid shrink-0 animate-enter place-items-center">
		<span
			class={[
				"grid size-5 place-items-center rounded-full border bg-overlay transition-colors duration-100",
				check.status === "waiting" && "border-line-strong",
				check.status === "checking" && "border-line-focus text-fg",
				check.status === "found" && "border-line-focus text-fg",
				(check.status === "empty" || check.status === "skipped") &&
					"border-line-strong text-fg-muted",
				check.status === "failed" &&
					"border-danger/40 bg-danger/10 text-danger-muted",
			]}>
			{#key check.status}
				{#if check.status === "checking" && check.servers.length === 0}
					<Icon svg={circleNotchIcon} class="spinner" size={12} />
				{:else if check.status === "checking" || check.status === "found"}
					<span class="size-1.5 animate-pop rounded-full bg-fg"></span>
				{:else if check.status === "empty"}
					<Icon svg={minusIcon} class="animate-pop" size={12} />
				{:else if check.status === "skipped"}
					<Icon svg={skipForwardIcon} class="animate-pop" size={12} />
				{:else if check.status === "failed"}
					<Icon svg={xIcon} class="animate-pop" size={12} />
				{/if}
			{/key}
		</span>
		<span
			class={[
				"absolute top-full mt-1.5 max-w-24 truncate text-2xs font-medium whitespace-nowrap",
				alignment === "start" && "left-0",
				alignment === "center" && "left-1/2 -translate-x-1/2",
				alignment === "end" && "right-0",
				isCurrent ? "text-fg" : "text-fg-muted",
			]}>{check.nearbyRegion.region.name}</span>
	</span>
{/snippet}

{#if search.phase === "loading"}
	<div role="status" aria-busy="true">
		<span class="sr-only">Loading regions</span>
		<div class="flex items-center gap-1.5 pb-6" aria-hidden="true">
			<span class="size-5 shrink-0 skeleton rounded-full"></span>
			<span class="h-px flex-1 skeleton"></span>
			<span class="size-5 shrink-0 skeleton rounded-full"></span>
		</div>
		<div class="mt-3 text-sm" aria-hidden="true">
			<span class="skeleton-line w-24"></span>
		</div>
		<div class="mt-1 text-xs" aria-hidden="true">
			<span class="skeleton-line w-36"></span>
		</div>
	</div>
{:else if currentCheck}
	<!-- The lines below the track state the region and its progress, so the
	     track itself is only visual. -->
	<div class="flex items-center gap-1.5 pb-6" aria-hidden="true">
		{#if previousCheck}
			{@render regionNode(previousCheck, false, "start")}
			{#key currentCheck.nearbyRegion.region.code}
				<span
					class="min-w-4 flex-1 origin-left scale-x-100 border-t border-fg-faint transition-transform duration-150 ease-out starting:scale-x-0"
				></span>
			{/key}
		{/if}
		{@render regionNode(currentCheck, true, previousCheck ? "center" : "start")}
		<span class="w-4 shrink-0 border-t border-fg-faint"></span>
		<span class="relative flex shrink-0 items-center gap-1.5">
			{#if currentCheck.servers.length === 0}
				{#each placeholderServerIndexes as placeholderIndex (placeholderIndex)}
					{#if placeholderIndex > 0}
						<span class="w-3 border-t border-dashed border-line-strong"
						></span>
					{/if}
					<span
						class="size-4 rounded-full border border-dashed border-line-strong"
					></span>
				{/each}
			{:else}
				{#each currentCheck.servers as server, serverIndex (server.jobId)}
					{#if serverIndex > 0}
						<span class="w-3 border-t border-dashed border-line-focus"
						></span>
					{/if}
					<span
						class={[
							"grid size-4 animate-enter place-items-center rounded-full border transition-colors duration-100",
							server.status === "checking" && "border-line-focus text-fg",
							server.status === "open" &&
								"border-line-focus text-fg-muted",
							(server.status === "closed" ||
								server.status === "skipped") &&
								"border-line-strong text-fg-muted",
							server.status === "found" &&
								"border-success/40 bg-success/10 text-success",
							server.status === "failed" &&
								"border-danger/40 bg-danger/10 text-danger-muted",
						]}>
						{#key server.status}
							{#if server.status === "checking"}
								<Icon svg={circleNotchIcon} class="spinner" size={12} />
							{:else if server.status === "open"}
								<Icon svg={checkIcon} class="animate-pop" size={12} />
							{:else if server.status === "found"}
								<Icon svg={checkIcon} class="animate-bloom" size={12} />
							{:else if server.status === "closed"}
								<Icon svg={minusIcon} class="animate-pop" size={12} />
							{:else if server.status === "skipped"}
								<Icon
									svg={skipForwardIcon}
									class="animate-pop"
									size={12} />
							{:else}
								<Icon svg={xIcon} class="animate-pop" size={12} />
							{/if}
						{/key}
					</span>
				{/each}
			{/if}
			{#if currentCheck.servers.length !== 1}
				<span
					class="absolute top-full left-1/2 mt-1.5 -translate-x-1/2 text-2xs whitespace-nowrap text-fg-faint"
					>Servers</span>
			{/if}
		</span>
		{#if nextCheck}
			<span class="min-w-4 flex-1 border-t border-dotted border-line-focus"
			></span>
			{@render regionNode(
				nextCheck,
				false,
				hiddenAfterCount > 0 ? "center" : "end",
			)}
		{:else}
			<span class="flex-1"></span>
		{/if}
		{#if hiddenAfterCount > 0}
			<span class="shrink-0 pl-1 font-mono text-2xs text-fg-faint"
				>+{hiddenAfterCount}</span>
		{/if}
	</div>

	<div class="mt-3 flex items-baseline justify-between gap-2">
		<span class="truncate text-sm font-semibold text-fg"
			>{currentCheck.nearbyRegion.region.name}</span>
		<span class="shrink-0 font-mono text-2xs text-fg-muted tabular-nums"
			>{distanceLabel(currentCheck)} · region {search.activeRegionIndex + 1} of {search
				.regionChecks.length}</span>
	</div>
	{#if search.phase === "found"}
		<div
			class="mt-2 flex animate-enter items-center gap-2 rounded-lg bg-success/5 px-3 py-2"
			role="status">
			<span class="text-xs font-medium text-success">Found</span>
			<span class="truncate font-mono text-2xs text-fg-muted"
				>{search.foundJobId}</span>
			<span class="ml-auto shrink-0 text-2xs text-fg-faint"
				>{search.accountId === 0
					? "Not checked"
					: `Server ${foundServerNumber} of ${currentCheck.servers.length}`}</span>
		</div>
	{:else if search.phase === "checking"}
		<div class="mt-1 flex items-center gap-1.5 text-xs text-fg-muted">
			<Icon svg={circleNotchIcon} class="spinner" size={12} />
			{currentCheck.servers.length === 0
				? "Finding servers…"
				: `Checking ${currentCheck.servers.length} ${currentCheck.servers.length === 1 ? "server" : "servers"}…`}
		</div>
		<p class="mt-2 h-4 text-2xs leading-4 text-fg-faint">
			{#if nextCheck}Next: {nextCheck.nearbyRegion.region.name}, {distanceLabel(
					nextCheck,
				)}{/if}
		</p>
	{/if}
{/if}

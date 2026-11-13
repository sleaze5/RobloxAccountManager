<script lang="ts">
	import caretRightIcon from "@phosphor-icons/core/regular/caret-right.svg?raw"
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

	type ConnectorKind = "travelled" | "found" | "next" | "upcoming"

	let { search }: { search: NearestServerSearch } = $props()

	const maxVisibleRegionCount = 4,
		numberFormat = new Intl.NumberFormat(),
		regionColumnsByCount = [
			"grid-cols-1",
			"grid-cols-1",
			"grid-cols-2",
			"grid-cols-3",
			"grid-cols-4",
		],
		// Each region spans two connector columns, so a connector from the middle
		// of one region to the middle of the next spans the two columns between
		// them.
		connectorColumnsByRegionCount = [
			"grid-cols-2",
			"grid-cols-2",
			"grid-cols-4",
			"grid-cols-6",
			"grid-cols-8",
		],
		connectorStartColumns = ["col-start-2", "col-start-4", "col-start-6"]

	const firstVisibleIndex = $derived(
			Math.min(
				Math.max(search.activeRegionIndex - 1, 0),
				Math.max(search.regionChecks.length - maxVisibleRegionCount, 0),
			),
		),
		visibleChecks = $derived(
			search.regionChecks.slice(
				firstVisibleIndex,
				firstVisibleIndex + maxVisibleRegionCount,
			),
		),
		hiddenAfterCount = $derived(
			search.regionChecks.length - firstVisibleIndex - visibleChecks.length,
		),
		connectors = $derived(
			visibleChecks.slice(1).map((destination, offset) => {
				const source = visibleChecks[offset],
					kind = connectorKind(source, destination)
				return {
					key: `${source.nearbyRegion.region.code}>${destination.nearbyRegion.region.code}:${kind}`,
					kind,
					startColumn: connectorStartColumns[offset],
				}
			}),
		)

	function connectorKind(
		source: RegionCheck,
		destination: RegionCheck,
	): ConnectorKind {
		if (destination.status === "found") return "found"
		if (destination.status !== "waiting") return "travelled"
		return source.status === "checking" ? "next" : "upcoming"
	}

	function statusLabel(check: RegionCheck, isUpNext: boolean): string {
		switch (check.status) {
			case "checking":
				return "Checking…"
			case "found":
				return "Server found"
			case "empty":
				return "No open server"
			case "skipped":
				return "Skipped"
			case "failed":
				return "Failed"
			case "waiting":
				return isUpNext ? "Up next" : "Waiting"
		}
	}

	function distanceLabel(check: RegionCheck): string {
		return check.nearbyRegion.region.code === appSettings.roValraRegion
			? "Preferred"
			: `${numberFormat.format(Math.round(check.nearbyRegion.distanceKilometers))} km`
	}
</script>

<div class="rounded-lg border border-line px-3 pt-2 pb-3">
	<div class="flex items-center justify-between gap-2 text-xs text-fg-muted">
		<span class="font-medium">Regions by distance</span>
		{#if search.phase !== "loading"}
			<span class="font-mono text-2xs tabular-nums"
				>{search.checkedRegionCount} of {search.regionChecks.length} checked</span>
		{/if}
	</div>

	{#if search.phase === "loading"}
		<div class="mt-3" role="status" aria-busy="true">
			<span class="sr-only">Loading regions</span>
			<div class="grid grid-cols-4" aria-hidden="true">
				{#each ["w-16", "w-12", "w-14", "w-12"] as nameWidth, index (index)}
					<div class="flex min-w-0 flex-col items-center px-1">
						<span class="size-7 skeleton rounded-full"></span>
						<span class="mt-2 text-xs"
							><span class={["skeleton-line", nameWidth]}></span></span>
						<span class="text-2xs"
							><span class="skeleton-line w-10"></span></span>
						<span class="h-4 text-2xs leading-4"
							><span class="skeleton-line w-12"></span></span>
					</div>
				{/each}
			</div>
		</div>
	{:else}
		<div class="relative mt-3">
			<!-- The connectors sit level with the middle of the size-7 markers and
			     stop short of them by the marker radius plus a small gap. -->
			<div
				class={[
					"absolute inset-x-0 top-3.5 grid",
					connectorColumnsByRegionCount[visibleChecks.length],
				]}
				aria-hidden="true">
				{#each connectors as connector (connector.key)}
					<span
						class={[
							"relative col-span-2 mx-5 border-t",
							connector.startColumn,
							connector.kind === "found" && "border-success",
							connector.kind === "travelled" && "border-fg-faint",
							(connector.kind === "travelled" ||
								connector.kind === "found") &&
								"origin-left scale-x-100 transition-transform duration-150 ease-out starting:scale-x-0",
							connector.kind === "next" &&
								"animate-enter border-dashed border-fg-faint",
							connector.kind === "upcoming" &&
								"animate-enter border-dotted border-line-focus",
						]}>
						{#if connector.kind === "next"}
							<Icon
								svg={caretRightIcon}
								class="absolute -top-1.5 -right-1.5 text-fg-faint"
								size={12} />
						{/if}
					</span>
				{/each}
			</div>
			{#if firstVisibleIndex > 0}
				<span
					class="absolute top-1.5 left-0 font-mono text-2xs text-fg-faint"
					data-tooltip={`${firstVisibleIndex} earlier ${firstVisibleIndex === 1 ? "region" : "regions"}`}
					aria-hidden="true">+{firstVisibleIndex}</span>
			{/if}
			{#if hiddenAfterCount > 0}
				<span
					class="absolute top-1.5 right-0 font-mono text-2xs text-fg-faint"
					data-tooltip={`${hiddenAfterCount} more ${hiddenAfterCount === 1 ? "region" : "regions"}`}
					aria-hidden="true">+{hiddenAfterCount}</span>
			{/if}
			<ol
				class={["grid", regionColumnsByCount[visibleChecks.length]]}
				aria-label="Regions by distance">
				{#if firstVisibleIndex > 0}
					<li class="sr-only">{firstVisibleIndex} earlier regions checked</li>
				{/if}
				{#each visibleChecks as check, offset (check.nearbyRegion.region.code)}
					{@const index = firstVisibleIndex + offset}
					{@const isUpNext =
						check.status === "waiting" &&
						search.phase === "checking" &&
						index === search.activeRegionIndex + 1}
					<li
						class="flex min-w-0 animate-enter flex-col items-center px-1 text-center"
						aria-current={index === search.activeRegionIndex &&
						search.phase === "checking"
							? "step"
							: undefined}>
						<span
							class={[
								"grid size-7 shrink-0 place-items-center rounded-full border bg-overlay transition-colors duration-100",
								check.status === "waiting" && "border-line-strong",
								check.status === "checking" &&
									"border-line-focus text-fg",
								check.status === "found" &&
									"border-success/40 bg-success/10 text-success",
								(check.status === "empty" ||
									check.status === "skipped") &&
									"border-line-strong text-fg-muted",
								check.status === "failed" &&
									"border-danger/40 bg-danger/10 text-danger-muted",
							]}
							aria-hidden="true">
							{#key check.status}
								{#if check.status === "checking"}
									<Icon
										svg={circleNotchIcon}
										class="spinner"
										size={14} />
								{:else if check.status === "found"}
									<Icon
										svg={checkIcon}
										class="animate-bloom"
										size={14} />
								{:else if check.status === "empty"}
									<Icon
										svg={minusIcon}
										class="animate-pop"
										size={14} />
								{:else if check.status === "skipped"}
									<Icon
										svg={skipForwardIcon}
										class="animate-pop"
										size={14} />
								{:else if check.status === "failed"}
									<Icon svg={xIcon} class="animate-pop" size={14} />
								{/if}
							{/key}
						</span>
						<span
							class={[
								"mt-2 w-full truncate text-xs font-medium",
								check.status === "waiting"
									? "text-fg-muted"
									: "text-fg",
							]}>{check.nearbyRegion.region.name}</span>
						<span
							class="font-mono text-2xs whitespace-nowrap text-fg-muted tabular-nums"
							>{distanceLabel(check)}</span>
						<!-- A region without an open server shows only its marker, so
						     its status is spoken but not shown. -->
						<span
							class={[
								"h-4 w-full truncate text-2xs leading-4",
								check.status === "found" && "font-medium text-success",
								check.status === "failed" && "text-danger-muted",
								check.status === "waiting" &&
									!isUpNext &&
									"text-fg-faint",
								(check.status === "checking" ||
									check.status === "skipped" ||
									isUpNext) &&
									"text-fg-muted",
							]}
							><span class={[check.status === "empty" && "sr-only"]}
								>{statusLabel(check, isUpNext)}</span
							></span>
					</li>
				{/each}
				{#if hiddenAfterCount > 0}
					<li class="sr-only">{hiddenAfterCount} more regions</li>
				{/if}
			</ol>
		</div>
	{/if}
</div>

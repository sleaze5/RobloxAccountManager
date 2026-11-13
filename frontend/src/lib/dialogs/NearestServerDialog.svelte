<script lang="ts">
	import arrowSquareOutIcon from "@phosphor-icons/core/regular/arrow-square-out.svg?raw"
	import checkCircleIcon from "@phosphor-icons/core/regular/check-circle.svg?raw"
	import checkIcon from "@phosphor-icons/core/regular/check.svg?raw"
	import circleNotchIcon from "@phosphor-icons/core/regular/circle-notch.svg?raw"
	import minusIcon from "@phosphor-icons/core/regular/minus.svg?raw"
	import skipForwardIcon from "@phosphor-icons/core/regular/skip-forward.svg?raw"
	import xIcon from "@phosphor-icons/core/regular/x.svg?raw"
	import { Browser } from "@wailsio/runtime"
	import { onMount, tick } from "svelte"
	import Dialog from "../shared/Dialog.svelte"
	import Icon from "../shared/Icon.svelte"
	import SkeletonList from "../shared/SkeletonList.svelte"
	import SkeletonMediaRow from "../shared/SkeletonMediaRow.svelte"
	import type {
		NearestServerSearch,
		RegionCheck,
	} from "../layout/nearest-server-search.svelte"
	import { appSettings } from "../settings/settings-store.svelte"

	let {
		search,
		onUseServer,
		onClose,
	}: {
		search: NearestServerSearch
		onUseServer: (jobId: string) => void
		onClose: () => void
	} = $props()

	const numberFormat = new Intl.NumberFormat()
	let regionList = $state<HTMLOListElement>(),
		useServerButton = $state<HTMLButtonElement>()

	const isSearching = $derived(
			search.phase === "loading" || search.phase === "checking",
		),
		activeRegion = $derived(
			search.regionChecks[search.activeRegionIndex]?.nearbyRegion.region,
		),
		progressLabel = $derived(
			search.regionChecks.length === 0
				? ""
				: `${search.checkedRegionCount} of ${search.regionChecks.length} checked`,
		),
		announcement = $derived(
			search.phase === "checking" && activeRegion
				? `Checking ${activeRegion.name}`
				: "",
		)

	function statusLabel(check: RegionCheck, index: number): string {
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
				return search.phase === "checking" &&
					index === search.activeRegionIndex + 1
					? "Up next"
					: ""
		}
	}

	function regionDetails(check: RegionCheck): string {
		const { region, servers } = check.nearbyRegion,
			serverCount = `${numberFormat.format(servers)} ${servers === 1 ? "server" : "servers"}`
		return region.country && region.country !== region.name
			? `${region.country} · ${serverCount}`
			: serverCount
	}

	function useServer(): void {
		if (search.phase !== "found") return
		onUseServer(search.foundJobId)
	}

	$effect(() => {
		const index = search.activeRegionIndex
		if (search.phase === "loading" || !regionList) return
		const rows = regionList.querySelectorAll<HTMLElement>("[data-region-row]")
		rows[index + 1]?.scrollIntoView({ block: "nearest" })
		rows[index]?.scrollIntoView({ block: "nearest" })
	})

	$effect(() => {
		if (search.phase !== "found") return
		void tick().then(() => useServerButton?.focus())
	})

	onMount(() => {
		void search.start()
		return () => search.dispose()
	})
</script>

<Dialog
	class="max-w-md"
	tabindex={-1}
	aria-labelledby="nearest-server-title"
	aria-describedby="nearest-server-description"
	onkeydown={(event) => {
		if (event.key === "Escape") {
			event.preventDefault()
			event.stopPropagation()
			onClose()
		}
	}}>
	<div class="dialog-heading">
		<div>
			<h2 id="nearest-server-title">Find nearest server</h2>
			<p id="nearest-server-description">
				Checks regions from nearest to farthest, starting at {appSettings.preferredRegionLabel}.
			</p>
		</div>
		<button type="button" aria-label="Cancel server search" onclick={onClose}>
			<Icon svg={xIcon} size={14} aria-hidden="true" />
		</button>
	</div>
	<p class="-mt-2 mb-3 flex items-center gap-1 text-xs text-fg-muted">
		Powered by
		<button
			class="inline-flex items-center gap-1 rounded-sm font-medium text-fg underline-offset-2 hover:underline"
			type="button"
			aria-label="Powered by RoValra, open RoValra website"
			data-tooltip="Open RoValra website"
			onclick={() => void Browser.OpenURL("https://www.rovalra.com/")}
			>RoValra<Icon
				svg={arrowSquareOutIcon}
				size={12}
				aria-hidden="true" /></button>
	</p>

	{#if search.phase === "loading" || search.regionChecks.length > 0}
		<div class="overflow-hidden rounded-lg border border-line">
			<div
				class="flex h-8 items-center justify-between gap-2 border-b border-line-subtle bg-fill px-3 text-xs text-fg-muted">
				<span class="font-medium">Regions by distance</span>
				<span class="font-mono text-2xs tabular-nums">{progressLabel}</span>
			</div>
			{#if search.phase === "loading"}
				<SkeletonList
					label="Loading regions"
					rowCount={5}
					className="max-h-56 overflow-hidden">
					{#snippet row(index)}
						<SkeletonMediaRow {index} imageClass="size-5 rounded-full" />
					{/snippet}
				</SkeletonList>
			{:else}
				<ol
					bind:this={regionList}
					class="max-h-56 overflow-y-auto overscroll-contain"
					aria-label="Regions by distance">
					{#each search.regionChecks as check, index (check.nearbyRegion.region.code)}
						{@const label = statusLabel(check, index)}
						<li
							class={[
								"flex h-12 items-center gap-3 border-b border-line-subtle px-3 transition-colors duration-100 last:border-b-0",
								index === search.activeRegionIndex &&
									search.phase !== "exhausted" &&
									"bg-active",
							]}
							data-region-row
							aria-current={index === search.activeRegionIndex &&
							isSearching
								? "step"
								: undefined}>
							<span
								class={[
									"grid size-5 shrink-0 place-items-center rounded-full border transition-colors duration-100",
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
											size={12} />
									{:else if check.status === "found"}
										<Icon
											svg={checkIcon}
											class="animate-bloom"
											size={12} />
									{:else if check.status === "empty"}
										<Icon
											svg={minusIcon}
											class="animate-pop"
											size={12} />
									{:else if check.status === "skipped"}
										<Icon
											svg={skipForwardIcon}
											class="animate-pop"
											size={12} />
									{:else if check.status === "failed"}
										<Icon
											svg={xIcon}
											class="animate-pop"
											size={12} />
									{/if}
								{/key}
							</span>
							<span class="flex min-w-0 flex-1 flex-col">
								<span
									class={[
										"truncate text-sm font-medium",
										check.status === "waiting"
											? "text-fg-muted"
											: "text-fg",
									]}>{check.nearbyRegion.region.name}</span>
								<span class="truncate text-2xs text-fg-muted"
									>{regionDetails(check)}</span>
							</span>
							{#if check.nearbyRegion.region.code === appSettings.roValraRegion}
								<span class="server-tag">Preferred</span>
							{:else}
								<span
									class="shrink-0 font-mono text-2xs whitespace-nowrap text-fg-muted tabular-nums"
									>{numberFormat.format(
										Math.round(
											check.nearbyRegion.distanceKilometers,
										),
									)} km</span>
							{/if}
							<span
								class={[
									"w-24 shrink-0 text-right text-xs",
									check.status === "found" &&
										"font-medium text-success",
									check.status === "failed" && "text-danger-muted",
									check.status !== "found" &&
										check.status !== "failed" &&
										"text-fg-muted",
								]}>
								{#if label}{label}{:else}<span class="sr-only"
										>Waiting</span
									>{/if}
							</span>
						</li>
					{/each}
				</ol>
			{/if}
		</div>
	{/if}

	<span class="sr-only" role="status">{announcement}</span>

	{#if search.phase === "found" && activeRegion}
		<div
			class="mt-3 flex animate-enter items-center gap-3 rounded-lg border border-success/30 bg-success/5 px-3 py-2"
			role="status">
			<Icon
				svg={checkCircleIcon}
				class="shrink-0 animate-bloom text-success"
				size={16}
				aria-hidden="true" />
			<span class="flex min-w-0 flex-1 flex-col">
				<span class="text-sm font-semibold text-fg"
					>Server found in {activeRegion.name}</span>
				<span class="truncate font-mono text-2xs text-fg-muted"
					>{search.foundJobId}</span>
			</span>
		</div>
	{:else if search.phase === "exhausted"}
		<div class="mt-3 notice-error" role="alert">
			{search.regionChecks.length === 0
				? "RoValra has no servers of this place right now. Try again later."
				: `No open server was found in ${search.regionChecks.length} ${search.regionChecks.length === 1 ? "region" : "regions"}. Try again later.`}
		</div>
	{:else if search.phase === "failed"}
		<div class="mt-3 notice-error" role="alert">{search.error}</div>
	{/if}

	<div class="dialog-actions">
		{#if search.phase === "exhausted"}
			<button class="button" type="button" onclick={onClose}>Close</button>
			<button
				class="button-primary"
				type="button"
				onclick={() => void search.start()}>Search again</button>
		{:else if search.phase === "failed"}
			<button class="button" type="button" onclick={onClose}>Cancel</button>
			<button
				class="button-primary"
				type="button"
				onclick={() => void search.retry()}>Try again</button>
		{:else}
			<button class="button" type="button" onclick={onClose}>Cancel</button>
			<button
				class="button"
				type="button"
				disabled={search.phase !== "found"}
				onclick={() => void search.skipFoundServer()}>Skip server</button>
			<button
				bind:this={useServerButton}
				class="button-primary"
				type="button"
				disabled={search.phase !== "found"}
				onclick={useServer}>{isSearching ? "Searching…" : "Use server"}</button>
		{/if}
	</div>
</Dialog>

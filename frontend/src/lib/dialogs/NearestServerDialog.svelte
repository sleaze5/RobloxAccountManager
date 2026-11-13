<script lang="ts">
	import arrowSquareOutIcon from "@phosphor-icons/core/regular/arrow-square-out.svg?raw"
	import checkCircleIcon from "@phosphor-icons/core/regular/check-circle.svg?raw"
	import xIcon from "@phosphor-icons/core/regular/x.svg?raw"
	import { Browser } from "@wailsio/runtime"
	import { tick } from "svelte"
	import Dialog from "../shared/Dialog.svelte"
	import Icon from "../shared/Icon.svelte"
	import NearestServerRoute from "../layout/NearestServerRoute.svelte"
	import type { NearestServerSearch } from "../layout/nearest-server-search.svelte"
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

	let useServerButton = $state<HTMLButtonElement>()

	const isSearching = $derived(
			search.phase === "loading" || search.phase === "checking",
		),
		activeRegion = $derived(
			search.regionChecks[search.activeRegionIndex]?.nearbyRegion.region,
		),
		announcement = $derived(
			search.phase === "checking" && activeRegion
				? `Checking ${activeRegion.name}`
				: "",
		)

	function useServer(): void {
		if (search.phase !== "found") return
		onUseServer(search.foundJobId)
	}

	$effect(() => {
		if (search.phase !== "found") return
		void tick().then(() => useServerButton?.focus())
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

	{#if search.phase === "loading" || search.regionChecks.length > 0}
		<NearestServerRoute {search} />
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

	<div class="dialog-actions items-center">
		<p class="mr-auto flex items-center gap-1 text-xs text-fg-muted">
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

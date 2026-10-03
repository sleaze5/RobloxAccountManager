<script lang="ts">
	import ChevronLeft from "@lucide/svelte/icons/chevron-left"
	import LoaderCircle from "@lucide/svelte/icons/loader-circle"
	import RefreshCw from "@lucide/svelte/icons/refresh-cw"
	import { onMount, untrack } from "svelte"
	import { appSettings } from "../settings/settings-store.svelte"
	import GameServerMenu from "./GameServerMenu.svelte"
	import RecordedServerList from "./RecordedServerList.svelte"
	import RobloxServerList from "./RobloxServerList.svelte"
	import {
		GameServers,
		RecordedServers,
		ServerStatsLoader,
		type ServerSource,
	} from "./game-servers-state.svelte"
	import type { GamesStore } from "./games-store.svelte"

	let {
		store,
		onBack,
		onFillLaunch,
	}: {
		store: GamesStore
		onBack: () => void
		onFillLaunch: (placeId: number, jobId: string) => void
	} = $props()

	const sources: { id: ServerSource; label: string; tooltip: string }[] = [
		{
			id: "roblox",
			label: "Roblox",
			tooltip: "Servers running now, with players, ping, and FPS",
		},
		{
			id: "rovalra",
			label: "RoValra",
			tooltip: "Servers seen by RoValra users, by region and age",
		},
	]

	// GamesPage remounts this page when the selected place changes.
	const place = untrack(() => store.selected!),
		robloxServers = untrack(
			() =>
				new GameServers(
					place.placeId,
					store.serverPages,
					() => appSettings.roValraEnabled,
				),
		),
		recordedServers = untrack(
			() => new RecordedServers(place.placeId, store.recordPages),
		),
		stats = untrack(() => new ServerStatsLoader(place.placeId)),
		menuID = $props.id()
	let heading = $state<HTMLHeadingElement | undefined>(undefined),
		activeMenu = $state<{ jobId: string; trigger: HTMLElement } | null>(null)
	const source = $derived<ServerSource>(
			appSettings.roValraEnabled ? store.serverSource : "roblox",
		),
		active = $derived(source === "rovalra" ? recordedServers : robloxServers)

	onMount(() => {
		heading?.focus({ preventScroll: true })
		return () => {
			robloxServers.dispose()
			recordedServers.dispose()
			stats.dispose()
		}
	})

	$effect(() => {
		const current = active
		untrack(() => {
			if (!current.page && !current.loading) void current.load()
		})
	})

	$effect(() => {
		void active.page
		untrack(() => (activeMenu = null))
	})

	// Reload the Roblox list when the integration changes, since its pages carry RoValra records.
	$effect(() => {
		void appSettings.roValraEnabled
		untrack(() => {
			if (robloxServers.page) void robloxServers.load()
		})
	})

	// RoValra's regions and newest version serve both lists while the integration is on.
	$effect(() => {
		if (appSettings.roValraEnabled)
			untrack(() => {
				if (!stats.stats && !stats.loading) void stats.load()
			})
	})

	function openMenu(jobId: string, trigger: HTMLElement, toggle: boolean): void {
		activeMenu =
			toggle && activeMenu?.trigger === trigger ? null : { jobId, trigger }
	}

	function refresh(): void {
		active.refresh()
		if (appSettings.roValraEnabled) void stats.load()
	}
</script>

<section class="settings-page game-servers-page" aria-labelledby="game-servers-title">
	<header class="settings-header">
		<button
			type="button"
			aria-label="Back to game details"
			data-tooltip="Back to game details"
			onclick={onBack}>
			<ChevronLeft size={16} aria-hidden="true" />
		</button>
		<h1 id="game-servers-title" tabindex="-1" bind:this={heading}>Servers</h1>
		<span class="account-settings-identity">{store.displayName(place)}</span>
		{#if appSettings.roValraEnabled}
			<div class="game-servers-source" role="radiogroup" aria-label="Server list">
				{#each sources as option (option.id)}
					<label data-tooltip={option.tooltip} data-tooltip-side="bottom-end">
						<input
							type="radio"
							name={`${menuID}-source`}
							value={option.id}
							checked={source === option.id}
							onchange={() => (store.serverSource = option.id)} />
						<span>{option.label}</span>
					</label>
				{/each}
			</div>
		{/if}
		<button
			type="button"
			aria-label={active.loading ? "Refreshing servers" : "Refresh servers"}
			data-tooltip="Refresh servers"
			data-tooltip-side="bottom-end"
			aria-busy={active.loading}
			disabled={active.loading}
			onclick={refresh}>
			{#if active.loading}
				<LoaderCircle class="spinner" size={15} aria-hidden="true" />
			{:else}
				<RefreshCw size={15} aria-hidden="true" />
			{/if}
		</button>
	</header>

	{#if source === "rovalra"}
		<RecordedServerList
			servers={recordedServers}
			{stats}
			activeJobId={activeMenu?.jobId}
			{menuID}
			onMenu={openMenu} />
	{:else}
		<RobloxServerList
			servers={robloxServers}
			records={appSettings.roValraEnabled}
			newestVersion={stats.stats?.newestVersion ?? 0}
			activeJobId={activeMenu?.jobId}
			{menuID}
			onMenu={openMenu} />
	{/if}
</section>

{#if activeMenu}
	<GameServerMenu
		id={menuID}
		anchor={activeMenu.trigger}
		onClose={() => (activeMenu = null)}
		onFillLaunch={() => {
			if (!activeMenu) return
			const { jobId } = activeMenu
			activeMenu = null
			onFillLaunch(place.placeId, jobId)
		}}
		onCopyJobId={() => {
			if (!activeMenu) return
			const { jobId } = activeMenu
			activeMenu = null
			void store.copy(jobId, "Job ID")
		}} />
{/if}

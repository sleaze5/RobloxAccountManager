<script lang="ts">
	import circleNotchIcon from "@phosphor-icons/core/regular/circle-notch.svg?raw"
	import crosshairIcon from "@phosphor-icons/core/regular/crosshair.svg?raw"
	import { untrack } from "svelte"
	import Icon from "../shared/Icon.svelte"
	import type { AccountStore } from "../accounts/account-store.svelte"
	import NearestServerDialog from "../dialogs/NearestServerDialog.svelte"
	import type { NotificationCenter } from "../notifications/notification-center.svelte"
	import { appSettings } from "../settings/settings-store.svelte"
	import { NearestServerSearch } from "./nearest-server-search.svelte"

	let {
		store,
		notifications,
	}: { store: AccountStore; notifications: NotificationCenter } = $props()

	// Holding the button is an intentionally hidden shortcut. It skips the
	// dialog and fills the nearest server directly, as the button did before
	// the dialog existed. It has no visible hint or tooltip on purpose, so that
	// the button stays simple for everyone else. A click still opens the dialog.
	const holdDelayMs = 500

	// The search stays set after the dialog closes, because the dialog still
	// reads it during its exit transition.
	let search = $state<NearestServerSearch | null>(null),
		isSearchOpen = $state(false),
		backgroundSearch = $state<NearestServerSearch | null>(null)
	let holdTimer = 0,
		wasHeld = false
	const region = $derived(appSettings.roValraRegion),
		placeId = $derived(
			/^\d+$/.test(store.launchInput.placeId.trim())
				? Number(store.launchInput.placeId.trim())
				: 0,
		),
		hasPlace = $derived(Number.isSafeInteger(placeId) && placeId > 0),
		tooltip = $derived(
			!region
				? "Choose a preferred region in Settings first"
				: !hasPlace
					? "Enter a Place ID first"
					: `Find a server in or near your preferred region (${appSettings.preferredRegionLabel}). Can be changed in settings`,
		)

	function openSearch(): void {
		search?.dispose()
		search = new NearestServerSearch(placeId, store.selectedAccountId ?? 0)
		isSearchOpen = true
		void search.start()
	}

	function closeSearch(): void {
		search?.dispose()
		isSearchOpen = false
	}

	function beginHold(event: PointerEvent): void {
		if (event.button !== 0) return
		wasHeld = false
		window.clearTimeout(holdTimer)
		holdTimer = window.setTimeout(() => {
			wasHeld = true
			void fillNearestServer()
		}, holdDelayMs)
	}

	function endHold(): void {
		window.clearTimeout(holdTimer)
	}

	async function fillNearestServer(): Promise<void> {
		const placeInput = store.launchInput.placeId,
			nearestSearch = new NearestServerSearch(
				placeId,
				store.selectedAccountId ?? 0,
			)
		backgroundSearch = nearestSearch
		await nearestSearch.start()
		if (backgroundSearch !== nearestSearch) return
		backgroundSearch = null
		nearestSearch.dispose()
		if (!store.vault.unlocked || store.launchInput.placeId !== placeInput) return
		if (nearestSearch.phase !== "found") {
			notifications.show({
				id: "nearest-server",
				title: "No server found",
				tone: "error",
				message:
					nearestSearch.phase === "failed"
						? nearestSearch.error
						: "No open server of this place was found. Try again later.",
			})
			return
		}
		store.launchInput.jobId = nearestSearch.foundJobId
		const foundRegion =
			nearestSearch.regionChecks[nearestSearch.activeRegionIndex]?.nearbyRegion
				.region
		if (foundRegion && foundRegion.code !== appSettings.roValraRegion) {
			notifications.show({
				id: "nearest-server",
				title: "Nearest server filled",
				message: `No open server was found in ${appSettings.preferredRegionLabel}, so a server in ${foundRegion.name} was filled.`,
			})
		}
	}

	function cancelSearches(): void {
		closeSearch()
		window.clearTimeout(holdTimer)
		backgroundSearch?.dispose()
		backgroundSearch = null
	}

	$effect(() => {
		if (!store.vault.unlocked) untrack(cancelSearches)
	})
</script>

<span class="flex shrink-0" data-tooltip={tooltip} data-tooltip-side="top-end">
	<button
		class="button-icon size-8"
		type="button"
		aria-label="Find nearest server"
		aria-haspopup="dialog"
		aria-busy={backgroundSearch !== null}
		disabled={!region || !hasPlace || store.launching || backgroundSearch !== null}
		onpointerdown={beginHold}
		onpointerup={endHold}
		onpointerleave={endHold}
		onpointercancel={endHold}
		onkeydown={() => (wasHeld = false)}
		onclick={() => {
			if (wasHeld) {
				wasHeld = false
				return
			}
			openSearch()
		}}>
		{#if backgroundSearch}
			<Icon svg={circleNotchIcon} class="spinner" size={16} aria-hidden="true" />
		{:else}
			<Icon svg={crosshairIcon} size={16} aria-hidden="true" />
		{/if}
	</button>
</span>

{#if isSearchOpen && search}
	{#key search}
		<NearestServerDialog
			{search}
			onUseServer={(jobId) => {
				store.launchInput.jobId = jobId
				closeSearch()
			}}
			onClose={closeSearch} />
	{/key}
{/if}

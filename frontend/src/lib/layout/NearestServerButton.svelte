<script lang="ts">
	import LoaderCircle from "@lucide/svelte/icons/loader-circle"
	import LocateFixed from "@lucide/svelte/icons/locate-fixed"
	import type { AccountStore } from "../accounts/account-store.svelte"
	import { appSettings } from "../settings/settings-store.svelte"

	let { store }: { store: AccountStore } = $props()
	const region = $derived(appSettings.roValraRegion),
		hasPlace = $derived(
			/^\d+$/.test(store.launchInput.placeId.trim()) &&
				Number.isSafeInteger(Number(store.launchInput.placeId.trim())) &&
				Number(store.launchInput.placeId.trim()) > 0,
		),
		tooltip = $derived(
			!region
				? "Choose a preferred region in Settings first"
				: !hasPlace
					? "Enter a Place ID first"
					: `Automatically find a server in or near your preferred region (${appSettings.preferredRegionLabel}). Can be changed in settings`,
		)
</script>

<!-- The wrapper shows the tooltip while the button is disabled. -->
<span class="nearest-server" data-tooltip={tooltip} data-tooltip-side="top-end">
	<button
		class="icon-action bordered"
		type="button"
		aria-label="Fill nearest server"
		aria-busy={store.findingServer}
		disabled={!region || !hasPlace || store.launching || store.findingServer}
		onclick={() => void store.fillNearestServer()}>
		{#if store.findingServer}
			<LoaderCircle class="spinner" size={15} aria-hidden="true" />
		{:else}
			<LocateFixed size={15} aria-hidden="true" />
		{/if}
	</button>
</span>

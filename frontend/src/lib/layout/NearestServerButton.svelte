<script lang="ts">
	import crosshairIcon from "@phosphor-icons/core/regular/crosshair.svg?raw"
	import Icon from "../shared/Icon.svelte"
	import type { AccountStore } from "../accounts/account-store.svelte"
	import NearestServerDialog from "../dialogs/NearestServerDialog.svelte"
	import { appSettings } from "../settings/settings-store.svelte"
	import { NearestServerSearch } from "./nearest-server-search.svelte"

	let { store }: { store: AccountStore } = $props()
	let search = $state<NearestServerSearch | null>(null)
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

	$effect(() => {
		if (!store.vault.unlocked) search = null
	})
</script>

<span class="flex shrink-0" data-tooltip={tooltip} data-tooltip-side="top-end">
	<button
		class="button-icon size-8"
		type="button"
		aria-label="Find nearest server"
		aria-haspopup="dialog"
		disabled={!region || !hasPlace || store.launching}
		onclick={() => {
			search = new NearestServerSearch(placeId, store.selectedAccountId ?? 0)
		}}>
		<Icon svg={crosshairIcon} size={16} aria-hidden="true" />
	</button>
</span>

{#if search}
	<NearestServerDialog
		{search}
		onUseServer={(jobId) => {
			store.launchInput.jobId = jobId
			search = null
		}}
		onClose={() => (search = null)} />
{/if}

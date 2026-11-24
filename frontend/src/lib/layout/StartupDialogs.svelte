<script lang="ts">
	import type { AccountStore } from "../accounts/account-store.svelte"
	import { FileState } from "../backend/bridge"
	import AppLocationDialog from "../dialogs/AppLocationDialog.svelte"
	import DevelopmentBuildDialog from "../dialogs/DevelopmentBuildDialog.svelte"
	import LegacyVaultDialog from "../dialogs/LegacyVaultDialog.svelte"
	import VaultDialog from "../dialogs/VaultDialog.svelte"
	import type { AppStartup } from "./app-startup.svelte"

	// StartupDialogs shows one startup step at a time: the development build
	// warning, the data folder choice, then the vault until it unlocks.
	let { startup, store }: { startup: AppStartup; store: AccountStore } = $props()
</script>

{#if startup.developmentWarning === "open" && startup.build}
	<DevelopmentBuildDialog
		version={startup.build.version}
		onContinue={() => startup.closeDevelopmentWarning()} />
{:else if startup.developmentWarning === "closed"}
	{#if startup.appLocation && !startup.appLocation.initialized}
		<AppLocationDialog
			location={startup.appLocation}
			onConfirm={() => {
				startup.confirmAppLocation()
				void store.initialize()
			}} />
	{:else if store.vault.fileState === FileState.FileStateLegacy}
		<LegacyVaultDialog {store} />
	{:else if !store.vault.unlocked || store.vault.fileState === FileState.FileStateIncomplete}
		<VaultDialog {store} />
	{/if}
{/if}

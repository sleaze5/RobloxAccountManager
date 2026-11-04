<script lang="ts">
	import fileArrowDownIcon from "@phosphor-icons/core/regular/file-arrow-down.svg?raw"
	import fileArrowUpIcon from "@phosphor-icons/core/regular/file-arrow-up.svg?raw"
	import Icon from "../shared/Icon.svelte"
	import type { AccountStore } from "../accounts/account-store.svelte"
	import ExportVaultDialog from "../dialogs/ExportVaultDialog.svelte"
	import ImportVaultDialog from "../dialogs/ImportVaultDialog.svelte"
	import { chooseExportFile } from "../dialogs/vault-transfer"
	import type { NotificationCenter } from "../notifications/notification-center.svelte"

	let {
			store,
			notifications,
			query = "",
		}: {
			store: AccountStore
			notifications: NotificationCenter
			query?: string
		} = $props(),
		exporting = $state(false),
		importPath = $state(""),
		searchTerms = $derived(query.trim().toLowerCase().split(/\s+/).filter(Boolean))

	const exportTerms =
			"encrypted file password hint accounts saved passwords tags favorite places settings backup move computer save",
		importTerms =
			"merge overwrite export file password hint accounts cookies tags favorite places settings open"

	function matches(name: string, description: string): boolean {
		const text = `vault transfer export import ${name} ${description}`.toLowerCase()
		return searchTerms.every((term) => text.includes(term))
	}

	async function chooseImport(): Promise<void> {
		importPath = await chooseExportFile(notifications)
	}
</script>

{#if matches("Export vault data", exportTerms) || matches("Import vault data", importTerms)}
	<section class="settings-section" aria-labelledby="vault-transfer-title">
		<h3 id="vault-transfer-title">Export and import</h3>
		<div class="settings-rows">
			{#if matches("Export vault data", exportTerms)}
				<div class="settings-row">
					<div class="settings-row-copy">
						<strong>Export vault data</strong>
						<span
							>Save accounts, tags, favorite places, and settings to an
							encrypted file.</span>
					</div>
					<button
						class="btn"
						type="button"
						disabled={store.busy || !store.vault.unlocked}
						onclick={() => (exporting = true)}>
						<Icon svg={fileArrowUpIcon} size={16} aria-hidden="true" />
						Export data
					</button>
				</div>
			{/if}
			{#if matches("Import vault data", importTerms)}
				<div class="settings-row">
					<div class="settings-row-copy">
						<strong>Import vault data</strong>
						<span
							>Overwrite or merge this vault with data from an export
							file.</span>
					</div>
					<button
						class="btn"
						type="button"
						disabled={store.busy || !store.vault.unlocked}
						onclick={() => void chooseImport()}>
						<Icon svg={fileArrowDownIcon} size={16} aria-hidden="true" />
						Import data
					</button>
				</div>
			{/if}
		</div>
	</section>
{/if}

{#if exporting}
	<ExportVaultDialog {notifications} onClose={() => (exporting = false)} />
{/if}

{#if importPath}
	<ImportVaultDialog
		{store}
		{notifications}
		path={importPath}
		onClose={() => (importPath = "")} />
{/if}

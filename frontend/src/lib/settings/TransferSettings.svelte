<script lang="ts">
	import fileArrowDownIcon from "@phosphor-icons/core/regular/file-arrow-down.svg?raw"
	import fileArrowUpIcon from "@phosphor-icons/core/regular/file-arrow-up.svg?raw"
	import Icon from "../shared/Icon.svelte"
	import { onMount } from "svelte"
	import type { AccountStore } from "../accounts/account-store.svelte"
	import { accountBackend } from "../backend/bridge"
	import ExportVaultDialog from "../dialogs/ExportVaultDialog.svelte"
	import ImportVaultDialog from "../dialogs/ImportVaultDialog.svelte"
	import { getGamesStore } from "../games/games-store.svelte"
	import type { NotificationCenter } from "../notifications/notification-center.svelte"
	import { appSettings } from "./settings-store.svelte"

	let {
			store,
			notifications,
			query = "",
		}: {
			store: AccountStore
			notifications: NotificationCenter
			query?: string
		} = $props(),
		directory = $state(""),
		error = $state(""),
		activeDialog = $state<"export" | "import" | null>(null),
		searchTerms = $derived(query.trim().toLowerCase().split(/\s+/).filter(Boolean))
	const games = getGamesStore()

	function matches(name: string, description: string): boolean {
		const text = `vault transfer export import ${name} ${description}`.toLowerCase()
		return searchTerms.every((term) => text.includes(term))
	}

	onMount(() => {
		let disposed = false
		void (async () => {
			try {
				const exports = await accountBackend.ListVaultExports()
				if (!disposed) directory = exports.directory
			} catch {
				if (!disposed) error = "The export folder could not be found."
			}
		})()
		return () => {
			disposed = true
		}
	})

	async function copyPath(): Promise<void> {
		try {
			await navigator.clipboard.writeText(directory)
			notifications.show({
				id: "clipboard-copy",
				title: "Copied",
				tone: "success",
				message: "Export folder path copied to clipboard.",
			})
		} catch {
			notifications.show({
				id: "clipboard-copy",
				title: "Path not copied",
				tone: "error",
				message: "The export folder path could not be copied. Try again.",
			})
		}
	}

	function imported(): void {
		void store.initialize()
		games.reset()
		games.initialize()
		void appSettings.initialize(notifications)
	}
</script>

{#if matches("Export vault data", "encrypted file passcode accounts saved passwords tags favorite places settings backup move computer") || matches("Import vault data", "merge replace export file passcode accounts cookies tags favorite places settings")}
	<section class="settings-section" aria-labelledby="vault-transfer-title">
		<h3 id="vault-transfer-title">Export and import</h3>
		{#if error}<div class="notice-error" role="alert">{error}</div>{/if}
		<div class="settings-rows">
			{#if matches("Export vault data", "encrypted file passcode accounts saved passwords tags favorite places settings backup move computer")}
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
						disabled={store.busy || !store.vault.unlocked || !directory}
						onclick={() => (activeDialog = "export")}>
						<Icon svg={fileArrowUpIcon} size={16} aria-hidden="true" />
						Export data
					</button>
				</div>
			{/if}
			{#if matches("Import vault data", "merge replace export file passcode accounts cookies tags favorite places settings")}
				<div class="settings-row">
					<div class="settings-row-copy">
						<strong>Import vault data</strong>
						<span
							>Merge or replace this vault with data from an export file.</span>
					</div>
					<button
						class="btn"
						type="button"
						disabled={store.busy || !store.vault.unlocked || !directory}
						onclick={() => (activeDialog = "import")}>
						<Icon svg={fileArrowDownIcon} size={16} aria-hidden="true" />
						Import data
					</button>
				</div>
			{/if}
		</div>
	</section>
{/if}

{#if activeDialog === "export"}
	<ExportVaultDialog
		{directory}
		onCopyPath={() => void copyPath()}
		onClose={() => (activeDialog = null)} />
{/if}

{#if activeDialog === "import"}
	<ImportVaultDialog
		{directory}
		onCopyPath={() => void copyPath()}
		onImported={imported}
		onClose={() => (activeDialog = null)} />
{/if}

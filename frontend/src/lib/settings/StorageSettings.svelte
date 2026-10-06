<script lang="ts">
	import ArrowsLeftRightIcon from "phosphor-svelte/lib/ArrowsLeftRightIcon"
	import CopyIcon from "phosphor-svelte/lib/CopyIcon"
	import { onMount } from "svelte"
	import type { AccountStore } from "../accounts/account-store.svelte"
	import { accountBackend, FileState, StorageMode } from "../backend/bridge"
	import type { AppLocation } from "../backend/bridge"
	import MoveDataDialog from "../dialogs/MoveDataDialog.svelte"
	import type { NotificationCenter } from "../notifications/notification-center.svelte"
	import SettingsStatus from "./SettingsStatus.svelte"

	let {
			store,
			notifications,
			query = "",
		}: {
			store: AccountStore
			notifications: NotificationCenter
			query?: string
		} = $props(),
		location = $state<AppLocation | null>(null),
		error = $state(""),
		moving = $state(false),
		searchTerms = $derived(query.trim().toLowerCase().split(/\s+/).filter(Boolean))

	const other = $derived(
			location?.candidates?.find(
				(candidate) => candidate.mode !== location?.mode,
			),
		),
		currentLabel = $derived(modeLabel(location?.mode)),
		otherLabel = $derived(modeLabel(other?.mode))

	function modeLabel(mode: StorageMode | undefined): string {
		return mode === StorageMode.ModePortable ? "Portable" : "Standard"
	}

	function matches(name: string, description: string): boolean {
		const text = `vault storage ${name} ${description}`.toLowerCase()
		return searchTerms.every((term) => text.includes(term))
	}

	onMount(() => {
		let disposed = false
		void (async () => {
			try {
				const next = await accountBackend.GetAppLocation()
				if (!disposed) location = next
			} catch {
				if (!disposed) error = "The storage location could not be loaded."
			}
		})()
		return () => {
			disposed = true
		}
	})

	async function copyPath(): Promise<void> {
		if (!location) return
		try {
			await navigator.clipboard.writeText(location.directory)
			notifications.show({
				id: "clipboard-copy",
				title: "Copied",
				message: "Data folder path copied to clipboard.",
			})
		} catch {
			error = "The path could not be copied. Try again."
		}
	}
</script>

{#if location && (matches("Storage location", "data folder path portable standard copy") || (other && matches(`Move to ${otherLabel} storage`, "move data vault backups settings restart")))}
	<section class="settings-section" aria-labelledby="vault-storage-title">
		<h3 id="vault-storage-title">Storage</h3>
		{#if error}<div class="settings-inline-error" role="alert">{error}</div>{/if}
		<div class="settings-rows">
			{#if matches("Storage location", "data folder path portable standard copy")}
				<div class="settings-row">
					<div class="settings-row-copy">
						<strong>Storage location</strong>
						<span
							>{location.mode === StorageMode.ModePortable
								? "Next to the app."
								: "In your user folder."}</span>
						<span class="settings-row-path">{location.directory}</span>
						<SettingsStatus value={currentLabel} />
					</div>
					<button
						class="settings-row-action"
						type="button"
						onclick={() => void copyPath()}>
						<CopyIcon size={16} aria-hidden="true" />
						Copy path
					</button>
				</div>
			{/if}

			{#if other && matches(`Move to ${otherLabel} storage`, "move data vault backups settings restart")}
				<div class="settings-row">
					<div class="settings-row-copy">
						<strong>Move to {otherLabel} storage</strong>
						<span
							>Move the vault, backups, and settings {other.mode ===
							StorageMode.ModePortable
								? "next to the app"
								: "to your user folder"}. The app restarts.</span>
					</div>
					<button
						class="settings-row-action"
						type="button"
						disabled={store.busy ||
							store.vault.fileState !== FileState.FileStateReady}
						onclick={() => (moving = true)}>
						<ArrowsLeftRightIcon size={16} aria-hidden="true" />
						Move data
					</button>
				</div>
			{/if}
		</div>
	</section>

	{#if moving && other}
		<MoveDataDialog
			target={other}
			targetLabel={otherLabel}
			onClose={() => (moving = false)} />
	{/if}
{/if}

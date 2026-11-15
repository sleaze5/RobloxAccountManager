<script lang="ts">
	import xIcon from "@phosphor-icons/core/regular/x.svg?raw"
	import Icon from "../shared/Icon.svelte"
	import Dialog from "../shared/Dialog.svelte"
	import { onMount } from "svelte"
	import { accountBackend, RobloxCleanScope } from "../backend/bridge"
	import type { NotificationCenter } from "../notifications/notification-center.svelte"
	import { robloxClientOptions as options } from "../settings/roblox-client-options.svelte"
	import { errorMessage } from "../shared/errors"

	let {
		scope,
		notifications,
		onClose,
	}: {
		scope: RobloxCleanScope
		notifications: NotificationCenter
		onClose: () => void
	} = $props()
	const confirmationWord = "UNINSTALL"
	let runningProcessCount = $state(0),
		isCleaning = $state(false),
		error = $state(""),
		typedConfirmation = $state("")
	let disposed = false
	const isUninstall = $derived(scope === RobloxCleanScope.ScopeUninstall),
		cleaner = $derived(options.cleaner),
		administratorNote = $derived.by(() => {
			if (!cleaner?.elevationSupported || cleaner.isElevated) return ""
			return cleaner.useAdministratorRights
				? "Windows asks for administrator rights before Roblox is cleaned."
				: "Without administrator rights, items shared by all Windows users are skipped. Turn on Use administrator rights to include them."
		}),
		canConfirm = $derived(
			!isCleaning && (!isUninstall || typedConfirmation === confirmationWord),
		)

	async function clean(): Promise<void> {
		if (!canConfirm) return
		isCleaning = true
		error = ""
		try {
			const report = await accountBackend.CleanRoblox(scope)
			notifications.show({
				id: "roblox-clean",
				tone: report.hasSkippedItems ? "warning" : "success",
				title: isUninstall ? "Roblox uninstalled" : "Roblox data cleaned",
				message: report.hasSkippedItems
					? "Some items were in use or need administrator rights, so they were skipped."
					: isUninstall
						? "Roblox and its files, settings, and registry entries were deleted."
						: "Roblox caches and sign-in data were deleted.",
			})
			if (!disposed) onClose()
		} catch (cause) {
			if (!disposed) {
				error = errorMessage(cause, "Roblox could not be cleaned. Try again.")
			}
		} finally {
			if (!disposed) isCleaning = false
		}
	}

	function dismiss(): void {
		if (!isCleaning) onClose()
	}

	async function loadRunningProcessCount(): Promise<void> {
		try {
			const snapshot = await accountBackend.GetRobloxProcesses()
			if (!disposed) runningProcessCount = snapshot.processes?.length ?? 0
		} catch {
			// The count is only a hint. Cleaning closes running processes anyway.
		}
	}

	onMount(() => {
		void options.refreshCleaner()
		void loadRunningProcessCount()
		return () => {
			disposed = true
		}
	})
</script>

<Dialog
	class="max-w-md text-left"
	role="alertdialog"
	tabindex={-1}
	aria-labelledby="roblox-clean-title"
	aria-describedby="roblox-clean-description"
	onkeydown={(event) => {
		if (event.key === "Escape") {
			event.preventDefault()
			event.stopPropagation()
			dismiss()
		}
	}}>
	<div class="dialog-heading mb-2">
		<div>
			<h2 id="roblox-clean-title">
				{isUninstall ? "Uninstall Roblox?" : "Clean Roblox data?"}
			</h2>
			<p id="roblox-clean-description">
				{#if isUninstall}
					Deletes Roblox Player, Roblox Studio, their settings, shortcuts,
					file associations, and registry entries, and the Roblox folders in
					Documents, Pictures, and Videos. This cannot be undone.
				{:else}
					Deletes Roblox caches, cookies, and other sign-in data. Roblox
					Player, Roblox Studio, their settings, and Studio plugins stay
					installed.
				{/if}
			</p>
		</div>
		<button
			type="button"
			aria-label="Cancel"
			disabled={isCleaning}
			onclick={dismiss}>
			<Icon svg={xIcon} size={14} aria-hidden="true" />
		</button>
	</div>
	<div class="flex flex-col gap-2">
		{#if runningProcessCount > 0}
			<div class="notice-warning">
				{runningProcessCount === 1
					? "1 running Roblox process is"
					: `${runningProcessCount} running Roblox processes are`} closed first.
				Unsaved progress may be lost.
			</div>
		{/if}
		{#if isUninstall}
			<div class="notice-warning">
				Screenshots and videos in the Roblox folders are deleted with Roblox.
				Download Roblox again to play.
			</div>
		{:else if cleaner?.deleteLogsWhenCleaning}
			<div class="notice-note">
				Roblox logs are deleted too, so Logs Explorer loses the games it lists.
				Turn off Delete logs when cleaning in Roblox settings to keep them.
			</div>
		{:else if cleaner}
			<div class="notice-note">
				Roblox logs are kept, so Logs Explorer keeps the games it lists.
			</div>
		{/if}
		{#if administratorNote}
			<div class="notice-note">{administratorNote}</div>
		{/if}
	</div>
	{#if isUninstall}
		<label class="dialog-field">
			<span>Type {confirmationWord} to confirm</span>
			<input
				bind:value={typedConfirmation}
				autocomplete="off"
				spellcheck="false"
				disabled={isCleaning} />
		</label>
	{/if}
	{#if error}<div class="mt-3 notice-error" role="alert">{error}</div>{/if}
	<div class="dialog-actions">
		<button class="button" type="button" disabled={isCleaning} onclick={dismiss}
			>Cancel</button>
		<button
			class="button-danger"
			type="button"
			disabled={!canConfirm}
			onclick={() => void clean()}>
			{#if isUninstall}
				{isCleaning ? "Uninstalling…" : "Uninstall Roblox"}
			{:else}
				{isCleaning ? "Cleaning…" : "Clean Roblox data"}
			{/if}
		</button>
	</div>
</Dialog>

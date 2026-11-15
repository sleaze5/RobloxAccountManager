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
	// Uninstalling deletes the user's Roblox screenshots, videos, and places,
	// which have no backup, so it needs a typed confirmation.
	const uninstallConfirmationWord = "UNINSTALL"
	let runningProcessCount = $state(0),
		isCleaning = $state(false),
		error = $state(""),
		uninstallConfirmation = $state("")
	let disposed = false
	const isUninstall = $derived(scope === RobloxCleanScope.ScopeUninstall),
		isConfirmed = $derived(
			!isUninstall || uninstallConfirmation === uninstallConfirmationWord,
		),
		cleaner = $derived(options.cleaner),
		notes = $derived(
			[
				runningProcessCount === 1 &&
					"The running Roblox process closes first, so unsaved progress may be lost.",
				runningProcessCount > 1 &&
					`${runningProcessCount} running Roblox processes close first, so unsaved progress may be lost.`,
				!isUninstall &&
					cleaner?.deleteLogsWhenCleaning &&
					"Roblox logs are deleted too, so Logs Explorer loses the games it lists.",
			].filter((note) => typeof note === "string"),
		)

	async function clean(): Promise<void> {
		if (isCleaning || !isConfirmed) return
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
					Deletes Roblox Player, Roblox Studio, their settings and plugins,
					shortcuts, file associations, and registry entries. It also deletes
					your saved places, autosaves, screenshots, and videos in the Roblox
					folders. This cannot be undone.
				{:else}
					Deletes Roblox caches, cookies, and other sign-in data. Roblox
					Player, Roblox Studio, their settings, and your Studio plugins and
					places stay.
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
	{#if notes.length > 0}
		<div class="notice-note">{notes.join(" ")}</div>
	{/if}
	{#if isUninstall}
		<label class="dialog-field">
			<span>Type {uninstallConfirmationWord} to confirm</span>
			<input
				bind:value={uninstallConfirmation}
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
			disabled={isCleaning || !isConfirmed}
			onclick={() => void clean()}>
			{#if isUninstall}
				{isCleaning ? "Uninstalling…" : "Uninstall Roblox"}
			{:else}
				{isCleaning ? "Cleaning…" : "Clean Roblox data"}
			{/if}
		</button>
	</div>
</Dialog>

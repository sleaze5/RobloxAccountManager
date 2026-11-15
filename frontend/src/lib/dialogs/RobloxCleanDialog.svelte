<script lang="ts">
	import xIcon from "@phosphor-icons/core/regular/x.svg?raw"
	import Icon from "../shared/Icon.svelte"
	import Dialog from "../shared/Dialog.svelte"
	import { Events } from "@wailsio/runtime"
	import { onMount } from "svelte"
	import {
		accountBackend,
		RobloxCleanScope,
		type RobloxCleanReport,
	} from "../backend/bridge"
	import type { NotificationCenter } from "../notifications/notification-center.svelte"
	import {
		cleanReportSummary,
		robloxClientOptions as options,
	} from "../settings/roblox-client-options.svelte"
	import { errorMessage } from "../shared/errors"

	type CleanProgress = {
		stage: "closing-processes" | "deleting-files" | "deleting-registry-entries"
		completedCount: number
		totalCount: number
	}

	const stageLabels: Record<CleanProgress["stage"], string> = {
		"closing-processes": "Closing Roblox…",
		"deleting-files": "Deleting files…",
		"deleting-registry-entries": "Removing registry entries…",
	}

	let {
		scope,
		notifications,
		onClose,
	}: {
		scope: RobloxCleanScope
		notifications: NotificationCenter
		onClose: () => void
	} = $props()
	let runningProcessCount = $state(0),
		isCleaning = $state(false),
		progress = $state<CleanProgress | null>(null),
		report = $state<RobloxCleanReport | null>(null),
		error = $state("")
	let disposed = false
	const isUninstall = $derived(scope === RobloxCleanScope.ScopeUninstall),
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
		),
		progressPercent = $derived(
			progress && progress.totalCount > 0
				? (progress.completedCount / progress.totalCount) * 100
				: 0,
		),
		doneTitle = $derived(isUninstall ? "Roblox uninstalled" : "Roblox data cleaned")

	async function clean(): Promise<void> {
		if (isCleaning) return
		isCleaning = true
		progress = null
		error = ""
		try {
			const cleanReport = await accountBackend.CleanRoblox(scope)
			options.recordCleanReport(scope, cleanReport)
			if (disposed) {
				// The vault locked while cleaning, so the result goes to a notification.
				notifications.show({
					id: "roblox-clean",
					tone: cleanReport.skippedCount > 0 ? "warning" : "success",
					title: doneTitle,
					message: cleanReportSummary(cleanReport),
				})
				return
			}
			report = cleanReport
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
		const removeProgressListener = Events.On("roblox:clean-progress", (event) => {
			if (isCleaning) progress = event.data as CleanProgress
		})
		return () => {
			disposed = true
			removeProgressListener()
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
				{#if report}
					{doneTitle}
				{:else}
					{isUninstall ? "Uninstall Roblox?" : "Clean Roblox data?"}
				{/if}
			</h2>
			<p id="roblox-clean-description">
				{#if report}
					{cleanReportSummary(report)}
				{:else if isUninstall}
					Deletes Roblox Player, Roblox Studio, their settings and plugins,
					shortcuts, file associations, and registry entries. It also deletes
					your saved places, autosaves, screenshots, and videos in the Roblox
					folders. This cannot be undone.
				{:else}
					Deletes everything in the Roblox folder except Roblox Player, Roblox
					Studio, their settings, and Studio plugins. That includes caches,
					cookies, and other sign-in data.
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
	{#if isCleaning}
		<div class="transfer animate-enter" role="status">
			<strong>{progress ? stageLabels[progress.stage] : "Starting…"}</strong>
			<span
				>{progress && progress.totalCount > 0
					? `${progress.completedCount}/${progress.totalCount}`
					: "--"}</span>
			<div
				class="progress-bar"
				role="progressbar"
				aria-label={isUninstall ? "Uninstall progress" : "Cleaning progress"}
				aria-valuemin={0}
				aria-valuemax={100}
				aria-valuenow={Math.round(progressPercent)}>
				<div style:transform="scaleX({progressPercent / 100})"></div>
			</div>
		</div>
	{:else if !report && notes.length > 0}
		<div class="notice-note">{notes.join(" ")}</div>
	{/if}
	{#if error}<div class="mt-3 notice-error" role="alert">{error}</div>{/if}
	<div class="dialog-actions">
		{#if report}
			<button class="button-primary" type="button" onclick={onClose}
				>Close</button>
		{:else}
			<button class="button" type="button" disabled={isCleaning} onclick={dismiss}
				>Cancel</button>
			<button
				class="button-danger"
				type="button"
				disabled={isCleaning}
				onclick={() => void clean()}>
				{#if isUninstall}
					{isCleaning ? "Uninstalling…" : "Uninstall Roblox"}
				{:else}
					{isCleaning ? "Cleaning…" : "Clean Roblox data"}
				{/if}
			</button>
		{/if}
	</div>
</Dialog>

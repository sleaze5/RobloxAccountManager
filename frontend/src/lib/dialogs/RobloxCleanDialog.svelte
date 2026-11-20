<script lang="ts">
	import checkIcon from "@phosphor-icons/core/regular/check.svg?raw"
	import circleNotchIcon from "@phosphor-icons/core/regular/circle-notch.svg?raw"
	import warningIcon from "@phosphor-icons/core/regular/warning.svg?raw"
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

	type CleanPhase =
		| "closing-processes"
		| "removing-applications"
		| "removing-shortcuts"
		| "removing-registry-entries"
		| "deleting-user-content"
		| "deleting-caches"
		| "deleting-logs"
		| "deleting-temporary-files"

	type CleanPhaseProgress = {
		phase: CleanPhase
		state: "waiting" | "running" | "done"
		completedCount: number
		totalCount: number
		skippedCount: number
	}

	type CleanProgress = { phases: CleanPhaseProgress[] }

	const phaseLabels: Record<CleanPhase, string> = {
		"closing-processes": "Close Roblox",
		"removing-applications": "Remove Roblox and Studio",
		"removing-shortcuts": "Remove shortcuts",
		"removing-registry-entries": "Remove registry entries",
		"deleting-user-content": "Delete saved places, screenshots, and videos",
		"deleting-caches": "Delete caches and sign-in data",
		"deleting-logs": "Delete logs",
		"deleting-temporary-files": "Delete temporary files",
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
				runningProcessCount > 0 &&
					`Closes ${runningProcessCount} running Roblox ${runningProcessCount === 1 ? "process" : "processes"}.`,
				!isUninstall &&
					cleaner?.deleteLogsWhenCleaning &&
					"Clears Logs Explorer history.",
			].filter((note) => typeof note === "string"),
		),
		runningPhase = $derived(
			progress?.phases.find((phaseProgress) => phaseProgress.state === "running"),
		),
		isWaitingForAdministratorRights = $derived(
			isCleaning &&
				Boolean(
					progress?.phases.every(
						(phaseProgress) => phaseProgress.state === "waiting",
					),
				) &&
				Boolean(
					cleaner?.elevationSupported &&
					cleaner.useAdministratorRights &&
					!cleaner.isElevated,
				),
		),
		statusText = $derived(
			isWaitingForAdministratorRights
				? "Waiting for administrator rights…"
				: runningPhase
					? phaseLabels[runningPhase.phase]
					: "",
		),
		doneTitle = $derived(isUninstall ? "Roblox uninstalled" : "Roblox data cleaned")

	async function clean(): Promise<void> {
		if (isCleaning) return
		isCleaning = true
		progress = null
		error = ""
		try {
			const cleanReport = await accountBackend.CleanRoblox(scope)
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
				error = errorMessage(cause, "Could not clean Roblox. Try again.")
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
					Removes Roblox, Studio, and their files, including saved places,
					screenshots, and videos. This cannot be undone.
				{:else}
					Deletes Roblox caches and sign-in data. Roblox and its settings
					stay.
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
	{#if progress && (isCleaning || report || error)}
		<p class="sr-only" role="status">{statusText}</p>
		{#if isWaitingForAdministratorRights}
			<p class="mb-2 animate-enter text-xs text-fg-muted" aria-hidden="true">
				{statusText}
			</p>
		{/if}
		<ol
			class="flex animate-enter flex-col gap-1.5"
			aria-label={isUninstall ? "Uninstall steps" : "Cleaning steps"}
			aria-busy={isCleaning}>
			{#each progress.phases as phaseProgress (phaseProgress.phase)}
				{@const hasFailed = phaseProgress.state === "running" && error !== ""}
				<li
					class={[
						"flex min-w-0 items-center gap-2 text-xs",
						phaseProgress.state === "waiting" ? "text-fg-muted" : "text-fg",
					]}>
					{#if hasFailed}
						<Icon
							svg={xIcon}
							size={14}
							class="shrink-0 animate-pop text-danger"
							aria-hidden="true" />
						<span class="sr-only">Failed:</span>
					{:else if phaseProgress.state === "running"}
						<Icon
							svg={circleNotchIcon}
							size={14}
							class="shrink-0 spinner"
							aria-hidden="true" />
						<span class="sr-only">In progress:</span>
					{:else if phaseProgress.state === "waiting"}
						<Icon
							svg={circleNotchIcon}
							size={14}
							class="shrink-0"
							aria-hidden="true" />
						<span class="sr-only">Waiting:</span>
					{:else if phaseProgress.skippedCount > 0}
						<Icon
							svg={warningIcon}
							size={14}
							class="shrink-0 animate-pop text-warning"
							aria-hidden="true" />
						<span class="sr-only">Done with skipped items:</span>
					{:else}
						<Icon
							svg={checkIcon}
							size={14}
							class="shrink-0 animate-bloom text-success"
							aria-hidden="true" />
						<span class="sr-only">Done:</span>
					{/if}
					<span class="min-w-0 truncate"
						>{phaseLabels[phaseProgress.phase]}</span>
					{#if !hasFailed && phaseProgress.state === "running" && phaseProgress.totalCount > 0}
						<span
							class="ml-auto shrink-0 font-mono text-2xs text-fg-muted tabular-nums"
							>{phaseProgress.completedCount}/{phaseProgress.totalCount}</span>
					{:else if phaseProgress.skippedCount > 0}
						<span
							class="ml-auto shrink-0 font-mono text-2xs text-warning tabular-nums"
							>{phaseProgress.skippedCount} skipped</span>
					{/if}
				</li>
			{/each}
		</ol>
	{:else if !isCleaning && !report && notes.length > 0}
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

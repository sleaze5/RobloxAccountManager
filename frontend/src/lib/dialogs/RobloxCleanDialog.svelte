<script lang="ts">
	import checkCircleIcon from "@phosphor-icons/core/regular/check-circle.svg?raw"
	import warningCircleIcon from "@phosphor-icons/core/regular/warning-circle.svg?raw"
	import xIcon from "@phosphor-icons/core/regular/x.svg?raw"
	import Icon from "../shared/Icon.svelte"
	import Dialog from "../shared/Dialog.svelte"
	import { Events } from "@wailsio/runtime"
	import { onMount, tick } from "svelte"
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
	import { formatCompactDuration } from "../shared/duration"
	import { errorMessage } from "../shared/errors"
	import CleanPhaseIcon from "./CleanPhaseIcon.svelte"
	import {
		cleanPhaseText,
		skipReasonText,
		type CleanActivity,
		type CleanPhaseProgress,
		type CleanProgress,
	} from "./roblox-cleaning"

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
		hasStarted = $state(false),
		isCleaning = $state(false),
		phases = $state<CleanPhaseProgress[]>([]),
		activities = $state<CleanActivity[]>([]),
		report = $state<RobloxCleanReport | null>(null),
		error = $state(""),
		durationMilliseconds = $state(0),
		activityLog = $state<HTMLDivElement>()
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
		doneTitle = $derived(
			isUninstall ? "Roblox uninstalled" : "Roblox data cleaned",
		),
		runningPhase = $derived(phases.find((phase) => phase.state === "running")),
		completedCount = $derived(
			phases.reduce((sum, phase) => sum + phase.completedCount, 0),
		),
		totalCount = $derived(phases.reduce((sum, phase) => sum + phase.totalCount, 0)),
		isWaitingForAdministratorRights = $derived(
			isCleaning &&
				phases.length > 0 &&
				phases.every((phase) => phase.state === "waiting") &&
				Boolean(
					cleaner?.elevationSupported &&
					cleaner.useAdministratorRights &&
					!cleaner.isElevated,
				),
		),
		statusText = $derived.by(() => {
			if (report) return "Finished"
			if (error) return "Stopped"
			if (isWaitingForAdministratorRights)
				return "Waiting for administrator rights…"
			return runningPhase
				? cleanPhaseText[runningPhase.phase].progressLabel
				: "Starting…"
		}),
		progressFraction = $derived(
			report ? 1 : totalCount > 0 ? completedCount / totalCount : 0,
		),
		countText = $derived(
			[
				totalCount > 0 && `${completedCount} of ${totalCount}`,
				hasStarted &&
					!isCleaning &&
					formatCompactDuration(durationMilliseconds),
			]
				.filter((part) => typeof part === "string")
				.join(" · "),
		)

	async function clean(): Promise<void> {
		if (isCleaning) return
		hasStarted = true
		isCleaning = true
		phases = []
		activities = []
		error = ""
		const startedAt = performance.now()
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
			// The report can arrive before the last progress event, and a
			// finished run has finished every phase.
			for (const phase of phases) {
				phase.state = "done"
				phase.completedCount = phase.totalCount
			}
			report = cleanReport
		} catch (cause) {
			if (!disposed) {
				error = errorMessage(cause, "Could not clean Roblox. Try again.")
			}
		} finally {
			if (!disposed) {
				durationMilliseconds = performance.now() - startedAt
				isCleaning = false
			}
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

	// New activity scrolls into view unless the user scrolled up to read.
	$effect(() => {
		const entryCount = activities.length,
			log = activityLog
		if (!log || entryCount === 0) return
		const isAtBottom = log.scrollHeight - log.scrollTop - log.clientHeight < 32
		if (isAtBottom) {
			void tick().then(() => log.scrollTo({ top: log.scrollHeight }))
		}
	})

	onMount(() => {
		void options.refreshCleaner()
		void loadRunningProcessCount()
		const removeProgressListener = Events.On("roblox:clean-progress", (event) => {
			if (!hasStarted || report) return
			const progress = event.data as CleanProgress
			phases = progress.phases
			if (progress.activity) activities.push(progress.activity)
		})
		return () => {
			disposed = true
			removeProgressListener()
		}
	})
</script>

<Dialog
	class={[hasStarted ? "max-w-xl" : "max-w-md", "text-left"]}
	role="alertdialog"
	hasFlexibleBody
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
				{:else if hasStarted}
					{isUninstall ? "Uninstalling Roblox" : "Cleaning Roblox data"}
				{:else}
					{isUninstall ? "Uninstall Roblox?" : "Clean Roblox data?"}
				{/if}
			</h2>
			<p id="roblox-clean-description">
				{#if isUninstall}
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
			aria-label={report ? "Close" : "Cancel"}
			disabled={isCleaning}
			onclick={dismiss}>
			<Icon svg={xIcon} size={14} aria-hidden="true" />
		</button>
	</div>
	{#if hasStarted && phases.length > 0}
		<div
			class="-mx-4 mt-2 flex min-h-0 animate-enter flex-col gap-3 overflow-y-auto px-4">
			<p class="sr-only" role="status">{statusText}</p>
			<ol
				class="flex flex-col"
				aria-label={isUninstall ? "Uninstall steps" : "Cleaning steps"}
				aria-busy={isCleaning}>
				{#each phases as phase, index (phase.phase)}
					{@const hasFailed = phase.state === "running" && error !== ""}
					<li class="relative flex gap-3 pb-3 last:pb-0">
						{#if index < phases.length - 1}
							<span
								class={[
									"absolute top-6 bottom-1 left-2 w-px transition-colors duration-150",
									phase.state === "done"
										? "bg-success/40"
										: "bg-line",
								]}
								aria-hidden="true"></span>
						{/if}
						<span class="flex h-5 items-center">
							<CleanPhaseIcon {phase} {hasFailed} />
						</span>
						<div class="flex min-w-0 flex-1 flex-col gap-0.5">
							<span
								class={[
									"text-sm font-medium",
									phase.state === "waiting"
										? "text-fg-muted"
										: "text-fg",
								]}>{cleanPhaseText[phase.phase].title}</span>
							<span class="text-xs text-fg-muted"
								>{cleanPhaseText[phase.phase].description}</span>
						</div>
						{#if phase.skippedCount > 0}
							<span
								class="mt-0.5 shrink-0 font-mono text-2xs text-warning tabular-nums"
								>{phase.skippedCount} skipped</span>
						{:else if !hasFailed && phase.state === "running" && phase.totalCount > 0}
							<span
								class="mt-0.5 shrink-0 font-mono text-2xs text-fg-muted tabular-nums"
								>{phase.completedCount}/{phase.totalCount}</span>
						{/if}
					</li>
				{/each}
			</ol>
			<div class="flex flex-col gap-1.5">
				<div
					class="progress-bar"
					role="progressbar"
					aria-label={isUninstall
						? "Uninstall progress"
						: "Cleaning progress"}
					aria-valuemin={0}
					aria-valuemax={100}
					aria-valuenow={Math.round(progressFraction * 100)}>
					<div style:transform="scaleX({progressFraction})"></div>
				</div>
				<div
					class="flex justify-between gap-3 font-mono text-2xs text-fg-muted tabular-nums"
					aria-hidden="true">
					<span class="truncate">{statusText}</span>
					<span class="shrink-0">{countText}</span>
				</div>
			</div>
			{#if report}
				<div
					class={[
						"flex items-center gap-2",
						report.skippedCount > 0 ? "notice-warning" : "notice-note",
					]}>
					{#if report.skippedCount > 0}
						<Icon
							svg={warningCircleIcon}
							size={14}
							class="shrink-0 animate-bloom"
							aria-hidden="true" />
					{:else}
						<Icon
							svg={checkCircleIcon}
							size={14}
							class="shrink-0 animate-bloom text-success"
							aria-hidden="true" />
					{/if}
					<span class="text-fg"
						>{doneTitle}. {cleanReportSummary(report)}</span>
				</div>
			{/if}
			<div class="flex flex-col gap-1.5">
				<h3 class="text-xs font-semibold text-fg-muted">Activity</h3>
				<div
					bind:this={activityLog}
					class="max-h-40 overflow-y-auto rounded-md border border-line bg-fill px-3 py-2 font-mono text-2xs leading-5 select-text"
					role="log"
					aria-label="Activity">
					{#each activities as activity, index (index)}
						<p
							class={[
								"wrap-anywhere",
								activity.action === "skipped" && "text-warning",
							]}>
							{#if activity.action === "closed"}Closed{:else if activity.action === "deleted"}Deleted{:else}Skipped{/if}
							{activity.item}{#if activity.skipReason}: {skipReasonText[
									activity.skipReason
								]}{/if}
						</p>
					{:else}
						<p class="text-fg-muted">
							{isCleaning
								? "Nothing changed yet."
								: "Nothing needed cleaning."}
						</p>
					{/each}
				</div>
			</div>
		</div>
	{:else if !hasStarted && notes.length > 0}
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

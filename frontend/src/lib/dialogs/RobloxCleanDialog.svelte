<script lang="ts">
	import xIcon from "@phosphor-icons/core/regular/x.svg?raw"
	import Icon from "../shared/Icon.svelte"
	import Dialog from "../shared/Dialog.svelte"
	import { onMount } from "svelte"
	import { accountBackend, RobloxCleanScope } from "../backend/bridge"
	import type { NotificationCenter } from "../notifications/notification-center.svelte"
	import CleanPhaseIcon from "../settings/CleanPhaseIcon.svelte"
	import { robloxClientOptions as options } from "../settings/roblox-client-options.svelte"
	import {
		cleanDoneTitle,
		cleanPhaseText,
		cleanReportSummary,
		robloxCleaning as run,
	} from "../settings/roblox-cleaning.svelte"

	let {
		notifications,
		onClose,
	}: {
		notifications: NotificationCenter
		onClose: () => void
	} = $props()
	let runningProcessCount = $state(0),
		hasStarted = $state(false)
	let disposed = false
	const isShowingRun = $derived(
			hasStarted && run.scope === RobloxCleanScope.ScopeData,
		),
		report = $derived(isShowingRun ? run.report : null),
		error = $derived(isShowingRun ? run.error : ""),
		isCleaning = $derived(run.isRunning),
		notes = $derived(
			[
				runningProcessCount > 0 &&
					`Closes ${runningProcessCount} running Roblox ${runningProcessCount === 1 ? "process" : "processes"}.`,
				options.cleaner?.deleteLogsWhenCleaning &&
					"Clears Logs Explorer history.",
			].filter((note) => typeof note === "string"),
		)

	function clean(): void {
		hasStarted = true
		void run.start(RobloxCleanScope.ScopeData, notifications)
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
		const endCleaningView = run.view()
		return () => {
			disposed = true
			endCleaningView()
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
				{report
					? cleanDoneTitle(RobloxCleanScope.ScopeData)
					: "Clean Roblox data?"}
			</h2>
			<p id="roblox-clean-description">
				{#if report}
					{cleanReportSummary(report)}
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
	{#if isShowingRun && run.phases.length > 0}
		<ol
			class="flex animate-enter flex-col gap-1.5"
			aria-label="Cleaning steps"
			aria-busy={isCleaning}>
			{#each run.phases as phase (phase.phase)}
				{@const hasFailed = phase.state === "running" && error !== ""}
				<li
					class={[
						"flex min-w-0 items-center gap-2 text-xs",
						phase.state === "waiting" ? "text-fg-muted" : "text-fg",
					]}>
					<CleanPhaseIcon {phase} {hasFailed} size={14} />
					<span class="min-w-0 truncate"
						>{cleanPhaseText[phase.phase].title}</span>
					{#if phase.skippedCount > 0}
						<span
							class="ml-auto shrink-0 font-mono text-2xs text-warning tabular-nums"
							>{phase.skippedCount} skipped</span>
					{:else if !hasFailed && phase.state === "running" && phase.totalCount > 0}
						<span
							class="ml-auto shrink-0 font-mono text-2xs text-fg-muted tabular-nums"
							>{phase.completedCount}/{phase.totalCount}</span>
					{/if}
				</li>
			{/each}
		</ol>
	{:else if !isCleaning && notes.length > 0}
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
				onclick={clean}>
				{isShowingRun && isCleaning ? "Cleaning…" : "Clean Roblox data"}
			</button>
		{/if}
	</div>
</Dialog>

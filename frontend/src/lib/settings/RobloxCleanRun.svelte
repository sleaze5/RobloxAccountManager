<script lang="ts">
	import checkCircleIcon from "@phosphor-icons/core/regular/check-circle.svg?raw"
	import warningCircleIcon from "@phosphor-icons/core/regular/warning-circle.svg?raw"
	import { tick } from "svelte"
	import type { RobloxCleanScope } from "../backend/bridge"
	import { formatCompactDuration } from "../shared/duration"
	import Icon from "../shared/Icon.svelte"
	import CleanPhaseIcon from "./CleanPhaseIcon.svelte"
	import { robloxClientOptions as options } from "./roblox-client-options.svelte"
	import {
		cleanDoneTitle,
		cleanPhaseText,
		cleanReportSummary,
		robloxCleaning as run,
		skipReasonText,
	} from "./roblox-cleaning.svelte"

	let { scope, label }: { scope: RobloxCleanScope; label: string } = $props()
	let activityLog = $state<HTMLDivElement>()

	const isWaitingForAdministratorRights = $derived(
			run.isRunning &&
				run.phases.length > 0 &&
				run.phases.every((phase) => phase.state === "waiting") &&
				Boolean(
					options.cleaner?.elevationSupported &&
					options.cleaner.useAdministratorRights &&
					!options.cleaner.isElevated,
				),
		),
		statusText = $derived.by(() => {
			if (run.report) return "Finished"
			if (run.error) return "Stopped"
			if (isWaitingForAdministratorRights)
				return "Waiting for administrator rights…"
			return run.runningPhase
				? cleanPhaseText[run.runningPhase.phase].progressLabel
				: "Starting…"
		}),
		progressFraction = $derived(
			run.report
				? 1
				: run.totalCount > 0
					? run.completedCount / run.totalCount
					: 0,
		),
		countText = $derived(
			[
				run.totalCount > 0 && `${run.completedCount} of ${run.totalCount}`,
				!run.isRunning && formatCompactDuration(run.durationMilliseconds),
			]
				.filter((part) => typeof part === "string")
				.join(" · "),
		)

	// New activity scrolls into view unless the user scrolled up to read.
	$effect(() => {
		const entryCount = run.activities.length,
			log = activityLog
		if (!log || entryCount === 0) return
		const isAtBottom = log.scrollHeight - log.scrollTop - log.clientHeight < 32
		if (isAtBottom) {
			void tick().then(() => log.scrollTo({ top: log.scrollHeight }))
		}
	})
</script>

{#if run.scope === scope && run.phases.length > 0}
	<div
		class="flex animate-enter flex-col gap-3 border-b border-line-subtle px-3 pt-1 pb-3 last:border-b-0"
		role="group"
		aria-label={label}>
		<p class="sr-only" role="status">{statusText}</p>
		<ol class="flex flex-col" aria-label="Steps" aria-busy={run.isRunning}>
			{#each run.phases as phase, index (phase.phase)}
				{@const hasFailed = phase.state === "running" && run.error !== ""}
				<li class="relative flex gap-3 pb-3 last:pb-0">
					{#if index < run.phases.length - 1}
						<span
							class={[
								"absolute top-6 bottom-1 left-2 w-px transition-colors duration-150",
								phase.state === "done" ? "bg-success/40" : "bg-line",
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
								phase.state === "waiting" ? "text-fg-muted" : "text-fg",
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
				aria-label="Overall progress"
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
		{#if run.report}
			<div
				class={[
					"flex items-center gap-2",
					run.report.skippedCount > 0 ? "notice-warning" : "notice-note",
				]}>
				{#if run.report.skippedCount > 0}
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
					>{cleanDoneTitle(scope)}. {cleanReportSummary(run.report)}</span>
			</div>
		{:else if run.error}
			<div class="notice-error" role="alert">{run.error}</div>
		{/if}
		<div class="flex flex-col gap-1.5">
			<div class="flex items-center justify-between gap-2">
				<h4 class="text-xs font-semibold text-fg-muted">Activity</h4>
				{#if !run.isRunning}
					<button
						class="button-text"
						type="button"
						onclick={() => run.dismiss()}>Hide results</button>
				{/if}
			</div>
			<div
				bind:this={activityLog}
				class="max-h-40 overflow-y-auto rounded-md border border-line bg-fill px-3 py-2 font-mono text-2xs leading-5 select-text"
				role="log"
				aria-label="Activity">
				{#each run.activities as activity, index (index)}
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
						{run.isRunning
							? "Nothing changed yet."
							: "Nothing needed cleaning."}
					</p>
				{/each}
			</div>
		</div>
	</div>
{/if}

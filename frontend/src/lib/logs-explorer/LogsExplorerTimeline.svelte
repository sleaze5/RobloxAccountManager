<script lang="ts">
	import DotsThreeIcon from "phosphor-svelte/lib/DotsThreeIcon"
	import { VisitKind } from "../backend/bridge"
	import type { LogsExplorerVisit } from "../backend/bridge"
	import Timestamp from "../shared/Timestamp.svelte"
	import LogsExplorerVisitMenu from "./LogsExplorerVisitMenu.svelte"
	import LogsExplorerVisitDetails from "./LogsExplorerVisitDetails.svelte"
	import { formatElapsed } from "./logs-explorer-time"

	let {
		visits,
		logStartedAtMs,
		newestFirst,
		onFillLaunch,
	}: {
		visits: LogsExplorerVisit[]
		logStartedAtMs: number
		newestFirst: boolean
		onFillLaunch: (visit: LogsExplorerVisit) => void
	} = $props()
	let activeMenu = $state<{ visit: LogsExplorerVisit; trigger: HTMLElement } | null>(
		null,
	)
	const menuID = $props.id()
	const orderedVisits = $derived(newestFirst ? visits.toReversed() : visits)

	$effect(() => {
		void orderedVisits
		activeMenu = null
	})

	function toggleMenu(event: MouseEvent, visit: LogsExplorerVisit): void {
		event.preventDefault()
		const trigger = event.currentTarget as HTMLElement
		activeMenu = activeMenu?.trigger === trigger ? null : { visit, trigger }
	}
</script>

<ol
	class="logs-explorer-timeline"
	aria-label={`Game joins and teleports, ${newestFirst ? "newest" : "oldest"} first`}>
	{#each orderedVisits as visit, index (index)}
		<li class="logs-explorer-visit">
			<div class="logs-explorer-timeline-point">
				<span
					class="logs-explorer-timeline-marker"
					aria-label="Time from log start">
					{#if logStartedAtMs && visit.startedAtMs}
						+{formatElapsed(visit.startedAtMs - logStartedAtMs)}
					{:else}--:--{/if}
				</span>
			</div>
			<div class="logs-explorer-visit-card">
				<div class="logs-explorer-visit-heading">
					<h3>
						{visit.kind === VisitKind.VisitRejoin
							? "Rejoin"
							: visit.kind === VisitKind.VisitTeleport
								? "Teleport"
								: visit === visits[0]
									? "Initial join"
									: "Join"}
					</h3>
					{#if visit.startedAtMs && visit.endedAtMs}
						<span
							class="logs-explorer-visit-duration"
							data-tooltip={visit.endEstimated
								? "Estimated duration"
								: "Duration"}>
							{visit.endEstimated ? "≈" : ""}{formatElapsed(
								visit.durationMs,
							)}
						</span>
					{/if}
					<button
						class="logs-explorer-visit-actions icon-action"
						type="button"
						aria-label="Event actions"
						aria-haspopup="menu"
						aria-expanded={activeMenu?.visit === visit}
						aria-controls={activeMenu?.visit === visit ? menuID : undefined}
						data-tooltip="Event actions"
						data-tooltip-side="bottom-end"
						onclick={(event) => toggleMenu(event, visit)}
						oncontextmenu={(event) => toggleMenu(event, visit)}
						onkeydown={(event) => {
							if (
								event.key === "ArrowDown" ||
								event.key === "ContextMenu" ||
								(event.shiftKey && event.key === "F10")
							) {
								event.preventDefault()
								activeMenu = { visit, trigger: event.currentTarget }
							}
						}}>
						<DotsThreeIcon size={18} aria-hidden="true" />
					</button>
				</div>
				<LogsExplorerVisitDetails {visit} />
				<div class="logs-explorer-visit-time">
					{#if visit.startedAtMs}<Timestamp
							value={visit.startedAtMs} />{:else}Time unavailable{/if}
				</div>
			</div>
		</li>
	{/each}
</ol>

{#if activeMenu}
	<LogsExplorerVisitMenu
		id={menuID}
		anchor={activeMenu.trigger}
		onClose={() => (activeMenu = null)}
		onFillLaunch={() => {
			if (!activeMenu) return
			const { visit } = activeMenu
			activeMenu = null
			onFillLaunch(visit)
		}} />
{/if}

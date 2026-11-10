<script lang="ts">
	import dotsThreeIcon from "@phosphor-icons/core/regular/dots-three.svg?raw"
	import listPlusIcon from "@phosphor-icons/core/regular/list-plus.svg?raw"
	import Icon from "../shared/Icon.svelte"
	import { untrack } from "svelte"
	import { VisitKind } from "../backend/bridge"
	import type { LogsExplorerVisit } from "../backend/bridge"
	import Timestamp from "../shared/Timestamp.svelte"
	import AnchoredActionMenu from "../shared/AnchoredActionMenu.svelte"
	import LogsExplorerVisitDetails from "./LogsExplorerVisitDetails.svelte"
	import { formatClock } from "../shared/duration"

	let {
		visits,
		logStartedAtMs,
		newestFirst,
		onFillLaunch,
		onSearchPlace,
	}: {
		visits: LogsExplorerVisit[]
		logStartedAtMs: number
		newestFirst: boolean
		onFillLaunch: (visit: LogsExplorerVisit) => void
		onSearchPlace: (placeId: number) => void
	} = $props()
	let activeMenu = $state<{
		visit: LogsExplorerVisit
		trigger: HTMLElement
		open: boolean
	} | null>(null)
	const menuID = $props.id()
	const orderedVisits = $derived(newestFirst ? visits.toReversed() : visits),
		openVisit = $derived(activeMenu?.open ? activeMenu.visit : null)

	$effect(() => {
		void orderedVisits
		untrack(closeMenu)
	})

	function toggleMenu(event: MouseEvent, visit: LogsExplorerVisit): void {
		event.preventDefault()
		const trigger = event.currentTarget as HTMLElement
		if (activeMenu?.open && activeMenu.trigger === trigger) closeMenu()
		else activeMenu = { visit, trigger, open: true }
	}

	function closeMenu(): void {
		if (activeMenu) activeMenu = { ...activeMenu, open: false }
	}
</script>

<ol
	class="timeline"
	aria-label={`Game joins and teleports, ${newestFirst ? "newest" : "oldest"} first`}>
	{#each orderedVisits as visit, index (index)}
		<li class="relative col-span-full grid grid-cols-subgrid gap-3 pb-3 last:pb-0">
			<div
				class={[
					"relative",
					index < orderedVisits.length - 1 &&
						"after:absolute after:top-6 after:-bottom-3 after:left-1/2 after:w-px after:bg-line-strong",
				]}>
				<span
					class="relative z-10 grid min-h-6 place-items-center rounded-md border border-line-strong bg-surface px-1 font-mono text-2xs whitespace-nowrap text-fg-muted"
					aria-label="Time from log start">
					{#if logStartedAtMs && visit.startedAtMs}
						+{formatClock((visit.startedAtMs - logStartedAtMs) / 1000)}
					{:else}--:--{/if}
				</span>
			</div>
			<div
				class="flex min-w-0 flex-col gap-2 rounded-lg border border-line px-3 py-2">
				<div class="flex min-h-6 flex-wrap items-center gap-2">
					<h3 class="text-sm font-semibold">
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
							class="font-mono text-2xs text-fg-muted"
							data-tooltip={visit.endEstimated
								? "Estimated duration"
								: "Duration"}>
							{visit.endEstimated ? "≈" : ""}{formatClock(
								visit.durationMs / 1000,
							)}
						</span>
					{/if}
					<button
						class="ml-auto button-ghost-sm"
						type="button"
						aria-label="Event actions"
						aria-haspopup="menu"
						aria-expanded={openVisit === visit}
						aria-controls={openVisit === visit ? menuID : undefined}
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
								activeMenu = {
									visit,
									trigger: event.currentTarget,
									open: true,
								}
							}
						}}>
						<Icon svg={dotsThreeIcon} size={14} aria-hidden="true" />
					</button>
				</div>
				<LogsExplorerVisitDetails {visit} {onSearchPlace} />
				<div class="flex justify-end text-2xs">
					{#if visit.startedAtMs}<Timestamp
							value={visit.startedAtMs} />{:else}Time unavailable{/if}
				</div>
			</div>
		</li>
	{/each}
</ol>

{#if activeMenu?.open}
	{@const { visit } = activeMenu}
	<AnchoredActionMenu
		id={menuID}
		label="Event actions"
		anchor={activeMenu.trigger}
		actions={[
			{
				label: "Fill launch options",
				icon: listPlusIcon,
				onSelect: () => {
					closeMenu()
					onFillLaunch(visit)
				},
			},
		]}
		onClose={closeMenu} />
{/if}

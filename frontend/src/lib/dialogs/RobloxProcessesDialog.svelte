<script lang="ts">
	import CheckIcon from "phosphor-svelte/lib/CheckIcon"
	import CircleNotchIcon from "phosphor-svelte/lib/CircleNotchIcon"
	import MinusIcon from "phosphor-svelte/lib/MinusIcon"
	import ArrowsClockwiseIcon from "phosphor-svelte/lib/ArrowsClockwiseIcon"
	import XIcon from "phosphor-svelte/lib/XIcon"
	import { onMount, tick } from "svelte"
	import {
		accountBackend,
		type RobloxProcess,
		type RobloxProcessSnapshot,
	} from "../backend/bridge"
	import {
		modalBackdropIn,
		modalBackdropOut,
		modalCardIn,
		modalCardOut,
		modalFocus,
	} from "../shared/presence"

	let { onClose }: { onClose: () => void } = $props()
	let snapshot = $state<RobloxProcessSnapshot | null>(null),
		selected = $state<string[]>([]),
		loading = $state(true),
		killing = $state(false),
		loadError = $state(""),
		error = $state(""),
		cancelButton = $state<HTMLButtonElement>()
	let disposed = false
	const processes = $derived(snapshot?.processes ?? []),
		selectedProcesses = $derived(
			processes.filter((process) => selected.includes(processKey(process))),
		),
		allSelected = $derived(
			processes.length > 0 && selectedProcesses.length === processes.length,
		),
		someSelected = $derived(selectedProcesses.length > 0 && !allSelected)

	function processKey(process: RobloxProcess): string {
		return `${process.pid}:${process.startTime}`
	}

	async function refresh(): Promise<void> {
		loading = true
		try {
			const state = await accountBackend.GetRobloxProcesses()
			if (disposed) return
			const previous = new Set(processes.map(processKey))
			selected = (state.processes ?? [])
				.map(processKey)
				.filter((key) => !previous.has(key) || selected.includes(key))
			snapshot = state
			loadError = ""
		} catch {
			if (!disposed) {
				loadError = "Could not load Roblox processes. Try refreshing."
			}
		} finally {
			if (!disposed) loading = false
		}
	}

	function toggleProcess(key: string, checked: boolean): void {
		selected = checked
			? [...selected, key]
			: selected.filter((item) => item !== key)
	}

	function toggleAll(): void {
		selected = allSelected ? [] : processes.map(processKey)
	}

	async function killSelected(): Promise<void> {
		if (killing || loading || selectedProcesses.length === 0) return
		const targets = [...selectedProcesses]
		killing = true
		error = ""
		try {
			await accountBackend.KillRobloxProcesses(targets)
		} catch {
			if (!disposed) {
				error = "Some Roblox processes could not be killed. Try again."
			}
		} finally {
			if (!disposed) {
				await refresh()
				killing = false
				await tick()
				cancelButton?.focus()
			}
		}
	}

	function dismiss(): void {
		if (!killing) onClose()
	}

	onMount(() => {
		void refresh()
		return () => {
			disposed = true
		}
	})
</script>

<div class="modal-backdrop" in:modalBackdropIn out:modalBackdropOut>
	<div
		class="modal-card roblox-process-card"
		role="alertdialog"
		tabindex="-1"
		aria-modal="true"
		aria-labelledby="roblox-process-title"
		use:modalFocus
		in:modalCardIn
		out:modalCardOut
		onkeydown={(event) => {
			if (event.key === "Escape") {
				event.preventDefault()
				event.stopPropagation()
				dismiss()
			}
		}}>
		<div class="modal-heading">
			<h2 id="roblox-process-title">Kill Roblox processes</h2>
			<div class="process-heading-actions">
				<button
					class="icon-action"
					type="button"
					aria-label="Refresh process list"
					data-tooltip="Refresh process list"
					disabled={loading || killing || snapshot?.supported === false}
					onclick={() => void refresh()}>
					{#if loading}<CircleNotchIcon
							class="spinner"
							size={16}
							aria-hidden="true" />
					{:else}<ArrowsClockwiseIcon size={16} aria-hidden="true" />{/if}
				</button>
				<button
					class="icon-action"
					type="button"
					aria-label="Cancel"
					disabled={killing}
					onclick={dismiss}><XIcon size={17} aria-hidden="true" /></button>
			</div>
		</div>
		<div class="process-results">
			<div class="cookie-results-toolbar">
				<label class="cookie-select-all">
					<span class="cookie-selection">
						<input
							type="checkbox"
							checked={allSelected}
							indeterminate={someSelected}
							disabled={loading || killing || processes.length === 0}
							onchange={toggleAll} />
						<span aria-hidden="true">
							{#if someSelected}<MinusIcon
									size={11}
									weight="bold"
									aria-hidden="true" />
							{:else}<CheckIcon
									size={11}
									weight="bold"
									aria-hidden="true" />{/if}
						</span>
					</span>
					<span>{allSelected ? "Deselect all" : "Select all"}</span>
				</label>
				<span class="cookie-selection-count" role="status"
					>{selectedProcesses.length} of {processes.length} selected</span>
			</div>
			<div class="process-table-scroll">
				<table aria-label="Roblox processes" aria-busy={loading || killing}>
					<tbody>
						{#each processes as process (processKey(process))}
							<tr class:selected={selected.includes(processKey(process))}>
								<td class="selection-column">
									<label class="cookie-selection">
										<input
											type="checkbox"
											checked={selected.includes(
												processKey(process),
											)}
											disabled={loading || killing}
											aria-label={`Select ${process.name}, PID ${process.pid}`}
											onchange={(event) =>
												toggleProcess(
													processKey(process),
													event.currentTarget.checked,
												)} />
										<span aria-hidden="true"
											><CheckIcon
												size={11}
												weight="bold"
												aria-hidden="true" /></span>
									</label>
								</td>
								<td class="process-name">{process.name}</td>
								<td class="pid-column">PID: {process.pid}</td>
							</tr>
						{:else}
							<tr
								><td colspan="3" class="process-empty">
									<span role="status">
										{#if loading}Loading processes…
										{:else if snapshot?.supported === false}Available
											on Windows only.
										{:else if loadError}Process list unavailable.
										{:else}No Roblox processes are running.{/if}
									</span>
								</td></tr>
						{/each}
					</tbody>
				</table>
			</div>
		</div>
		{#if error || loadError}<div class="modal-error" role="alert">
				{error || loadError}
			</div>{/if}
		<div class="modal-actions">
			<button
				bind:this={cancelButton}
				type="button"
				disabled={killing}
				onclick={dismiss}>Cancel</button>
			<button
				class="danger-button"
				type="button"
				disabled={loading ||
					killing ||
					selectedProcesses.length === 0 ||
					!snapshot?.supported}
				onclick={() => void killSelected()}
				>{killing ? "Killing…" : "Kill selected"}</button>
		</div>
	</div>
</div>

<style>
	.roblox-process-card {
		width: min(440px, 100%);
	}

	.modal-heading {
		align-items: center;
	}

	.process-heading-actions {
		display: flex;
		gap: var(--space-1);
	}

	.process-results {
		overflow: hidden;
		border: 1px solid var(--color-border);
		border-radius: var(--radius-control);
	}

	.process-table-scroll {
		max-height: 240px;
		overflow: auto;
	}

	table {
		width: 100%;
		border-spacing: 0;
		font-size: var(--text-12);
		text-align: left;
	}

	td {
		padding: var(--space-2);
	}

	.process-name {
		font-family: var(--font-mono);
		font-weight: var(--font-medium);
	}

	tr + tr td {
		border-top: 1px solid var(--color-border-subtle);
	}

	tr.selected td {
		background: var(--color-active);
	}

	.selection-column {
		width: 30px;
	}

	.pid-column {
		color: var(--color-text-faint);
		font-family: var(--font-mono);
		font-size: var(--text-11);
		text-align: right;
		white-space: nowrap;
	}

	.process-empty {
		padding: var(--space-4);
		color: var(--color-text-faint);
		text-align: center;
	}
</style>

<script lang="ts">
	import arrowsClockwiseIcon from "@phosphor-icons/core/regular/arrows-clockwise.svg?raw"
	import checkBoldIcon from "@phosphor-icons/core/bold/check-bold.svg?raw"
	import circleNotchIcon from "@phosphor-icons/core/regular/circle-notch.svg?raw"
	import minusBoldIcon from "@phosphor-icons/core/bold/minus-bold.svg?raw"
	import xIcon from "@phosphor-icons/core/regular/x.svg?raw"
	import Icon from "../shared/Icon.svelte"
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
				error = "Some Roblox processes could not be closed. Try again."
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

<div class="dialog-backdrop" in:modalBackdropIn out:modalBackdropOut>
	<div
		class="dialog-card max-w-md"
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
		<div class="dialog-heading items-center">
			<h2 id="roblox-process-title">Close Roblox processes</h2>
			<div class="flex gap-1">
				<button
					class="btn-ghost-sm"
					type="button"
					aria-label="Refresh process list"
					data-tooltip="Refresh process list"
					disabled={loading || killing || snapshot?.supported === false}
					onclick={() => void refresh()}>
					{#if loading}<Icon
							svg={circleNotchIcon}
							class="spinner"
							size={14}
							aria-hidden="true" />
					{:else}<Icon
							svg={arrowsClockwiseIcon}
							data-turn
							size={14}
							aria-hidden="true" />{/if}
				</button>
				<button
					class="btn-ghost-sm"
					type="button"
					aria-label="Cancel"
					disabled={killing}
					onclick={dismiss}
					><Icon svg={xIcon} size={14} aria-hidden="true" /></button>
			</div>
		</div>
		<div class="overflow-hidden rounded-md border border-line">
			<div
				class="flex min-h-8 items-center justify-between gap-2 border-b border-line-subtle bg-fill px-2 text-xs text-fg-muted">
				<label class="flex cursor-pointer items-center gap-1 font-medium">
					<span class="checkbox">
						<input
							type="checkbox"
							checked={allSelected}
							indeterminate={someSelected}
							disabled={loading || killing || processes.length === 0}
							onchange={toggleAll} />
						<span aria-hidden="true">
							{#if someSelected}<Icon
									svg={minusBoldIcon}
									size={11}
									aria-hidden="true" />
							{:else}<Icon
									svg={checkBoldIcon}
									size={11}
									aria-hidden="true" />{/if}
						</span>
					</span>
					<span>{allSelected ? "Deselect all" : "Select all"}</span>
				</label>
				<span class="font-mono text-fg-faint" role="status"
					>{selectedProcesses.length} of {processes.length} selected</span>
			</div>
			<div class="max-h-60 overflow-auto">
				<table
					class="w-full border-separate border-spacing-0 text-left text-xs"
					aria-label="Roblox processes"
					aria-busy={loading || killing}>
					<tbody>
						{#each processes as process (processKey(process))}
							<tr
								class="*:p-2 not-first:*:border-t not-first:*:border-line-subtle data-selected:*:bg-active"
								data-selected={selected.includes(processKey(process)) ||
									undefined}>
								<td class="w-8">
									<label class="checkbox">
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
											><Icon
												svg={checkBoldIcon}
												size={11}
												aria-hidden="true" /></span>
									</label>
								</td>
								<td class="font-mono font-medium">{process.name}</td>
								<td
									class="text-right font-mono text-2xs whitespace-nowrap text-fg-faint"
									>PID: {process.pid}</td>
							</tr>
						{:else}
							<tr
								><td colspan="3" class="p-4 text-center text-fg-faint">
									<span role="status">
										{#if loading}Loading processes…
										{:else if snapshot?.supported === false}Not
											available on this system.
										{:else if loadError}Process list unavailable.
										{:else}No Roblox processes are running.{/if}
									</span>
								</td></tr>
						{/each}
					</tbody>
				</table>
			</div>
		</div>
		{#if error || loadError}<div class="mt-3 notice-error" role="alert">
				{error || loadError}
			</div>{/if}
		<div class="dialog-actions">
			<button
				class="btn"
				bind:this={cancelButton}
				type="button"
				disabled={killing}
				onclick={dismiss}>Cancel</button>
			<button
				class="btn-danger"
				type="button"
				disabled={loading ||
					killing ||
					selectedProcesses.length === 0 ||
					!snapshot?.supported}
				onclick={() => void killSelected()}
				>{killing ? "Closing…" : "Close selected"}</button>
		</div>
	</div>
</div>

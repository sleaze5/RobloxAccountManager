<script lang="ts">
	import XIcon from "phosphor-svelte/lib/XIcon"
	import CandidateList from "../accounts/CandidateList.svelte"
	import type { CandidateListItem } from "../accounts/candidate-model"
	import type { AccountStore } from "../accounts/account-store.svelte"
	import type { ImportPreview, ImportPreviewItem } from "../backend/bridge"
	import {
		modalBackdropIn,
		modalBackdropOut,
		modalCardIn,
		modalCardOut,
		modalFocus,
		replayValidationNudge,
	} from "../shared/presence"

	let { store, onClose }: { store: AccountStore; onClose: () => void } = $props(),
		cookieInput = $state(""),
		preview = $state<ImportPreview | null>(null),
		selectedIndexes = $state<number[]>([]),
		operation = $state<"validate" | "save" | null>(null),
		form = $state<HTMLFormElement>(),
		selectedCount = $derived(selectedIndexes.length),
		selectedUpdateCount = $derived(
			selectedIndexes.filter((index) =>
				preview?.items?.some(
					(item) => item.index === index && item.updatesExisting,
				),
			).length,
		),
		selectedCreateCount = $derived(selectedCount - selectedUpdateCount),
		allReadySelected = $derived(
			preview !== null &&
				preview.readyCount > 0 &&
				selectedCount === preview.readyCount,
		)
	const candidateItems = $derived<CandidateListItem[]>(
		(preview?.items ?? []).map((item) => ({
			avatarUrl: item.avatarUrl,
			displayName: accountName(item),
			duplicateText: item.sameAccountAsIndex
				? `Same account as cookie #${item.sameAccountAsIndex}`
				: undefined,
			error: item.authenticated
				? undefined
				: `failed to authenticate - ${item.errorStatus}: ${item.errorMessage}`,
			id: String(item.index),
			indexLabel: `${item.index}.`,
			robloxUserId: item.robloxUserId,
			selectable: item.ready,
			selectionLabel: selectionLabel(item),
			status: item.updatesExisting
				? groupIsSelected(item)
					? "Update cookie"
					: "Already added"
				: undefined,
			username: item.username,
		})),
	)

	function close(): void {
		if (store.busy) {
			return
		}

		discardPreview()
		store.clearError()
		cookieInput = ""
		onClose()
	}

	function discardPreview(): void {
		const batchId = preview?.batchId
		preview = null
		selectedIndexes = []
		if (batchId) {
			void store.discardValidatedCookies(batchId)
		}
	}

	function updateInput(value: string): void {
		if (value === cookieInput) {
			return
		}
		discardPreview()
		store.clearError()
		cookieInput = value
	}

	async function validate(): Promise<void> {
		if (store.busy) {
			return
		}

		const oldBatchId = preview?.batchId
		preview = null
		operation = "validate"
		try {
			if (oldBatchId) {
				await store.discardValidatedCookies(oldBatchId)
			}
			preview = await store.validateCookies(cookieInput)
			selectedIndexes = defaultSelectedIndexes(preview)
			if (!preview) {
				replayValidationNudge(form)
			}
		} finally {
			operation = null
		}
	}

	async function addValidated(): Promise<void> {
		if (!preview?.batchId || selectedCount < 1) {
			return
		}

		operation = "save"
		try {
			if (
				await store.saveValidatedCookies(preview.batchId, [...selectedIndexes])
			) {
				preview = null
				selectedIndexes = []
				cookieInput = ""
				onClose()
				return
			}
			preview = null
			selectedIndexes = []
		} finally {
			operation = null
		}
	}

	function accountName(item: ImportPreviewItem): string {
		return item.displayName?.trim() || item.username || "Unknown account"
	}

	function selectionGroup(item: ImportPreviewItem): number {
		return item.sameAccountAsIndex || item.index
	}

	function defaultSelectedIndexes(value: ImportPreview | null): number[] {
		const groups = new Set<number>(),
			indexes: number[] = []
		for (const item of value?.items ?? []) {
			if (!item.ready) {
				continue
			}
			const group = selectionGroup(item)
			if (groups.has(group)) {
				continue
			}
			groups.add(group)
			indexes.push(item.index)
		}
		return indexes
	}

	function groupIsSelected(item: ImportPreviewItem): boolean {
		const group = selectionGroup(item)
		return selectedIndexes.some((index) => {
			const selectedItem = preview?.items?.find(
				(candidate) => candidate.index === index,
			)
			return selectedItem !== undefined && selectionGroup(selectedItem) === group
		})
	}

	function selectionLabel(item: ImportPreviewItem): string {
		if (!item.ready) {
			return `Cookie ${item.index} cannot be selected`
		}
		if (item.sameAccountAsIndex) {
			return `Use cookie ${item.index} instead of cookie ${item.sameAccountAsIndex}`
		}
		if (item.updatesExisting) {
			return `Update @${item.username} with cookie ${item.index}`
		}
		return `Add account from cookie ${item.index}`
	}

	function saveActionLabel(): string {
		if (operation === "save") {
			return "Saving..."
		}
		if (selectedUpdateCount > 0 && selectedCreateCount > 0) {
			return `Apply ${selectedCount} changes`
		}
		if (selectedUpdateCount === 1) {
			return "Update cookie"
		}
		if (selectedUpdateCount > 1) {
			return `Update ${selectedUpdateCount} cookies`
		}
		return `Add ${selectedCount} ${selectedCount === 1 ? "account" : "accounts"}`
	}

	function toggleAccount(index: number, selected: boolean): void {
		const item = preview?.items?.find(
			(candidate) => candidate.index === index && candidate.ready,
		)
		if (!item || !preview) {
			return
		}
		if (!selected) {
			selectedIndexes = selectedIndexes.filter(
				(selectedIndex) => selectedIndex !== index,
			)
			return
		}

		const group = selectionGroup(item),
			items = preview.items ?? []
		selectedIndexes = [
			...selectedIndexes.filter((selectedIndex) => {
				const selectedItem = items.find(
					(candidate) => candidate.index === selectedIndex,
				)
				return (
					selectedItem === undefined || selectionGroup(selectedItem) !== group
				)
			}),
			index,
		].sort((left, right) => left - right)
	}

	function toggleAllAccounts(): void {
		if (!preview) {
			return
		}
		selectedIndexes = allReadySelected ? [] : defaultSelectedIndexes(preview)
	}
</script>

<div class="modal-backdrop" in:modalBackdropIn out:modalBackdropOut>
	<form
		novalidate
		class="modal-card cookie-card"
		bind:this={form}
		use:modalFocus
		in:modalCardIn
		out:modalCardOut
		onsubmit={(event) => {
			event.preventDefault()
			void validate()
		}}>
		<div class="modal-heading">
			<div>
				<h2>Paste cookies</h2>
				<p>
					Paste one .ROBLOSECURITY cookie per line. Each line can contain a
					raw value, cookie pair, or Cookie header.
				</p>
			</div>
			<button
				type="button"
				aria-label="Close add account dialog"
				disabled={store.busy}
				onclick={close}>
				<XIcon size={17} aria-hidden="true" />
			</button>
		</div>
		{#if store.error}
			<div class="modal-error" role="alert">
				{store.error}
			</div>
		{/if}
		<label class="modal-field">
			<span>.ROBLOSECURITY cookies <b>Required</b></span>
			<textarea
				value={cookieInput}
				autocomplete="off"
				spellcheck="false"
				placeholder="Paste one cookie per line"
				disabled={store.busy}
				required
				oninput={(event) => updateInput(event.currentTarget.value)}></textarea>
		</label>
		{#if preview}
			<CandidateList
				items={candidateItems}
				selected={selectedIndexes.map(String)}
				selectionTotal={preview.readyCount}
				disabled={store.busy}
				onToggle={(id, selected) => toggleAccount(Number(id), selected)}
				onToggleAll={toggleAllAccounts} />
		{/if}
		<div class="modal-actions">
			<button type="button" disabled={store.busy} onclick={close}>Cancel</button>
			{#if preview?.batchId && preview.readyCount > 0}
				<button type="submit" disabled={store.busy}>
					{operation === "validate" ? "Validating..." : "Validate"}
				</button>
				<button
					class="primary-action"
					type="button"
					disabled={store.busy || selectedCount === 0}
					onclick={() => void addValidated()}>
					{saveActionLabel()}
				</button>
			{:else}
				<button
					class="primary-action"
					type="submit"
					disabled={store.busy || !cookieInput.trim()}>
					{operation === "validate" ? "Validating..." : "Validate"}
				</button>
			{/if}
		</div>
	</form>
</div>

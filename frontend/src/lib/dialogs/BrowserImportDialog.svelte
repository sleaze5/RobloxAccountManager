<script lang="ts">
	import browserIcon from "@phosphor-icons/core/regular/browser.svg?raw"
	import checkIcon from "@phosphor-icons/core/regular/check.svg?raw"
	import circleNotchIcon from "@phosphor-icons/core/regular/circle-notch.svg?raw"
	import cornersOutIcon from "@phosphor-icons/core/regular/corners-out.svg?raw"
	import plusIcon from "@phosphor-icons/core/regular/plus.svg?raw"
	import warningCircleIcon from "@phosphor-icons/core/regular/warning-circle.svg?raw"
	import xIcon from "@phosphor-icons/core/regular/x.svg?raw"
	import Icon from "../shared/Icon.svelte"
	import Dialog from "../shared/Dialog.svelte"
	import { untrack } from "svelte"
	import CandidateList from "../accounts/CandidateList.svelte"
	import type { AccountStore } from "../accounts/account-store.svelte"
	import type { CandidateListItem } from "../accounts/candidate-model"
	import type { BrowserStore } from "../browser/browser-store.svelte"
	import { Lifecycle, SaveStatus } from "../backend/bridge"
	import type { NotificationCenter } from "../notifications/notification-center.svelte"

	let {
		browser,
		accounts,
		notifications,
		onClose,
	}: {
		browser: BrowserStore
		accounts: AccountStore
		notifications: NotificationCenter
		onClose: () => void
	} = $props()

	const sessionStatus: Record<Lifecycle, string> = {
		[Lifecycle.$zero]: "Opening…",
		[Lifecycle.LifecycleStarting]: "Opening…",
		[Lifecycle.LifecycleActive]: "Waiting for sign-in",
		[Lifecycle.LifecycleClosing]: "Closing…",
		[Lifecycle.LifecycleValidating]: "Checking account…",
		[Lifecycle.LifecycleClosed]: "Closed",
		[Lifecycle.LifecycleFailed]: "Failed",
	}

	let selected = $state<string[]>([]),
		saving = $state(false),
		isClosingBrowsersToSave = $state(false),
		savedCreated = $state(0),
		savedUpdated = $state(0),
		initializedUsers = $state<number[]>([])
	const sessions = $derived(browser.loginSessions),
		activeLoginSessions = $derived(browser.activeLoginSessions),
		candidates = $derived(browser.candidates),
		canClose = $derived(
			!browser.loading && sessions.length === 0 && candidates.length === 0,
		),
		candidateItems = $derived<CandidateListItem[]>(
			candidates.map((candidate, index) => ({
				avatarUrl: candidate.avatarUrl,
				displayName: candidate.displayName || candidate.username,
				duplicateText: candidate.sameAccountAsId
					? "Same account as another browser session"
					: undefined,
				id: candidate.id,
				indexLabel: `${index + 1}.`,
				robloxUserId: candidate.robloxUserId,
				selectable: true,
				selectionLabel: candidate.updatesExisting
					? `Update @${candidate.username}`
					: `Add @${candidate.username}`,
				status: candidate.updatesExisting ? "Update cookie" : undefined,
				username: candidate.username,
			})),
		),
		candidateGroupCount = $derived(
			new Set(candidates.map((candidate) => candidate.robloxUserId)).size,
		),
		saveLabel = $derived(
			saving
				? "Saving…"
				: selected.length === 0
					? "Save accounts"
					: `Save ${selected.length} ${selected.length === 1 ? "account" : "accounts"}`,
		)

	$effect(() => {
		const currentCandidates = candidates
		untrack(() => reconcileCandidateSelection(currentCandidates))
	})

	// Closing a browser checks its final cookie and can add or update candidates,
	// so the save waits until every browser is closed and the selection is reconciled.
	$effect(() => {
		if (!isClosingBrowsersToSave || activeLoginSessions.length > 0) return
		untrack(() => {
			isClosingBrowsersToSave = false
			if (selected.length === 0) {
				saving = false
				return
			}
			void save()
		})
	})

	function reconcileCandidateSelection(currentCandidates: typeof candidates): void {
		const candidatesById = new Map(
			currentCandidates.map((candidate) => [candidate.id, candidate]),
		)
		const availableUsers = new Set(
			currentCandidates.map((candidate) => candidate.robloxUserId),
		)
		const knownUsers = new Set(
			initializedUsers.filter((userId) => availableUsers.has(userId)),
		)
		const selectedUsers = new Set<number>(),
			nextSelected: string[] = []

		for (const id of selected) {
			const candidate = candidatesById.get(id)
			if (!candidate || selectedUsers.has(candidate.robloxUserId)) continue
			selectedUsers.add(candidate.robloxUserId)
			nextSelected.push(id)
		}
		for (const candidate of currentCandidates) {
			if (knownUsers.has(candidate.robloxUserId)) continue
			knownUsers.add(candidate.robloxUserId)
			if (selectedUsers.has(candidate.robloxUserId)) continue
			selectedUsers.add(candidate.robloxUserId)
			nextSelected.push(candidate.id)
		}

		const nextInitializedUsers = [...knownUsers]
		if (!sameItems(selected, nextSelected)) selected = nextSelected
		if (!sameItems(initializedUsers, nextInitializedUsers)) {
			initializedUsers = nextInitializedUsers
		}
	}

	function sameItems<T>(left: T[], right: T[]): boolean {
		return (
			left.length === right.length &&
			left.every((value, index) => value === right[index])
		)
	}

	function toggleCandidate(id: string, checked: boolean): void {
		const candidate = candidates.find((listed) => listed.id === id)
		if (!candidate) return
		if (!checked) {
			selected = selected.filter((selectedId) => selectedId !== id)
			return
		}
		selected = [
			...selected.filter((selectedId) => {
				const selectedCandidate = candidates.find(
					(listed) => listed.id === selectedId,
				)
				return selectedCandidate?.robloxUserId !== candidate.robloxUserId
			}),
			id,
		]
	}

	function toggleAll(): void {
		if (selected.length === candidateGroupCount) {
			selected = []
			return
		}
		const users = new Set<number>()
		selected = candidates
			.filter((candidate) => {
				if (users.has(candidate.robloxUserId)) return false
				users.add(candidate.robloxUserId)
				return true
			})
			.map((candidate) => candidate.id)
	}

	async function closeBrowsersAndSave(): Promise<void> {
		saving = true
		if (activeLoginSessions.length === 0) {
			await save()
			return
		}
		isClosingBrowsersToSave = true
		if (!(await browser.closeAllLogin())) {
			isClosingBrowsersToSave = false
			saving = false
		}
	}

	async function save(): Promise<void> {
		saving = true
		const saveResult = await browser.save(selected).finally(() => (saving = false))
		if (!saveResult) return
		const created = (saveResult.items ?? []).filter(
				(savedItem) => savedItem.status === SaveStatus.SaveCreated,
			).length,
			updated = (saveResult.items ?? []).filter(
				(savedItem) => savedItem.status === SaveStatus.SaveUpdated,
			).length,
			failed = (saveResult.items ?? []).filter(
				(savedItem) => savedItem.status === SaveStatus.SaveFailed,
			).length
		if (created + updated > 0) {
			await accounts.refreshAccounts()
			savedCreated += created
			savedUpdated += updated
		}
		if (failed > 0) {
			browser.error = `${failed} ${failed === 1 ? "account" : "accounts"} could not be saved.`
		}
	}

	function closeDialog(): void {
		if (!canClose) return
		if (savedCreated + savedUpdated > 0) {
			notifications.show({
				message: summary(savedCreated, savedUpdated),
				title: "Accounts saved",
				tone: "success",
			})
		}
		browser.error = ""
		onClose()
	}

	function summary(created: number, updated: number): string {
		if (created > 0 && updated > 0) {
			return `Added ${created} new ${created === 1 ? "account" : "accounts"}, and updated ${updated} saved account ${updated === 1 ? "cookie" : "cookies"}.`
		}
		if (created > 0) {
			return `Added ${created} new ${created === 1 ? "account" : "accounts"}.`
		}
		return `Updated ${updated} saved account ${updated === 1 ? "cookie" : "cookies"}.`
	}
</script>

<svelte:window
	onkeydown={(event) => {
		if (event.key === "Escape") closeDialog()
	}} />

<Dialog class="max-w-lg" aria-labelledby="browser-import-title">
	<div class="dialog-heading">
		<div>
			<h2 id="browser-import-title">Add accounts with a browser</h2>
			<p>Sign in to Roblox, then choose which accounts to save.</p>
		</div>
		<button
			type="button"
			aria-label="Close browser sign-in"
			disabled={!canClose}
			onclick={closeDialog}>
			<Icon svg={xIcon} size={14} aria-hidden="true" />
		</button>
	</div>
	{#if browser.error}<div class="mb-3 notice-error" role="alert">
			{browser.error}
		</div>{/if}
	{#if sessions.length === 0 && candidates.length === 0}
		<div
			class="empty-state rounded-md border border-dashed border-line-strong py-8"
			aria-busy={browser.loading}>
			<Icon svg={browserIcon} size={28} aria-hidden="true" />
			<h2>No browser open</h2>
			<p>Open a browser and sign in to Roblox to add an account.</p>
			<button
				type="button"
				class="mt-1 button-primary"
				disabled={browser.loading}
				onclick={() => void browser.startLogin()}
				>{browser.loading ? "Opening…" : "Open browser"}</button>
		</div>
	{:else}
		<ul class="overflow-hidden rounded-md border border-line bg-fill">
			{#each sessions as session, index (session.id)}
				<li
					class="flex min-h-12 animate-enter items-center gap-2 border-b border-line-subtle px-2 py-1.5">
					<span
						class="grid size-6 shrink-0 place-items-center text-fg-faint"
						aria-hidden="true">
						{#if session.lifecycle === Lifecycle.LifecycleActive}
							<span class="size-2 rounded-full bg-success"></span>
						{:else if session.lifecycle === Lifecycle.LifecycleFailed}
							<Icon
								svg={warningCircleIcon}
								class="text-danger"
								size={16} />
						{:else if session.lifecycle === Lifecycle.LifecycleClosed}
							<Icon svg={checkIcon} size={16} />
						{:else}
							<Icon svg={circleNotchIcon} class="spinner" size={16} />
						{/if}
					</span>
					<div class="min-w-0 flex-1">
						<p class="text-xs font-medium text-fg">
							Browser {index + 1}
						</p>
						<p class="text-2xs text-fg-faint">
							{sessionStatus[session.lifecycle]}
						</p>
						{#if session.checkpointWarning}<p
								class="mt-0.5 text-2xs text-warning">
								{session.checkpointWarning}
							</p>{/if}
						{#if session.error}<p
								class="mt-0.5 text-2xs text-danger-muted"
								role="alert">
								{session.error}
							</p>{/if}
					</div>
					<button
						type="button"
						class="button-ghost-sm"
						aria-label={`Show browser ${index + 1}`}
						data-tooltip="Show browser"
						disabled={session.lifecycle === Lifecycle.LifecycleFailed ||
							session.lifecycle === Lifecycle.LifecycleClosed}
						onclick={() => void browser.focus(session.id)}
						><Icon
							svg={cornersOutIcon}
							size={14}
							aria-hidden="true" /></button>
					<button
						type="button"
						class="button-ghost-sm"
						aria-label={`Close browser ${index + 1}`}
						data-tooltip="Close browser"
						disabled={session.lifecycle === Lifecycle.LifecycleClosing ||
							session.lifecycle === Lifecycle.LifecycleValidating}
						onclick={() => void browser.close(session.id)}
						><Icon svg={xIcon} size={14} aria-hidden="true" /></button>
				</li>
			{/each}
			<li>
				<button
					type="button"
					class="flex h-9 w-full items-center gap-2 px-2 text-xs text-fg-muted transition-colors duration-100 hover:bg-hover hover:text-fg"
					disabled={browser.loading || saving}
					onclick={() => void browser.startLogin()}>
					<span class="grid size-6 place-items-center" aria-hidden="true"
						><Icon svg={plusIcon} size={14} /></span>
					{sessions.length > 0 ? "Open another browser" : "Open browser"}
				</button>
			</li>
		</ul>
		{#if candidates.length > 0}
			<CandidateList
				items={candidateItems}
				{selected}
				selectionTotal={candidateGroupCount}
				disabled={saving}
				onToggle={toggleCandidate}
				onToggleAll={toggleAll} />
			{#if activeLoginSessions.length > 0}<p class="mt-2 text-xs text-fg-faint">
					Saving closes {activeLoginSessions.length === 1
						? "the browser"
						: "all browsers"}.
				</p>{/if}
		{:else}
			<p
				class="mt-3 rounded-md border border-dashed border-line-strong px-3 py-6 text-center text-xs text-fg-faint">
				Accounts appear here after you sign in.
			</p>
		{/if}
	{/if}
	<div class="dialog-actions">
		{#if activeLoginSessions.length > 1}<button
				type="button"
				class="mr-auto button"
				disabled={saving}
				onclick={() => void browser.closeAllLogin()}>Close all browsers</button
			>{/if}
		{#if candidates.length > 0 && activeLoginSessions.length === 0}<button
				class="button"
				type="button"
				disabled={saving}
				onclick={() => void browser.discard()}>Discard</button
			>{/if}
		{#if canClose}<button class="button" type="button" onclick={closeDialog}
				>Close</button
			>{/if}
		{#if candidates.length > 0}<button
				type="button"
				class="button-primary"
				disabled={selected.length === 0 || saving}
				onclick={() => void closeBrowsersAndSave()}>{saveLabel}</button
			>{/if}
	</div>
</Dialog>

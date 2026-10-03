<script lang="ts">
	import ChevronDown from "@lucide/svelte/icons/chevron-down"
	import ExternalLink from "@lucide/svelte/icons/external-link"
	import Focus from "@lucide/svelte/icons/focus"
	import Trash2 from "@lucide/svelte/icons/trash-2"
	import X from "@lucide/svelte/icons/x"
	import { onMount, untrack } from "svelte"
	import CandidateList from "../accounts/CandidateList.svelte"
	import type { AccountStore } from "../accounts/account-store.svelte"
	import type { CandidateListItem } from "../accounts/candidate-model"
	import type { BrowserStore } from "../browser/browser-store.svelte"
	import { Lifecycle, SaveStatus } from "../backend/bridge"
	import type { NotificationCenter } from "../notifications/notification-center.svelte"
	import {
		modalBackdropIn,
		modalBackdropOut,
		modalCardIn,
		modalCardOut,
		modalFocus,
	} from "../shared/presence"

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

	let selected = $state<string[]>([]),
		sessionsExpanded = $state(true),
		savedCreated = $state(0),
		savedUpdated = $state(0),
		initializedUsers = $state<number[]>([]),
		launchingInitial = $state(true)
	const sessions = $derived(browser.loginSessions),
		activeLoginSessions = $derived(browser.activeLoginSessions),
		candidates = $derived(browser.candidates),
		canClose = $derived(
			!launchingInitial &&
				!browser.loading &&
				sessions.length === 0 &&
				candidates.length === 0,
		),
		canCommit = $derived(activeLoginSessions.length === 0 && candidates.length > 0),
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
		)

	onMount(() => {
		if (activeLoginSessions.length > 0) {
			launchingInitial = false
			return
		}
		void browser.startLogin().finally(() => (launchingInitial = false))
	})

	$effect(() => {
		const currentCandidates = candidates
		untrack(() => reconcileCandidateSelection(currentCandidates))
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
		const candidate = candidates.find((item) => item.id === id)
		if (!candidate) return
		if (!checked) {
			selected = selected.filter((selectedId) => selectedId !== id)
			return
		}
		selected = [
			...selected.filter((selectedId) => {
				const selectedCandidate = candidates.find(
					(item) => item.id === selectedId,
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

	async function save(): Promise<void> {
		const result = await browser.save(selected)
		if (!result) return
		const created = (result.items ?? []).filter(
				(item) => item.status === SaveStatus.SaveCreated,
			).length,
			updated = (result.items ?? []).filter(
				(item) => item.status === SaveStatus.SaveUpdated,
			).length,
			failed = (result.items ?? []).filter(
				(item) => item.status === SaveStatus.SaveFailed,
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

<div class="modal-backdrop" in:modalBackdropIn out:modalBackdropOut>
	<div
		class="modal-card browser-import-card"
		use:modalFocus
		in:modalCardIn
		out:modalCardOut
		role="dialog"
		aria-modal="true"
		aria-labelledby="browser-import-title">
		<div class="modal-heading">
			<div>
				<h2 id="browser-import-title">Add accounts with a browser</h2>
				<p>
					Sign in normally. Account cookies stay in the backend until you
					choose what to save.
				</p>
			</div>
		</div>
		{#if browser.error}<div class="modal-error" role="alert">
				{browser.error}
			</div>{/if}
		<div class="browser-session-toolbar">
			<button
				type="button"
				class="settings-row-action"
				disabled={browser.loading}
				onclick={() => void browser.startLogin()}
				><ExternalLink size={14} />Open another browser</button>
			{#if activeLoginSessions.length > 0}<button
					type="button"
					class="danger-button"
					onclick={() => void browser.closeAllLogin()}
					><Trash2 size={14} />Close all</button
				>{/if}
		</div>
		<button
			class="browser-session-summary"
			type="button"
			aria-expanded={sessionsExpanded}
			onclick={() => (sessionsExpanded = !sessionsExpanded)}>
			<span
				>{activeLoginSessions.length} active browser {activeLoginSessions.length ===
				1
					? "session"
					: "sessions"}</span
			><ChevronDown class={sessionsExpanded ? "open" : undefined} size={14} />
		</button>
		{#if sessionsExpanded && sessions.length > 0}
			<div class="browser-session-list">
				{#each sessions as session (session.id)}
					<div class="browser-session-row">
						<span
							><strong>{session.title}</strong><small
								>{session.lifecycle}</small
							></span>
						<div>
							<button
								type="button"
								aria-label="Focus browser"
								disabled={session.lifecycle ===
									Lifecycle.LifecycleFailed}
								onclick={() => void browser.focus(session.id)}
								><Focus size={13} />Focus</button
							><button
								type="button"
								class="danger-action"
								aria-label="Close browser"
								disabled={session.lifecycle ===
									Lifecycle.LifecycleClosing ||
									session.lifecycle === Lifecycle.LifecycleValidating}
								onclick={() => void browser.close(session.id)}
								><X size={13} />Close</button>
						</div>
					</div>
					{#if session.checkpointWarning}<div class="modal-warning">
							{session.checkpointWarning}
						</div>{/if}
					{#if session.error}<div class="modal-warning">
							{session.error}
						</div>{/if}
				{/each}
			</div>
		{/if}
		{#if candidates.length > 0}
			<CandidateList
				items={candidateItems}
				{selected}
				selectionTotal={candidateGroupCount}
				disabled={browser.loading}
				onToggle={toggleCandidate}
				onToggleAll={toggleAll} />
		{:else if launchingInitial}
			<div class="modal-note">Starting browser...</div>
		{:else if activeLoginSessions.length === 0}
			<div class="modal-note">No account candidates are waiting to be saved.</div>
		{:else}
			<div class="modal-note">
				Account checks appear here after a browser signs in.
			</div>
		{/if}
		<div class="modal-actions">
			{#if candidates.length > 0 && activeLoginSessions.length === 0}<button
					type="button"
					onclick={() => void browser.discard()}>Discard</button
				>{/if}
			{#if canClose}<button type="button" onclick={closeDialog}>Close</button
				>{/if}
			{#if candidates.length > 0}<button
					type="button"
					class="primary-action"
					disabled={!canCommit || selected.length === 0 || browser.loading}
					onclick={() => void save()}
					>{browser.loading ? "Saving..." : "Save selected"}</button
				>{/if}
		</div>
	</div>
</div>

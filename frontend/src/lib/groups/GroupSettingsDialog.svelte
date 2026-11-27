<script lang="ts">
	import { notifyErrors } from "../notifications/error-notifications.svelte"
	import warningIcon from "@phosphor-icons/core/regular/warning.svg?raw"
	import xIcon from "@phosphor-icons/core/regular/x.svg?raw"
	import { onMount } from "svelte"
	import { accountBackend } from "../backend/bridge"
	import type { GroupSettings } from "../backend/bridge"
	import { getNotificationCenter } from "../notifications/notification-center.svelte"
	import type { CancellableRequest } from "../shared/cancellable-request"
	import ConfirmationDialog from "../shared/ConfirmationDialog.svelte"
	import Dialog from "../shared/Dialog.svelte"
	import { errorMessage } from "../shared/errors"
	import Icon from "../shared/Icon.svelte"
	import SettingChangeList from "../shared/SettingChangeList.svelte"
	import type { SettingChange } from "../shared/SettingChangeList.svelte"
	import SettingsRowsSkeleton from "../shared/SettingsRowsSkeleton.svelte"
	import { replayValidationNudge } from "../shared/transitions"
	import type { GroupsStore } from "./groups-store.svelte"

	let {
		store,
		groupId,
		groupName,
		onClose,
	}: {
		store: GroupsStore
		groupId: number
		groupName: string
		onClose: () => void
	} = $props()

	type SwitchKey = Exclude<keyof GroupSettings, "isMemberListVisible">

	const notifications = getNotificationCenter(),
		dialogId = $props.id(),
		switches: { key: SwitchKey; label: string; description: string }[] = [
			{
				key: "isApprovalRequired",
				label: "Require join requests",
				description:
					"Users send a join request. A member with the permission accepts or declines it.",
			},
			{
				key: "areFundsVisible",
				label: "Show funds",
				description: "Anyone can see the Robux of the group.",
			},
			{
				key: "areGamesVisible",
				label: "Show games",
				description: "Anyone can see the games of the group on its page.",
			},
			{
				key: "areEnemiesAllowed",
				label: "Allow enemies",
				description: "Other groups can declare this group an enemy.",
			},
		]

	let saved = $state.raw<GroupSettings | null>(null),
		draft = $state<GroupSettings | null>(null),
		isLoading = $state(true),
		isReviewing = $state(false),
		loadError = $state(""),
		card = $state<HTMLDivElement>()
	let request: CancellableRequest<GroupSettings> | null = null

	const isSaving = $derived(store.pendingAction === "settings"),
		onOff = (value: boolean) => (value ? "On" : "Off"),
		changes = $derived.by<SettingChange[]>(() => {
			if (!saved || !draft) return []
			const listed: SettingChange[] = switches
				.filter(({ key }) => saved?.[key] !== draft?.[key])
				.map(({ key, label }) => ({
					label,
					previous: onOff(!!saved?.[key]),
					next: onOff(!!draft?.[key]),
				}))
			if (
				saved.isMemberListVisible != null &&
				draft.isMemberListVisible != null &&
				saved.isMemberListVisible !== draft.isMemberListVisible
			) {
				listed.push({
					label: "Show members",
					previous: onOff(saved.isMemberListVisible),
					next: onOff(draft.isMemberListVisible),
				})
			}
			return listed
		})

	onMount(() => {
		void load()
		return () => void request?.cancel()
	})

	async function load(): Promise<void> {
		void request?.cancel()
		const pendingRequest = accountBackend.GetGroupSettings(store.accountId, groupId)
		request = pendingRequest
		isLoading = true
		loadError = ""
		try {
			const loaded = await pendingRequest
			if (request !== pendingRequest) return
			saved = loaded
			draft = { ...loaded }
		} catch (error) {
			if (request === pendingRequest)
				loadError = errorMessage(
					error,
					"Could not load the group settings. Try again.",
				)
		} finally {
			if (request === pendingRequest) isLoading = false
		}
	}

	function close(): void {
		if (!isSaving) onClose()
	}

	function review(): void {
		if (changes.length === 0) {
			replayValidationNudge(card)
			return
		}
		store.clearActionError()
		isReviewing = true
	}

	async function save(): Promise<void> {
		if (!draft) return
		if (!(await store.saveSettings({ ...draft }))) return
		notifications.show({
			id: `group-settings-${groupId}`,
			title: "Group settings saved",
			tone: "success",
			message: `On Roblox as @${store.account.username}`,
		})
		onClose()
	}

	notifyErrors("Group settings not loaded", () => loadError)
</script>

<svelte:window
	onkeydown={(event) => {
		if (event.key === "Escape" && !isReviewing) close()
	}} />

{#snippet switchRow(
	id: string,
	label: string,
	description: string,
	checked: boolean,
	onChange: (checked: boolean) => void,
)}
	<div class="settings-row">
		<div class="settings-row-copy">
			<strong id={`${dialogId}-${id}`}>{label}</strong>
			<span id={`${dialogId}-${id}-description`}>{description}</span>
		</div>
		<label class="switch">
			<input
				type="checkbox"
				role="switch"
				aria-labelledby={`${dialogId}-${id}`}
				aria-describedby={`${dialogId}-${id}-description`}
				{checked}
				onchange={(event) => onChange(event.currentTarget.checked)} />
			<span aria-hidden="true"></span>
		</label>
	</div>
{/snippet}

{#if isReviewing}
	<ConfirmationDialog
		title="Save group settings on Roblox?"
		description={`Roblox applies ${changes.length === 1 ? "this change" : `these ${changes.length} changes`} to ${groupName} for every member.`}
		confirmLabel="Save on Roblox"
		progressLabel="Saving…"
		tone="primary"
		account={store.account}
		isWorking={isSaving}
		onConfirm={() => void save()}
		onCancel={() => (isReviewing = false)}>
		<SettingChangeList {changes} />
	</ConfirmationDialog>
{:else}
	<Dialog
		class="max-w-md text-left"
		aria-labelledby={`${dialogId}-title`}
		bind:card
		onBackdropClick={close}>
		<form
			novalidate
			onsubmit={(event) => {
				event.preventDefault()
				review()
			}}>
			<div class="dialog-heading mb-2">
				<div class="min-w-0">
					<h2 id={`${dialogId}-title`}>Manage group</h2>
					<p>Changes save on Roblox after you review them.</p>
				</div>
				<button type="button" aria-label="Close dialog" onclick={close}>
					<Icon svg={xIcon} size={14} aria-hidden="true" />
				</button>
			</div>

			{#if isLoading && !draft}
				<SettingsRowsSkeleton
					rowCount={4}
					hasIcon={false}
					label="Loading group settings…" />
			{:else if !draft}
				<div class="empty-state py-8">
					<Icon svg={warningIcon} size={28} aria-hidden="true" />
					<h2>Group settings unavailable</h2>
					<button class="button" type="button" onclick={() => void load()}
						>Try again</button>
				</div>
			{:else}
				{@const settings = draft}
				<div class="settings-rows rounded-lg border border-line">
					{#each switches as setting (setting.key)}
						{@render switchRow(
							setting.key,
							setting.label,
							setting.description,
							settings[setting.key],
							(checked) => (settings[setting.key] = checked),
						)}
					{/each}
					{#if settings.isMemberListVisible != null}
						{@render switchRow(
							"isMemberListVisible",
							"Show members",
							"Anyone can see the member list.",
							settings.isMemberListVisible,
							(checked) => (settings.isMemberListVisible = checked),
						)}
					{/if}
				</div>
			{/if}

			<div class="dialog-actions">
				<button class="button" type="button" onclick={close}>Cancel</button>
				<button
					class="button-primary"
					type="submit"
					disabled={!draft || changes.length === 0}>Review changes</button>
			</div>
		</form>
	</Dialog>
{/if}

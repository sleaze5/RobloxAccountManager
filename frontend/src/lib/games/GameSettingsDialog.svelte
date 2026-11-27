<script lang="ts">
	import { notifyErrors } from "../notifications/error-notifications.svelte"
	import warningIcon from "@phosphor-icons/core/regular/warning.svg?raw"
	import xIcon from "@phosphor-icons/core/regular/x.svg?raw"
	import { onMount } from "svelte"
	import { accountBackend } from "../backend/bridge"
	import type { GameSettings, GameSettingsChange } from "../backend/bridge"
	import type { Account } from "../accounts/account-model"
	import type { NotificationCenter } from "../notifications/notification-center.svelte"
	import type { CancellableRequest } from "../shared/cancellable-request"
	import ConfirmationDialog from "../shared/ConfirmationDialog.svelte"
	import Dialog from "../shared/Dialog.svelte"
	import { errorMessage } from "../shared/errors"
	import Icon from "../shared/Icon.svelte"
	import SettingChangeList from "../shared/SettingChangeList.svelte"
	import type { SettingChange } from "../shared/SettingChangeList.svelte"
	import SettingsRowsSkeleton from "../shared/SettingsRowsSkeleton.svelte"
	import { replayValidationNudge } from "../shared/transitions"

	let {
		account,
		universeId,
		description,
		notifications,
		onSaved,
		onClose,
	}: {
		account: Account
		universeId: number
		// description is the description that the game details show, because
		// the settings API of Roblox does not return it.
		description: string
		notifications: NotificationCenter
		onSaved: () => void
		onClose: () => void
	} = $props()

	const maxNameLength = 50,
		maxDescriptionLength = 1000,
		dialogId = $props.id()

	let settings = $state.raw<GameSettings | null>(null),
		loadError = $state(""),
		saveError = $state(""),
		isLoading = $state(true),
		isReviewing = $state(false),
		isSaving = $state(false),
		name = $state(""),
		typedDescription = $state(""),
		isPublic = $state(false),
		arePrivateServersAllowed = $state(false),
		card = $state<HTMLDivElement>()
	let request: CancellableRequest<GameSettings> | null = null

	const onOff = (value: boolean) => (value ? "On" : "Off"),
		changedOrNull = <Value>(next: Value, previous: Value): Value | null =>
			next === previous ? null : next,
		change = $derived<GameSettingsChange>({
			name: settings ? changedOrNull(name.trim(), settings.name) : null,
			description: changedOrNull(typedDescription.trim(), description.trim()),
			isPublic: settings ? changedOrNull(isPublic, settings.isPublic) : null,
			arePrivateServersAllowed: settings
				? changedOrNull(
						arePrivateServersAllowed,
						settings.arePrivateServersAllowed,
					)
				: null,
		}),
		changes = $derived.by<SettingChange[]>(() => {
			if (!settings) return []
			const listed: SettingChange[] = [],
				characterCount = (text: string) =>
					text
						? `${text.length} ${text.length === 1 ? "character" : "characters"}`
						: "None"
			if (change.name !== null)
				listed.push({
					label: "Name",
					previous: settings.name,
					next: change.name,
				})
			if (change.description !== null)
				listed.push({
					label: "Description",
					previous: characterCount(description.trim()),
					next: characterCount(change.description),
				})
			if (change.isPublic !== null)
				listed.push({
					label: "Public",
					previous: onOff(settings.isPublic),
					next: onOff(change.isPublic),
				})
			if (change.arePrivateServersAllowed !== null)
				listed.push({
					label: "Private servers",
					previous: onOff(settings.arePrivateServersAllowed),
					next: onOff(change.arePrivateServersAllowed),
				})
			return listed
		}),
		isNameValid = $derived(name.trim().length > 0)

	onMount(() => {
		void load()
		return () => void request?.cancel()
	})

	async function load(): Promise<void> {
		void request?.cancel()
		const pendingRequest = accountBackend.GetGameSettings(account.id, universeId)
		request = pendingRequest
		isLoading = true
		loadError = ""
		try {
			const loaded = await pendingRequest
			if (request !== pendingRequest) return
			settings = loaded
			name = loaded.name
			typedDescription = description
			isPublic = loaded.isPublic
			arePrivateServersAllowed = loaded.arePrivateServersAllowed
		} catch (error) {
			if (request === pendingRequest)
				loadError = errorMessage(
					error,
					"Could not load the game settings. Try again.",
				)
		} finally {
			if (request === pendingRequest) isLoading = false
		}
	}

	function close(): void {
		if (!isSaving) onClose()
	}

	function review(): void {
		if (changes.length === 0 || !isNameValid) {
			replayValidationNudge(card)
			return
		}
		saveError = ""
		isReviewing = true
	}

	async function save(): Promise<void> {
		isSaving = true
		saveError = ""
		try {
			await accountBackend.SaveGameSettings(account.id, universeId, change)
			notifications.show({
				id: `game-settings-${universeId}`,
				title: "Game settings saved",
				tone: "success",
				message: `On Roblox as @${account.username}`,
			})
			onSaved()
			onClose()
		} catch (error) {
			saveError = errorMessage(
				error,
				"Roblox did not confirm the change. Refresh to check.",
			)
			isSaving = false
		}
	}

	notifyErrors("Game settings not loaded", () => loadError)
	notifyErrors("Game settings not saved", () => saveError)
</script>

<svelte:window
	onkeydown={(event) => {
		if (event.key === "Escape" && !isReviewing) close()
	}} />

{#if isReviewing}
	<ConfirmationDialog
		title="Save game settings on Roblox?"
		description={`Roblox applies ${changes.length === 1 ? "this change" : `these ${changes.length} changes`} for every player.`}
		confirmLabel="Save on Roblox"
		progressLabel="Saving…"
		tone="primary"
		{account}
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
					<h2 id={`${dialogId}-title`}>Manage game</h2>
					<p>Changes save on Roblox after you review them.</p>
				</div>
				<button type="button" aria-label="Close dialog" onclick={close}>
					<Icon svg={xIcon} size={14} aria-hidden="true" />
				</button>
			</div>

			{#if isLoading && !settings}
				<SettingsRowsSkeleton
					rowCount={4}
					hasIcon={false}
					label="Loading game settings…" />
			{:else if !settings}
				<div class="empty-state py-8">
					<Icon svg={warningIcon} size={28} aria-hidden="true" />
					<h2>Game settings unavailable</h2>
					<button class="button" type="button" onclick={() => void load()}
						>Try again</button>
				</div>
			{:else}
				<label class="dialog-field">
					<span>Name</span>
					<input
						class="field"
						type="text"
						autocomplete="off"
						maxlength={maxNameLength}
						aria-invalid={!isNameValid || undefined}
						bind:value={name} />
				</label>
				{#if !isNameValid}
					<p class="mt-1 text-xs text-danger-muted" role="alert">
						Enter a name.
					</p>
				{/if}
				<label class="dialog-field">
					<span class="flex items-center justify-between gap-2"
						>Description<span class="font-mono text-2xs"
							>{typedDescription.length}/{maxDescriptionLength}</span
						></span>
					<textarea
						class="h-28 resize-none"
						maxlength={maxDescriptionLength}
						bind:value={typedDescription}></textarea>
				</label>
				<div class="mt-3 settings-rows rounded-lg border border-line">
					<div class="settings-row">
						<div class="settings-row-copy">
							<strong id={`${dialogId}-public`}>Public</strong>
							<span id={`${dialogId}-public-description`}
								>Anyone can find and play the game.</span>
						</div>
						<label class="switch">
							<input
								type="checkbox"
								role="switch"
								aria-labelledby={`${dialogId}-public`}
								aria-describedby={`${dialogId}-public-description`}
								bind:checked={isPublic} />
							<span aria-hidden="true"></span>
						</label>
					</div>
					<div class="settings-row">
						<div class="settings-row-copy">
							<strong id={`${dialogId}-private-servers`}
								>Private servers</strong>
							<span id={`${dialogId}-private-servers-description`}
								>Players can buy private servers at the price set on
								Roblox.</span>
						</div>
						<label class="switch">
							<input
								type="checkbox"
								role="switch"
								aria-labelledby={`${dialogId}-private-servers`}
								aria-describedby={`${dialogId}-private-servers-description`}
								bind:checked={arePrivateServersAllowed} />
							<span aria-hidden="true"></span>
						</label>
					</div>
				</div>
			{/if}

			<div class="dialog-actions">
				<button class="button" type="button" onclick={close}>Cancel</button>
				<button
					class="button-primary"
					type="submit"
					disabled={!settings || changes.length === 0 || !isNameValid}
					>Review changes</button>
			</div>
		</form>
	</Dialog>
{/if}

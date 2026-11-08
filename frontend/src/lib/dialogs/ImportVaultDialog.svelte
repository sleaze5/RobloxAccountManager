<script lang="ts">
	import CheckboxMark from "../shared/CheckboxMark.svelte"
	import checkCircleIcon from "@phosphor-icons/core/regular/check-circle.svg?raw"
	import eyeIcon from "@phosphor-icons/core/regular/eye.svg?raw"
	import eyeSlashIcon from "@phosphor-icons/core/regular/eye-slash.svg?raw"
	import stackIcon from "@phosphor-icons/core/regular/stack.svg?raw"
	import warningCircleIcon from "@phosphor-icons/core/regular/warning-circle.svg?raw"
	import fileArrowDownIcon from "@phosphor-icons/core/regular/file-arrow-down.svg?raw"
	import xIcon from "@phosphor-icons/core/regular/x.svg?raw"
	import Icon from "../shared/Icon.svelte"
	import CandidateList from "../accounts/CandidateList.svelte"
	import type { CandidateListItem } from "../accounts/candidate-model"
	import Timestamp from "../shared/Timestamp.svelte"
	import { untrack } from "svelte"
	import {
		accountBackend,
		SettingsSection,
		TransferMode,
		TransferSettingsStatus,
		type ExportDetails,
		type TransferPreview,
		type TransferResult,
	} from "../backend/bridge"
	import { replayValidationNudge } from "../shared/presence"
	import { errorMessage } from "../shared/errors"
	import Dialog from "../shared/Dialog.svelte"
	import type { AccountStore } from "../accounts/account-store.svelte"
	import { getGamesStore } from "../games/games-store.svelte"
	import type { NotificationCenter } from "../notifications/notification-center.svelte"
	import { chooseExportFile, fileName } from "./vault-transfer"
	import { appSettings } from "../settings/settings-store.svelte"

	let {
		store,
		notifications,
		path,
		onClose,
	}: {
		store: AccountStore
		notifications: NotificationCenter
		path: string
		onClose: () => void
	} = $props()
	const games = getGamesStore()

	const settingsSections = [
			{ value: SettingsSection.SectionPresence, label: "Presence updates" },
			{
				value: SettingsSection.SectionRoblox,
				label: "Roblox multi-instance and Linux client",
			},
			{ value: SettingsSection.SectionIntegrations, label: "Integrations" },
			{ value: SettingsSection.SectionUserInterface, label: "User interface" },
			{ value: SettingsSection.SectionLogging, label: "Logging" },
			{ value: SettingsSection.SectionVault, label: "Password test reminder" },
		],
		settingsLabels: Record<TransferSettingsStatus, { text: string; tone: string }> =
			{
				[TransferSettingsStatus.$zero]: {
					text: "Skipped",
					tone: "text-fg-faint",
				},
				[TransferSettingsStatus.TransferSettingsSkipped]: {
					text: "Skipped",
					tone: "text-fg-faint",
				},
				[TransferSettingsStatus.TransferSettingsImported]: {
					text: "Imported",
					tone: "text-fg",
				},
				[TransferSettingsStatus.TransferSettingsPending]: {
					text: "Saves on next unlock",
					tone: "text-warning",
				},
			},
		fileGroup = $props.id()
	let file = $state(untrack(() => path)),
		password = $state(""),
		details = $state<ExportDetails | null>(null),
		visible = $state(false),
		preview = $state<TransferPreview | null>(null),
		mode = $state(TransferMode.TransferOverwrite),
		selectedAccounts = $state<number[]>([]),
		favoritePlaces = $state(true),
		savedPrivateServers = $state(true),
		sections = $state<SettingsSection[]>([]),
		confirmation = $state(""),
		importResult = $state<TransferResult | null>(null),
		opening = $state(false),
		importing = $state(false),
		error = $state(""),
		card = $state<HTMLDivElement>()

	const busy = $derived(opening || importing),
		readable = $derived(details?.supported === true),
		name = $derived(fileName(file)),
		folder = $derived(file.slice(0, Math.max(file.length - name.length - 1, 0))),
		accounts = $derived(preview?.accounts ?? []),
		allAccountsSelected = $derived(
			accounts.length > 0 && selectedAccounts.length === accounts.length,
		),
		mergeEmpty = $derived(
			selectedAccounts.length === 0 &&
				!(favoritePlaces && (preview?.newFavoritePlaces ?? 0) > 0) &&
				!(savedPrivateServers && (preview?.newSavedPrivateServers ?? 0) > 0) &&
				sections.length === 0,
		),
		resultRows = $derived(
			importResult
				? [
						{ label: "Accounts added", count: importResult.added },
						{ label: "Accounts updated", count: importResult.updated },
						{
							label: "Favorite places added",
							count: importResult.favoritePlaces,
						},
						{
							label: "Private server links added",
							count: importResult.savedPrivateServers,
						},
					].map((row) => ({
						label: row.label,
						changed: row.count > 0,
						value: row.count > 0 ? String(row.count) : "None",
					}))
				: [],
		),
		keptCookies = $derived(importResult?.keptCookies ?? []),
		candidateItems = $derived<CandidateListItem[]>(
			accounts.map((account) => ({
				avatarUrl: account.avatarUrl,
				displayName: account.displayName.trim() || account.username,
				id: String(account.robloxUserId),
				robloxUserId: account.robloxUserId,
				selectable: true,
				selectionLabel: account.existing
					? `Update @${account.username} from the export`
					: `Add @${account.username} from the export`,
				status: account.existing
					? selectedAccounts.includes(account.robloxUserId)
						? "Update cookie"
						: "Already added"
					: undefined,
				username: account.username,
			})),
		)

	$effect(() => {
		void loadDetails(file)
	})

	async function loadDetails(target: string): Promise<void> {
		details = null
		try {
			const value = await accountBackend.GetExportDetails(target)
			if (file === target) details = value
		} catch (cause) {
			if (file === target)
				error = errorMessage(cause, "The export file could not be read.")
		}
	}

	async function changeFile(): Promise<void> {
		const next = await chooseExportFile(notifications)
		if (!next) return
		file = next
		error = ""
	}

	async function openExport(): Promise<void> {
		opening = true
		error = ""
		try {
			preview = await accountBackend.OpenVaultExport(file, password)
			password = ""
			selectedAccounts = (preview.accounts ?? []).map(
				(account) => account.robloxUserId,
			)
			favoritePlaces = preview.newFavoritePlaces > 0
			savedPrivateServers = preview.newSavedPrivateServers > 0
		} catch (cause) {
			error = errorMessage(cause, "The export could not be opened. Try again.")
			replayValidationNudge(card)
		} finally {
			opening = false
		}
	}

	async function importExport(): Promise<void> {
		if (!preview) return
		importing = true
		error = ""
		try {
			const overwrite = mode === TransferMode.TransferOverwrite
			importResult = await accountBackend.ImportVaultExport(
				preview.id,
				{
					mode,
					robloxUserIds: overwrite ? [] : selectedAccounts,
					favoritePlaces: !overwrite && favoritePlaces,
					savedPrivateServers: !overwrite && savedPrivateServers,
					settingsSections: overwrite ? [] : sections,
				},
				overwrite ? confirmation : "",
			)
			preview = null
			void store.initialize()
			games.reset()
			games.initialize()
			void appSettings.initialize(notifications)
		} catch (cause) {
			error = errorMessage(cause, "The export could not be imported. Try again.")
			preview = null
			replayValidationNudge(card)
		} finally {
			importing = false
		}
	}

	function toggleAccount(robloxUserId: number, selected: boolean): void {
		selectedAccounts = selected
			? [...selectedAccounts, robloxUserId]
			: selectedAccounts.filter((id) => id !== robloxUserId)
	}

	function toggleAllAccounts(): void {
		selectedAccounts = allAccountsSelected
			? []
			: accounts.map((account) => account.robloxUserId)
	}

	function close(): void {
		if (busy) return
		if (preview) void accountBackend.DiscardVaultExport(preview.id)
		onClose()
	}

	function back(): void {
		if (preview) void accountBackend.DiscardVaultExport(preview.id)
		preview = null
		error = confirmation = ""
	}
</script>

<svelte:window
	onkeydown={(event) => {
		if (event.key === "Escape") {
			event.preventDefault()
			close()
		}
	}} />

<Dialog
	class="max-w-lg"
	aria-labelledby="import-vault-title"
	aria-busy={busy}
	bind:card
	onBackdropClick={close}>
	<div class="dialog-heading">
		<div>
			<h2 id="import-vault-title">
				{importResult ? "Data imported" : "Import vault data"}
			</h2>
			<p>
				{#if importResult}
					Imported from {name}.
				{:else if preview}
					Exported <Timestamp value={preview.exportedAtMs} />.
				{:else}
					Enter the export password that was set when this file was exported.
				{/if}
			</p>
		</div>
		<button
			type="button"
			aria-label="Close import dialog"
			disabled={busy}
			onclick={close}>
			<Icon svg={xIcon} size={14} aria-hidden="true" />
		</button>
	</div>

	{#if importResult}
		<div
			class="animate-enter overflow-hidden rounded-md border border-line bg-fill text-xs"
			role="status">
			{#each resultRows as row (row.label)}
				<div
					class="flex min-h-8 items-center justify-between gap-3 px-3 not-first:border-t not-first:border-line-subtle">
					<span class={row.changed ? "text-fg" : "text-fg-faint"}
						>{row.label}</span>
					<span
						class={["font-mono", row.changed ? "text-fg" : "text-fg-faint"]}
						>{row.value}</span>
				</div>
			{/each}
			<div
				class="flex min-h-8 items-center justify-between gap-3 border-t border-line-subtle px-3">
				<span
					class={importResult.settings ===
					TransferSettingsStatus.TransferSettingsSkipped
						? "text-fg-faint"
						: "text-fg"}>Settings</span>
				<span class={settingsLabels[importResult.settings].tone}
					>{settingsLabels[importResult.settings].text}</span>
			</div>
		</div>
		{#if keptCookies.length > 0}
			<div class="mt-3 notice-warning">
				<p>These accounts kept their current cookie:</p>
				<ul class="mt-1 flex flex-col gap-0.5">
					{#each keptCookies as kept (kept.username)}
						<li><strong>{kept.username}</strong>: {kept.reason}</li>
					{/each}
				</ul>
			</div>
		{/if}
		<div class="dialog-actions">
			<button class="button-primary" type="button" onclick={onClose}>Done</button>
		</div>
	{:else if preview}
		<form
			novalidate
			onsubmit={(event) => {
				event.preventDefault()
				void importExport()
			}}>
			<div class="segmented" role="radiogroup" aria-label="Import mode">
				<label>
					<input
						type="radio"
						name="{fileGroup}-mode"
						bind:group={mode}
						value={TransferMode.TransferOverwrite}
						disabled={busy} />Overwrite
				</label>
				<label>
					<input
						type="radio"
						name="{fileGroup}-mode"
						bind:group={mode}
						value={TransferMode.TransferMerge}
						disabled={busy} />Merge
				</label>
			</div>

			{#if mode === TransferMode.TransferMerge}
				<div class="animate-enter">
					<p class="mt-3 text-xs text-fg-muted">
						Accounts already in the vault take the imported cookie if Roblox
						accepts it. Otherwise they keep their current cookie.
					</p>
					{#if accounts.length === 0}
						<p class="mt-3 text-xs text-fg-faint">
							The export has no accounts.
						</p>
					{:else}
						<CandidateList
							items={candidateItems}
							selected={selectedAccounts.map(String)}
							disabled={busy}
							onToggle={(id, selected) =>
								toggleAccount(Number(id), selected)}
							onToggleAll={toggleAllAccounts} />
					{/if}
					<fieldset class="mt-3 flex flex-col gap-1.5">
						<legend class="mb-1 text-xs font-semibold text-fg-muted"
							>Games</legend>
						<label class="check-option">
							<span class="checkbox">
								<input
									type="checkbox"
									bind:checked={favoritePlaces}
									disabled={busy ||
										preview.newFavoritePlaces === 0} />
								<CheckboxMark />
							</span>
							<span>
								<strong>Favorite places</strong>
								<small
									>{preview.newFavoritePlaces} of {preview.favoritePlaces}
									are new.</small>
							</span>
						</label>
						<label class="check-option">
							<span class="checkbox">
								<input
									type="checkbox"
									bind:checked={savedPrivateServers}
									disabled={busy ||
										preview.newSavedPrivateServers === 0} />
								<CheckboxMark />
							</span>
							<span>
								<strong>Private server links</strong>
								<small
									>{preview.newSavedPrivateServers} of {preview.savedPrivateServers}
									are new.</small>
							</span>
						</label>
					</fieldset>
					<fieldset class="mt-3 flex flex-col gap-1.5">
						<legend class="mb-1 text-xs font-semibold text-fg-muted"
							>Settings</legend>
						<p class="text-2xs text-fg-muted">
							Selected settings replace your current ones.
						</p>
						{#each settingsSections as section (section.value)}
							<label class="check-option">
								<span class="checkbox">
									<input
										type="checkbox"
										bind:group={sections}
										value={section.value}
										disabled={busy} />
									<CheckboxMark />
								</span>
								<span><strong>{section.label}</strong></span>
							</label>
						{/each}
					</fieldset>
				</div>
			{:else}
				<div class="animate-enter">
					<div class="mt-3 notice-warning">
						Overwrite removes every account, tag, and favorite place in this
						vault, then imports everything from the file, including
						settings. The app backs up the current vault first.
					</div>
					<label class="dialog-field">
						<span
							><span
								>Type <strong class="font-semibold text-fg"
									>CONFIRM</strong> to overwrite this vault</span
							></span>
						<input
							bind:value={confirmation}
							autocomplete="off"
							disabled={busy} />
					</label>
				</div>
			{/if}

			{#if error}
				<div class="mt-3 notice-error" role="alert">{error}</div>
			{/if}
			<div class="dialog-actions">
				<button class="button" type="button" disabled={busy} onclick={back}
					>Back</button>
				{#if mode === TransferMode.TransferOverwrite}
					<button
						class="button-danger"
						type="submit"
						disabled={confirmation !== "CONFIRM" || busy}
						>{importing ? "Overwriting…" : "Overwrite vault"}</button>
				{:else}
					<button
						class="button-primary"
						type="submit"
						disabled={mergeEmpty || busy}
						>{importing ? "Importing…" : "Import data"}</button>
				{/if}
			</div>
		</form>
	{:else}
		<form
			novalidate
			onsubmit={(event) => {
				event.preventDefault()
				void openExport()
			}}>
			<div
				class="flex items-center gap-2 rounded-md border border-line bg-fill p-2">
				<span class="text-fg-faint">
					<Icon svg={fileArrowDownIcon} size={16} aria-hidden="true" />
				</span>
				<span class="flex min-w-0 flex-1 flex-col">
					<span class="font-mono text-xs wrap-anywhere text-fg">{name}</span>
					<span class="font-mono text-2xs wrap-anywhere text-fg-faint"
						>{folder}</span>
				</span>
				<button
					class="button"
					type="button"
					disabled={busy}
					onclick={() => void changeFile()}>Change file</button>
			</div>
			{#if details}
				<dl class="mt-3 facts" aria-label="Export file format">
					<div data-tone={details.supported ? "success" : "warning"}>
						<dt>
							<Icon
								svg={details.supported
									? checkCircleIcon
									: warningCircleIcon}
								size={16}
								aria-hidden="true" />File format
						</dt>
						<dd>
							<span>Version {details.formatVersion}</span>
							<small
								>{details.supported
									? "Supported"
									: "Not supported"}</small>
						</dd>
					</div>
					<div>
						<dt>
							<Icon
								svg={stackIcon}
								size={16}
								aria-hidden="true" />Supported formats
						</dt>
						<dd>
							<span
								>{details.minFormatVersion === details.maxFormatVersion
									? `Version ${details.maxFormatVersion}`
									: `Versions ${details.minFormatVersion}–${details.maxFormatVersion}`}</span>
						</dd>
					</div>
				</dl>
				{#if !details.supported}
					<div class="mt-3 notice-error" role="alert">
						This export was made by a newer version of the app. Update the
						app to import it.
					</div>
				{/if}
			{/if}
			<label class="dialog-field">
				<span>Export password</span>
				<input
					bind:value={password}
					type={visible ? "text" : "password"}
					autocomplete="off"
					disabled={opening || !readable}
					required />
			</label>
			<button
				class="mt-2 flex items-center gap-1 py-1 text-xs text-fg-muted transition-colors duration-100 *:animate-pop"
				type="button"
				aria-pressed={visible}
				disabled={!readable}
				onclick={() => (visible = !visible)}>
				{#if visible}<Icon svg={eyeSlashIcon} size={16} aria-hidden="true" /> Hide
					password{:else}<Icon svg={eyeIcon} size={16} aria-hidden="true" /> Show
					password{/if}
			</button>
			{#if details?.hint}
				<div class="mt-3 notice-note">Hint: {details.hint}</div>
			{/if}
			{#if error}
				<div class="mt-3 notice-error" role="alert">{error}</div>
			{/if}
			<div class="dialog-actions">
				<button class="button" type="button" disabled={busy} onclick={close}
					>Cancel</button>
				<button
					class="button-primary"
					type="submit"
					disabled={!password || busy || !readable}
					>{opening ? "Opening…" : "Open export"}</button>
			</div>
		</form>
	{/if}
</Dialog>

<script lang="ts">
	import arrowsClockwiseIcon from "@phosphor-icons/core/regular/arrows-clockwise.svg?raw"
	import checkBoldIcon from "@phosphor-icons/core/bold/check-bold.svg?raw"
	import copyIcon from "@phosphor-icons/core/regular/copy.svg?raw"
	import fileArrowDownIcon from "@phosphor-icons/core/regular/file-arrow-down.svg?raw"
	import xIcon from "@phosphor-icons/core/regular/x.svg?raw"
	import Icon from "../shared/Icon.svelte"
	import CandidateList from "../accounts/CandidateList.svelte"
	import type { CandidateListItem } from "../accounts/candidate-model"
	import Timestamp from "../shared/Timestamp.svelte"
	import { onMount } from "svelte"
	import {
		accountBackend,
		SettingsSection,
		TransferMode,
		TransferSettingsStatus,
		type ExportInfo,
		type TransferPreview,
		type TransferResult,
	} from "../backend/bridge"
	import {
		modalBackdropIn,
		modalBackdropOut,
		modalCardIn,
		modalCardOut,
		modalFocus,
		replayValidationNudge,
	} from "../shared/presence"
	import { errorMessage } from "../shared/errors"
	import type { AccountStore } from "../accounts/account-store.svelte"
	import { getGamesStore } from "../games/games-store.svelte"
	import type { NotificationCenter } from "../notifications/notification-center.svelte"
	import { copyExportFolder } from "../settings/export-folder"
	import { appSettings } from "../settings/settings-store.svelte"

	let {
		store,
		notifications,
		onClose,
	}: {
		store: AccountStore
		notifications: NotificationCenter
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
	let files = $state<ExportInfo[] | null>(null),
		directory = $state(""),
		selectedFile = $state(""),
		passcode = $state(""),
		preview = $state<TransferPreview | null>(null),
		mode = $state(TransferMode.TransferMerge),
		selectedAccounts = $state<number[]>([]),
		favoritePlaces = $state(true),
		sections = $state<SettingsSection[]>([]),
		confirmation = $state(""),
		result = $state<TransferResult | null>(null),
		loading = $state(false),
		opening = $state(false),
		importing = $state(false),
		error = $state(""),
		card = $state<HTMLDivElement>()

	const busy = $derived(loading || opening || importing),
		accounts = $derived(preview?.accounts ?? []),
		allAccountsSelected = $derived(
			accounts.length > 0 && selectedAccounts.length === accounts.length,
		),
		mergeEmpty = $derived(
			selectedAccounts.length === 0 &&
				!(favoritePlaces && (preview?.newFavoritePlaces ?? 0) > 0) &&
				sections.length === 0,
		),
		resultRows = $derived(
			result
				? [
						{ label: "Accounts added", count: result.added },
						{ label: "Accounts updated", count: result.updated },
						{
							label: "Favorite places added",
							count: result.favoritePlaces,
						},
					].map((row) => ({
						label: row.label,
						changed: row.count > 0,
						value: row.count > 0 ? String(row.count) : "None",
					}))
				: [],
		),
		keptCookies = $derived(result?.keptCookies ?? []),
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

	onMount(() => {
		void loadFiles()
	})

	async function loadFiles(): Promise<void> {
		loading = true
		error = ""
		try {
			const exports = await accountBackend.ListVaultExports()
			directory = exports.directory
			files = exports.files ?? []
			if (!files.some((file) => file.name === selectedFile)) {
				selectedFile = files[0]?.name ?? ""
			}
		} catch (cause) {
			files ??= []
			error = errorMessage(cause, "The export folder could not be read.")
		} finally {
			loading = false
		}
	}

	async function openExport(): Promise<void> {
		opening = true
		error = ""
		try {
			preview = await accountBackend.OpenVaultExport(selectedFile, passcode)
			passcode = ""
			selectedAccounts = (preview.accounts ?? []).map(
				(account) => account.robloxUserId,
			)
			favoritePlaces = preview.newFavoritePlaces > 0
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
			const replace = mode === TransferMode.TransferReplace
			result = await accountBackend.ImportVaultExport(
				preview.id,
				{
					mode,
					robloxUserIds: replace ? [] : selectedAccounts,
					favoritePlaces: !replace && favoritePlaces,
					settingsSections: replace ? [] : sections,
				},
				replace ? confirmation : "",
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

<div
	class="dialog-backdrop"
	in:modalBackdropIn
	out:modalBackdropOut
	onclick={(event) => {
		if (event.target === event.currentTarget) close()
	}}
	role="presentation">
	<div
		class="dialog-card max-w-lg"
		role="dialog"
		aria-modal="true"
		aria-labelledby="import-vault-title"
		aria-busy={busy}
		bind:this={card}
		use:modalFocus
		in:modalCardIn
		out:modalCardOut>
		<div class="dialog-heading">
			<div>
				<h2 id="import-vault-title">
					{result ? "Data imported" : "Import vault data"}
				</h2>
				<p>
					{#if result}
						Imported from {selectedFile}.
					{:else if preview}
						Exported <Timestamp value={preview.exportedAtMs} />.
					{:else}
						Choose an export file from the export folder.
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

		{#if result}
			<div
				class="animate-enter overflow-hidden rounded-md border border-line bg-fill text-xs"
				role="status">
				{#each resultRows as row (row.label)}
					<div
						class="flex min-h-8 items-center justify-between gap-3 px-3 not-first:border-t not-first:border-line-subtle">
						<span class={row.changed ? "text-fg" : "text-fg-faint"}
							>{row.label}</span>
						<span
							class={[
								"font-mono",
								row.changed ? "text-fg" : "text-fg-faint",
							]}>{row.value}</span>
					</div>
				{/each}
				<div
					class="flex min-h-8 items-center justify-between gap-3 border-t border-line-subtle px-3">
					<span
						class={result.settings ===
						TransferSettingsStatus.TransferSettingsSkipped
							? "text-fg-faint"
							: "text-fg"}>Settings</span>
					<span class={settingsLabels[result.settings].tone}
						>{settingsLabels[result.settings].text}</span>
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
				<button class="btn-primary" type="button" onclick={onClose}
					>Done</button>
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
							value={TransferMode.TransferMerge}
							disabled={busy} />Merge
					</label>
					<label>
						<input
							type="radio"
							name="{fileGroup}-mode"
							bind:group={mode}
							value={TransferMode.TransferReplace}
							disabled={busy} />Replace
					</label>
				</div>

				{#if mode === TransferMode.TransferMerge}
					<div class="animate-enter">
						<p class="mt-3 text-xs text-fg-muted">
							Accounts already in the vault take the imported cookie if
							Roblox accepts it. Otherwise they keep their current cookie.
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
									<span aria-hidden="true"
										><Icon
											svg={checkBoldIcon}
											size={11}
											aria-hidden="true" /></span>
								</span>
								<span>
									<strong>Favorite places</strong>
									<small
										>{preview.newFavoritePlaces} of {preview.favoritePlaces}
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
										<span aria-hidden="true"
											><Icon
												svg={checkBoldIcon}
												size={11}
												aria-hidden="true" /></span>
									</span>
									<span><strong>{section.label}</strong></span>
								</label>
							{/each}
						</fieldset>
					</div>
				{:else}
					<div class="animate-enter">
						<div class="mt-3 notice-warning">
							Replace removes every account, tag, and favorite place in
							this vault, then imports everything from the file, including
							settings. The app backs up the current vault first.
						</div>
						<label class="dialog-field">
							<span>Type REPLACE to confirm</span>
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
					<button class="btn" type="button" disabled={busy} onclick={back}
						>Back</button>
					{#if mode === TransferMode.TransferReplace}
						<button
							class="btn-danger"
							type="submit"
							disabled={confirmation !== "REPLACE" || busy}
							>{importing ? "Replacing…" : "Replace vault"}</button>
					{:else}
						<button
							class="btn-primary"
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
				{#if files === null}
					<p class="text-xs text-fg-muted" aria-busy="true">
						Loading export files…
					</p>
				{:else if files.length === 0}
					<div class="empty-state">
						<Icon svg={fileArrowDownIcon} size={28} aria-hidden="true" />
						<h2>No export files</h2>
						<p>
							Copy a .ramexport file into the export folder, then refresh.
						</p>
					</div>
				{:else}
					<div
						class="flex max-h-48 scrollbar-stable flex-col gap-0.5 overflow-y-auto pl-1"
						role="radiogroup"
						aria-label="Export file">
						{#each files as file (file.name)}
							<label
								class="radio-option"
								data-disabled={busy || undefined}>
								<span class="radio">
									<input
										type="radio"
										name={fileGroup}
										bind:group={selectedFile}
										value={file.name}
										disabled={busy} />
									<span aria-hidden="true"></span>
								</span>
								<span class="flex min-w-0 flex-col py-1">
									<span
										class="font-mono text-2xs wrap-anywhere text-fg"
										>{file.name}</span>
									<small class="text-2xs text-fg-muted"
										><Timestamp value={file.createdAtMs} /></small>
								</span>
							</label>
						{/each}
					</div>
				{/if}
				<div class="mt-3 flex items-center gap-2">
					<span
						class="min-w-0 flex-1 font-mono text-2xs wrap-anywhere text-fg-faint"
						>{directory}</span>
					<button
						class="btn-icon"
						type="button"
						aria-label="Copy folder path"
						data-tooltip="Copy folder path"
						disabled={!directory}
						onclick={() => void copyExportFolder(directory, notifications)}>
						<Icon svg={copyIcon} size={16} aria-hidden="true" />
					</button>
					<button
						class="btn-icon"
						type="button"
						aria-label="Refresh export files"
						data-tooltip="Refresh export files"
						disabled={busy}
						onclick={() => void loadFiles()}>
						<Icon
							svg={arrowsClockwiseIcon}
							size={16}
							aria-hidden="true"
							data-turn />
					</button>
				</div>
				<label class="dialog-field">
					<span>Passcode</span>
					<input
						bind:value={passcode}
						type="password"
						autocomplete="off"
						disabled={opening || files?.length === 0}
						required />
				</label>
				{#if error}
					<div class="mt-3 notice-error" role="alert">{error}</div>
				{/if}
				<div class="dialog-actions">
					<button class="btn" type="button" disabled={busy} onclick={close}
						>Cancel</button>
					<button
						class="btn-primary"
						type="submit"
						disabled={!selectedFile || !passcode || busy}
						>{opening ? "Opening…" : "Open export"}</button>
				</div>
			</form>
		{/if}
	</div>
</div>

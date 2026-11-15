<script lang="ts">
	import broomIcon from "@phosphor-icons/core/regular/broom.svg?raw"
	import trashIcon from "@phosphor-icons/core/regular/trash.svg?raw"
	import Icon from "../shared/Icon.svelte"
	import { onMount } from "svelte"
	import {
		accountBackend,
		RobloxCleanScope,
		type RobloxClientState,
	} from "../backend/bridge"
	import RobloxCleanDialog from "../dialogs/RobloxCleanDialog.svelte"
	import type { NotificationCenter } from "../notifications/notification-center.svelte"
	import Select from "../shared/Select.svelte"
	import { linuxClientOptions } from "./linux-clients"
	import { robloxClientOptions as options } from "./roblox-client-options.svelte"
	import SettingsStatus from "./SettingsStatus.svelte"
	import { errorMessage } from "../shared/errors"

	let { notifications }: { notifications: NotificationCenter } = $props()
	let clients = $state<RobloxClientState | null>(null),
		clientsLoading = $state(true),
		clientsSaving = $state(false),
		clientsError = $state(""),
		cleanScope = $state<RobloxCleanScope | null>(null)
	const snapshot = $derived(options.multiInstance),
		cleaner = $derived(options.cleaner),
		clientOptions = $derived(clients ? linuxClientOptions(clients) : []),
		installedClients = $derived(
			(clients?.clients ?? []).filter((client) => client.installed).length,
		)

	async function loadClients(): Promise<void> {
		clientsLoading = clients === null
		try {
			clients = await accountBackend.GetRobloxClients()
			clientsError = ""
		} catch {
			clientsError = "The Linux client list could not be loaded. Try again."
		} finally {
			clientsLoading = false
		}
	}

	async function setClient(client: string): Promise<void> {
		if (clientsSaving || !clients?.choiceSupported || client === clients.selected)
			return
		const previous = clients
		clientsSaving = true
		clientsError = ""
		try {
			clients = await accountBackend.SetLinuxClient(client)
		} catch (cause) {
			clients = previous
			clientsError = errorMessage(
				cause,
				"The Linux client could not be saved. Try again.",
			)
		} finally {
			clientsSaving = false
		}
	}

	onMount(() => {
		void options.load()
		void loadClients()
		const interval = setInterval(() => {
			if (snapshot?.supported && snapshot.enabled)
				void options.refreshMultiInstance()
		}, 500)
		return () => clearInterval(interval)
	})
</script>

<section class="settings-section" aria-labelledby="multi-instance-title">
	<div class="settings-rows">
		<div class="settings-row">
			<div class="settings-row-copy">
				<strong
					id="multi-instance-title"
					class={[snapshot?.supported === false && "text-fg-faint"]}
					>Multi-instance</strong>
				<span id="multi-instance-description"
					>Launch multiple games at the same time. Only works while Roblox
					Account Manager is open.</span>
				<SettingsStatus
					id="multi-instance-status"
					value={options.multiInstanceStatus.text}
					tone={options.multiInstanceStatus.tone} />
			</div>
			<label class="switch">
				<input
					type="checkbox"
					role="switch"
					aria-labelledby="multi-instance-title"
					aria-describedby="multi-instance-description multi-instance-status"
					checked={snapshot?.enabled ?? false}
					disabled={options.isLoadingMultiInstance ||
						options.isSavingMultiInstance ||
						!snapshot?.supported}
					onchange={(event) => {
						const enabled = event.currentTarget.checked
						event.currentTarget.checked = snapshot?.enabled ?? false
						void options.setMultiInstanceEnabled(enabled)
					}} />
				<span aria-hidden="true"></span>
			</label>
		</div>
	</div>
	{#if options.multiInstanceError || options.multiInstanceStatusError}
		<div class="notice-error" role="alert">
			{options.multiInstanceError || options.multiInstanceStatusError}
		</div>
		{#if !snapshot}
			<div class="flex justify-end gap-2">
				<button
					class="button"
					type="button"
					disabled={options.isLoadingMultiInstance}
					onclick={() => void options.refreshMultiInstance()}>Retry</button>
			</div>
		{/if}
	{/if}
</section>

{#snippet settingSwitch(
	id: string,
	isChecked: boolean,
	isDisabled: boolean,
	onToggle: (enabled: boolean) => void,
)}
	<label class="switch">
		<input
			type="checkbox"
			role="switch"
			aria-labelledby={`${id}-title`}
			aria-describedby={`${id}-description ${id}-status`}
			checked={isChecked}
			disabled={isDisabled}
			onchange={(event) => {
				const enabled = event.currentTarget.checked
				event.currentTarget.checked = isChecked
				onToggle(enabled)
			}} />
		<span aria-hidden="true"></span>
	</label>
{/snippet}

{#if cleaner?.supported}
	<section class="settings-section" aria-labelledby="roblox-cleaning-title">
		<h3 id="roblox-cleaning-title">Cleaning</h3>
		<div class="settings-rows">
			<div class="settings-row">
				<div class="settings-row-copy">
					<strong id="clean-before-launch-title">Clean before launch</strong>
					<span id="clean-before-launch-description"
						>Delete Roblox caches, cookies, and sign-in data before each
						launch, so the next account starts without traces of the
						previous one. Roblox, its settings, and Studio plugins stay
						installed, and only one account can launch at a time.</span>
					{#if cleaner.isBlockedByMultiInstance}
						<SettingsStatus
							id="clean-before-launch-status"
							value="Paused while multi-instance is on" />
					{/if}
				</div>
				{@render settingSwitch(
					"clean-before-launch",
					cleaner.cleanBeforeLaunch,
					options.isSavingCleanBeforeLaunch ||
						cleaner.isBlockedByMultiInstance,
					(enabled) => void options.setCleanBeforeLaunch(enabled),
				)}
			</div>
			<div
				class="settings-suboptions"
				role="group"
				aria-label="Options for every Roblox cleaning">
				<div class="settings-row">
					<div class="settings-row-copy">
						<strong id="delete-logs-title"
							>Delete logs when cleaning</strong>
						<span id="delete-logs-description"
							>Also delete Roblox logs, which name the accounts and games
							you played. Logs Explorer loses the games it lists.
							Uninstalling always deletes them.</span>
					</div>
					{@render settingSwitch(
						"delete-logs",
						cleaner.deleteLogsWhenCleaning,
						options.isSavingDeleteLogs,
						(enabled) => void options.setDeleteLogsWhenCleaning(enabled),
					)}
				</div>
				{#if cleaner.elevationSupported}
					<div class="settings-row">
						<div class="settings-row-copy">
							<strong id="administrator-rights-title"
								>Use administrator rights</strong>
							<span id="administrator-rights-description"
								>Ask Windows once per session when Roblox is closed or
								cleaned, so files, registry entries, and processes
								shared by all Windows users are included. Only a small
								helper gets the rights.</span>
							{#if cleaner.isElevated}
								<SettingsStatus
									id="administrator-rights-status"
									value="Already running as administrator"
									tone="success" />
							{/if}
						</div>
						{@render settingSwitch(
							"administrator-rights",
							cleaner.useAdministratorRights,
							options.isSavingAdministratorRights,
							(enabled) =>
								void options.setUseAdministratorRights(enabled),
						)}
					</div>
				{/if}
			</div>
			<div class="settings-row">
				<div class="settings-row-copy">
					<strong id="clean-now-title">Clean Roblox data</strong>
					<span id="clean-now-description"
						>Close every Roblox process, then delete caches, cookies, and
						sign-in data from this computer. Roblox stays installed and
						keeps its settings.</span>
				</div>
				<button
					class="button-warning"
					type="button"
					aria-haspopup="dialog"
					aria-describedby="clean-now-description"
					onclick={() => (cleanScope = RobloxCleanScope.ScopeData)}>
					<Icon svg={broomIcon} size={14} aria-hidden="true" />
					Clean now
				</button>
			</div>
			<div class="settings-row">
				<div class="settings-row-copy">
					<strong id="uninstall-title">Uninstall Roblox</strong>
					<span id="uninstall-description"
						>Close every Roblox process, then delete Roblox Player, Roblox
						Studio, their settings, shortcuts, file associations, registry
						entries, and the Roblox folders in Documents, Pictures, and
						Videos. Bootstrappers such as Bloxstrap keep their files but
						must register Roblox links again.</span>
				</div>
				<button
					class="button-danger"
					type="button"
					aria-haspopup="dialog"
					aria-describedby="uninstall-description"
					onclick={() => (cleanScope = RobloxCleanScope.ScopeUninstall)}>
					<Icon svg={trashIcon} size={14} aria-hidden="true" />
					Uninstall
				</button>
			</div>
		</div>
		{#if options.cleanerError}
			<div class="notice-error" role="alert">{options.cleanerError}</div>
		{/if}
	</section>
{:else if options.cleanerError}
	<div class="notice-error" role="alert">{options.cleanerError}</div>
{/if}

{#if cleanScope}
	<RobloxCleanDialog
		scope={cleanScope}
		{notifications}
		onClose={() => (cleanScope = null)} />
{/if}

{#if clients?.choiceSupported}
	<section class="settings-section" aria-labelledby="linux-client-title">
		<div class="settings-rows">
			<div class="settings-row">
				<div class="settings-row-copy">
					<strong id="linux-client-title">Linux client</strong>
					<span id="linux-client-description"
						>Join games with Sober or Mocktail. Automatic uses the installed
						client, or the desktop default when both are installed.</span>
					{#if !clientsLoading && installedClients === 0}
						<SettingsStatus value="Not installed" tone="warning" />
					{/if}
				</div>
				<Select
					label="Linux client"
					value={clients?.selected ?? ""}
					options={clientOptions}
					disabled={clientsLoading || clientsSaving || installedClients === 0}
					onChange={(client) => void setClient(client)} />
			</div>
		</div>
		{#if clientsError}
			<div class="notice-error" role="alert">{clientsError}</div>
		{/if}
	</section>
{:else if clientsError}
	<div class="notice-error" role="alert">{clientsError}</div>
	<div class="flex justify-end gap-2 px-1 pt-2">
		<button
			class="button"
			type="button"
			disabled={clientsLoading}
			onclick={() => void loadClients()}>Retry</button>
	</div>
{/if}

<script lang="ts">
	import { notifyErrors } from "../notifications/error-notifications.svelte"
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
	import SettingSwitch from "./SettingSwitch.svelte"
	import { errorMessage } from "../shared/errors"

	let { notifications }: { notifications: NotificationCenter } = $props()
	let clients = $state<RobloxClientState | null>(null),
		clientsLoading = $state(true),
		clientsSaving = $state(false),
		clientsError = $state(""),
		cleanScope = $state<RobloxCleanScope | null>(null)
	const snapshot = $derived(options.multiClient),
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
			clientsError = "Could not load the Linux client list. Try again."
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
				"Could not save the Linux client. Try again.",
			)
		} finally {
			clientsSaving = false
		}
	}

	onMount(() => {
		void options.load()
		void loadClients()
		return options.pollMultiClientWhileEnabled()
	})

	notifyErrors("Roblox Cleaner not run", () => options.cleanerError)
	notifyErrors(
		"Multi-client not changed",
		() => options.multiClientError || options.multiClientStatusError,
	)
	notifyErrors("Roblox clients not loaded", () => clientsError)
</script>

<section class="settings-section" aria-labelledby="multi-client-title">
	<div class="settings-rows">
		<div class="settings-row">
			<div class="settings-row-copy">
				<strong
					id="multi-client-title"
					class={[snapshot?.supported === false && "text-fg-faint"]}
					>Multi-client</strong>
				<span id="multi-client-description"
					>Launch several games at once. Works only while Roblox Account
					Manager is open.</span>
				<SettingsStatus
					id="multi-client-status"
					value={options.multiClientStatus.text}
					tone={options.multiClientStatus.tone} />
			</div>
			<SettingSwitch
				labelledBy="multi-client-title"
				describedBy="multi-client-description multi-client-status"
				isChecked={snapshot?.enabled ?? false}
				isDisabled={options.isLoadingMultiClient ||
					options.isSavingMultiClient ||
					!snapshot?.supported}
				onToggle={(enabled) => void options.setMultiClientEnabled(enabled)} />
		</div>
	</div>
	{#if options.multiClientStatusError && !snapshot}
		<div class="flex justify-end gap-2">
			<button
				class="button"
				type="button"
				disabled={options.isLoadingMultiClient}
				onclick={() => void options.refreshMultiClient()}>Try again</button>
		</div>
	{/if}
</section>

{#if cleaner?.supported}
	<section class="settings-section" aria-labelledby="roblox-cleaning-title">
		<h3 id="roblox-cleaning-title">Cleaning</h3>
		<div class="settings-rows">
			<div class="settings-row">
				<div class="settings-row-copy">
					<strong id="clean-before-launch-title">Clean before launch</strong>
					<span id="clean-before-launch-description"
						>Clear caches and sign-in data before each launch. Launches one
						account at a time.</span>
					{#if cleaner.isBlockedByMultiClient}
						<SettingsStatus
							id="clean-before-launch-status"
							value="Paused while Multi-client is on" />
					{/if}
				</div>
				<SettingSwitch
					labelledBy="clean-before-launch-title"
					describedBy={cleaner.isBlockedByMultiClient
						? "clean-before-launch-description clean-before-launch-status"
						: "clean-before-launch-description"}
					isChecked={cleaner.cleanBeforeLaunch}
					isDisabled={options.isSavingCleanBeforeLaunch ||
						cleaner.isBlockedByMultiClient}
					onToggle={(enabled) =>
						void options.setCleanBeforeLaunch(enabled)} />
			</div>
			<div class="settings-suboptions" role="group" aria-label="Cleaning options">
				<div class="settings-row">
					<div class="settings-row-copy">
						<strong id="delete-logs-title"
							>Delete logs when cleaning</strong>
						<span id="delete-logs-description"
							>Also clears Logs Explorer history.</span>
					</div>
					<SettingSwitch
						labelledBy="delete-logs-title"
						describedBy="delete-logs-description"
						isChecked={cleaner.deleteLogsWhenCleaning}
						isDisabled={options.isSavingDeleteLogs}
						onToggle={(enabled) =>
							void options.setDeleteLogsWhenCleaning(enabled)} />
				</div>
				{#if cleaner.elevationSupported}
					<div class="settings-row">
						<div class="settings-row-copy">
							<strong id="administrator-rights-title"
								>Use administrator rights</strong>
							<span id="administrator-rights-description"
								>Ask Windows once per session, to also clean shared
								files and processes.</span>
							{#if cleaner.isElevated}
								<SettingsStatus
									id="administrator-rights-status"
									value="Running as administrator"
									tone="success" />
							{/if}
						</div>
						<SettingSwitch
							labelledBy="administrator-rights-title"
							describedBy={cleaner.isElevated
								? "administrator-rights-description administrator-rights-status"
								: "administrator-rights-description"}
							isChecked={cleaner.useAdministratorRights}
							isDisabled={options.isSavingAdministratorRights}
							onToggle={(enabled) =>
								void options.setUseAdministratorRights(enabled)} />
					</div>
				{/if}
			</div>
			<div class="settings-row">
				<div class="settings-row-copy">
					<strong id="clean-now-title">Clean Roblox data</strong>
					<span id="clean-now-description"
						>Close Roblox and delete its caches and sign-in data.</span>
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
						>Remove Roblox, Studio, and their files, including saved places,
						screenshots, and videos.</span>
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
	</section>
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
	</section>
{:else if clientsError}
	<div class="flex justify-end gap-2 px-1 pt-2">
		<button
			class="button"
			type="button"
			disabled={clientsLoading}
			onclick={() => void loadClients()}>Try again</button>
	</div>
{/if}

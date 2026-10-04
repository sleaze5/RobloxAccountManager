<script lang="ts">
	import { onMount } from "svelte"
	import {
		accountBackend,
		type MultiInstanceSnapshot,
		type RobloxClientState,
	} from "../backend/bridge"
	import Select from "../shared/Select.svelte"
	import SettingsStatus from "./SettingsStatus.svelte"

	let snapshot = $state<MultiInstanceSnapshot | null>(null),
		loading = $state(true),
		saving = $state(false),
		error = $state(""),
		statusError = $state(""),
		clients = $state<RobloxClientState | null>(null),
		clientsLoading = $state(true),
		clientsSaving = $state(false),
		clientsError = $state("")
	const clientOptions = $derived.by(() => {
			const state = clients
			if (!state) return []
			const options = [
				{
					value: "",
					label: "Automatic",
					description:
						"Use the installed client, or the desktop default when both are installed.",
				},
			]
			for (const client of state.clients ?? []) {
				if (!client.installed && client.id !== state.selected) continue
				options.push({
					value: client.id,
					label: client.installed
						? client.name
						: `${client.name} (not installed)`,
					description: "",
				})
			}
			if (
				state.selected &&
				!options.some((option) => option.value === state.selected)
			) {
				options.push({
					value: state.selected,
					label: "Selected client (not installed)",
					description: "",
				})
			}
			return options
		}),
		installedClients = $derived(
			(clients?.clients ?? []).filter((client) => client.installed).length,
		)
	const statusText = $derived.by(() => {
			if (loading) return "Checking…"
			if (saving) return "Saving…"
			if (!snapshot) return "Not checked"
			if (!snapshot.supported) return "Unavailable - Windows only"
			if (!snapshot.enabled) return "Off"
			if (snapshot.ready) return "Ready"
			return `Not ready - ${snapshot.message}`
		}),
		statusTone = $derived(
			saving || loading || !snapshot?.enabled
				? "neutral"
				: snapshot.ready
					? "success"
					: "warning",
		)
	let checking = false,
		disposed = false,
		revision = 0

	async function load(): Promise<void> {
		if (checking || saving || disposed) return
		checking = true
		loading = snapshot === null
		const requestRevision = revision
		try {
			const state = await accountBackend.GetMultiInstanceState()
			if (disposed || requestRevision !== revision) return
			snapshot = state
			statusError = ""
		} catch {
			if (disposed || requestRevision !== revision) return
			statusError = "The multi-instance status could not be checked. Try again."
		} finally {
			checking = false
			if (!disposed && requestRevision === revision) loading = false
		}
	}

	async function setEnabled(enabled: boolean): Promise<void> {
		if (saving || loading || !snapshot?.supported) return
		const previous = snapshot
		revision++
		saving = true
		error = ""
		try {
			snapshot = await accountBackend.SetMultiInstanceEnabled(enabled)
			statusError = ""
		} catch (cause) {
			snapshot = previous
			error =
				cause instanceof Error
					? cause.message
					: "The multi-instance setting could not be saved. Try again."
		} finally {
			saving = false
		}
	}

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
			clientsError =
				cause instanceof Error
					? cause.message
					: "The Linux client could not be saved. Try again."
		} finally {
			clientsSaving = false
		}
	}

	onMount(() => {
		void load()
		void loadClients()
		const interval = setInterval(() => {
			if (snapshot?.supported && snapshot.enabled) void load()
		}, 500)
		return () => {
			disposed = true
			clearInterval(interval)
		}
	})
</script>

<section class="settings-section" aria-labelledby="multi-instance-title">
	<div class="settings-rows">
		<div class="settings-row" class:unavailable={snapshot?.supported === false}>
			<div class="settings-row-copy">
				<strong id="multi-instance-title">Multi-instance</strong>
				<span id="multi-instance-description"
					>Launch multiple games at the same time. Only works while Roblox
					Account Manager is open.</span>
				<SettingsStatus
					id="multi-instance-status"
					value={statusText}
					tone={statusTone} />
			</div>
			<label class="settings-toggle">
				<input
					type="checkbox"
					role="switch"
					aria-labelledby="multi-instance-title"
					aria-describedby="multi-instance-description multi-instance-status"
					checked={snapshot?.enabled ?? false}
					disabled={loading || saving || !snapshot?.supported}
					onchange={(event) => {
						const enabled = event.currentTarget.checked
						event.currentTarget.checked = snapshot?.enabled ?? false
						void setEnabled(enabled)
					}} />
				<span aria-hidden="true"></span>
			</label>
		</div>
	</div>
	{#if error || statusError}
		<div class="settings-inline-error" role="alert">{error || statusError}</div>
		{#if !snapshot}
			<div class="settings-form-actions">
				<button
					class="settings-row-action"
					type="button"
					disabled={loading}
					onclick={() => void load()}>Retry</button>
			</div>
		{/if}
	{/if}
</section>

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
			<div class="settings-inline-error" role="alert">{clientsError}</div>
		{/if}
	</section>
{:else if clientsError}
	<div class="settings-inline-error" role="alert">{clientsError}</div>
	<div class="settings-form-actions">
		<button
			class="settings-row-action"
			type="button"
			disabled={clientsLoading}
			onclick={() => void loadClients()}>Retry</button>
	</div>
{/if}

<style>
	.unavailable strong {
		color: var(--color-text-faint);
	}
</style>

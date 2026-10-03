<script lang="ts">
	import { onMount } from "svelte"
	import { accountBackend, type MultiInstanceSnapshot } from "../backend/bridge"
	import SettingsStatus from "./SettingsStatus.svelte"

	let snapshot = $state<MultiInstanceSnapshot | null>(null),
		loading = $state(true),
		saving = $state(false),
		error = $state(""),
		statusError = $state("")
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

	onMount(() => {
		void load()
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

<style>
	.unavailable strong {
		color: var(--color-text-faint);
	}
</style>

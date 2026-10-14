<script lang="ts">
	import XIcon from "phosphor-svelte/lib/XIcon"
	import { untrack } from "svelte"
	import { accountBackend, type RobloxClientState } from "../backend/bridge"
	import { automaticLinuxClient } from "../settings/linux-clients"
	import {
		modalBackdropIn,
		modalBackdropOut,
		modalCardIn,
		modalCardOut,
		modalFocus,
	} from "../shared/presence"
	import { errorMessage } from "../shared/errors"

	let {
		clients,
		onSaved,
		onClose,
	}: {
		clients: RobloxClientState
		onSaved: (clients: RobloxClientState) => void
		onClose: () => void
	} = $props()

	const radioName = $props.id(),
		options = $derived([
			{ ...automaticLinuxClient, installed: true },
			...(clients.clients ?? []).map((client) => ({
				value: client.id,
				label: client.name,
				description: "",
				installed: client.installed,
			})),
		]),
		installed = $derived((clients.clients ?? []).some((client) => client.installed))
	let selection = $state(untrack(() => clients.selected)),
		saving = $state(false),
		error = $state("")

	async function save(): Promise<void> {
		if (selection === clients.selected) {
			onClose()
			return
		}
		saving = true
		error = ""
		try {
			onSaved(await accountBackend.SetLinuxClient(selection))
			onClose()
		} catch (cause) {
			error = errorMessage(
				cause,
				"The Linux client could not be saved. Try again.",
			)
		} finally {
			saving = false
		}
	}
</script>

<svelte:window
	onkeydown={(event) => {
		if (event.key === "Escape") {
			event.preventDefault()
			onClose()
		}
	}} />

<div
	class="modal-backdrop"
	in:modalBackdropIn
	out:modalBackdropOut
	onclick={(event) => {
		if (event.target === event.currentTarget) onClose()
	}}
	role="presentation">
	<form
		novalidate
		class="modal-card launch-data-card"
		use:modalFocus
		in:modalCardIn
		out:modalCardOut
		aria-labelledby="linux-client-dialog-title"
		aria-busy={saving}
		onsubmit={(event) => {
			event.preventDefault()
			void save()
		}}>
		<div class="modal-heading compact-heading">
			<div>
				<h2 id="linux-client-dialog-title">Linux client</h2>
				<p>Choose the client that joins games.</p>
			</div>
			<button
				class="icon-action"
				type="button"
				aria-label="Close Linux client dialog"
				onclick={onClose}>
				<XIcon size={17} aria-hidden="true" />
			</button>
		</div>
		<div
			class="settings-radio-options"
			role="radiogroup"
			aria-labelledby="linux-client-dialog-title">
			{#each options as option (option.value)}
				<label
					class="settings-radio-option"
					class:disabled={saving || !option.installed}
					data-tooltip={option.description || undefined}>
					<span class="settings-radio">
						<input
							type="radio"
							name={radioName}
							bind:group={selection}
							value={option.value}
							disabled={saving || !option.installed} />
						<span aria-hidden="true"></span>
					</span>
					<span class="settings-radio-label">{option.label}</span>
					{#if !option.installed}
						<small class="linux-client-missing">Not installed</small>
					{/if}
				</label>
			{/each}
		</div>
		{#if !installed}
			<div class="modal-note">Install Sober or Mocktail to join games.</div>
		{/if}
		{#if error}
			<div class="modal-error" role="alert">{error}</div>
		{/if}
		<div class="modal-actions">
			<button type="button" disabled={saving} onclick={onClose}>Cancel</button>
			<button class="primary-action" type="submit" disabled={saving}>
				Save
			</button>
		</div>
	</form>
</div>

<style>
	.linux-client-missing {
		margin-left: auto;
		color: var(--color-text-faint);
		font-size: var(--text-11);
	}
</style>

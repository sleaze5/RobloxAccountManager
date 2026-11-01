<script lang="ts">
	import xIcon from "@phosphor-icons/core/regular/x.svg?raw"
	import Icon from "../shared/Icon.svelte"
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
	class="dialog-backdrop"
	in:modalBackdropIn
	out:modalBackdropOut
	onclick={(event) => {
		if (event.target === event.currentTarget) onClose()
	}}
	role="presentation">
	<div
		class="dialog-card max-w-md"
		role="dialog"
		aria-modal="true"
		use:modalFocus
		in:modalCardIn
		out:modalCardOut
		aria-labelledby="linux-client-dialog-title"
		aria-busy={saving}>
		<form
			novalidate
			onsubmit={(event) => {
				event.preventDefault()
				void save()
			}}>
			<div class="dialog-heading mb-2">
				<div>
					<h2 id="linux-client-dialog-title">Linux client</h2>
					<p>Choose the client that joins games.</p>
				</div>
				<button
					class="btn-ghost-sm"
					type="button"
					aria-label="Close Linux client dialog"
					onclick={onClose}>
					<Icon svg={xIcon} size={14} aria-hidden="true" />
				</button>
			</div>
			<div
				class="flex flex-col gap-0.5 pl-1"
				role="radiogroup"
				aria-labelledby="linux-client-dialog-title">
				{#each options as option (option.value)}
					<label
						class="radio-option"
						data-disabled={saving || !option.installed || undefined}
						data-tooltip={option.description || undefined}>
						<span class="radio">
							<input
								type="radio"
								name={radioName}
								bind:group={selection}
								value={option.value}
								disabled={saving || !option.installed} />
							<span aria-hidden="true"></span>
						</span>
						<span class="text-xs text-fg">{option.label}</span>
						{#if !option.installed}
							<small class="ml-auto text-2xs text-fg-faint"
								>Not installed</small>
						{/if}
					</label>
				{/each}
			</div>
			{#if !installed}
				<div class="mt-3 notice-note">
					Install Sober or Mocktail to join games.
				</div>
			{/if}
			{#if error}
				<div class="mt-3 notice-error" role="alert">{error}</div>
			{/if}
			<div class="dialog-actions">
				<button class="btn" type="button" disabled={saving} onclick={onClose}
					>Cancel</button>
				<button class="btn-primary" type="submit" disabled={saving}>
					Save
				</button>
			</div>
		</form>
	</div>
</div>

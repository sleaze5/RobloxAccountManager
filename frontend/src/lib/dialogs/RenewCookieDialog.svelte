<script lang="ts">
	import CheckIcon from "phosphor-svelte/lib/CheckIcon"
	import CircleNotchIcon from "phosphor-svelte/lib/CircleNotchIcon"
	import XIcon from "phosphor-svelte/lib/XIcon"
	import type { Account } from "../accounts/account-model"
	import type { AccountStore } from "../accounts/account-store.svelte"
	import { accountBackend } from "../backend/bridge"
	import {
		modalBackdropIn,
		modalBackdropOut,
		modalCardIn,
		modalCardOut,
		modalFocus,
	} from "../shared/presence"
	import { errorMessage } from "../shared/errors"

	let {
		store,
		account,
		onClose,
	}: {
		store: AccountStore
		account: Account
		onClose: () => void
	} = $props()
	let phase = $state<"confirm" | "pending" | "complete">("confirm"),
		error = $state(""),
		warning = $state("")
	let dialog = $state<HTMLDivElement | undefined>(undefined)
	const pending = $derived(phase === "pending")

	async function renew(): Promise<void> {
		if (phase !== "confirm" || store.busy || !store.vault.unlocked) return
		phase = "pending"
		store.busy = true
		dialog?.focus()
		try {
			const result = await accountBackend.RenewCookie(account.id)
			warning = result.warning ?? ""
		} catch (cause) {
			error = errorMessage(cause, "The renewal failed.")
		} finally {
			phase = "complete"
			store.busy = false
			if (store.vault.unlocked) void store.refreshAccounts()
		}
	}
</script>

<div class="dialog-backdrop" in:modalBackdropIn out:modalBackdropOut>
	<div
		bind:this={dialog}
		tabindex="-1"
		class="dialog-card max-w-sm text-left"
		role="dialog"
		aria-modal="true"
		aria-labelledby="renew-cookie-title"
		aria-describedby={phase === "confirm"
			? "renew-cookie-description"
			: "renew-cookie-status"}
		use:modalFocus
		in:modalCardIn
		out:modalCardOut>
		<div class="dialog-heading mb-2">
			<div>
				<h2 id="renew-cookie-title">
					{phase === "confirm" ? "Renew cookie?" : "Renew cookie"}
				</h2>
				<p>@{account.username}</p>
			</div>
			<button
				type="button"
				aria-label="Close cookie renewal"
				disabled={pending}
				onclick={onClose}>
				<XIcon size={14} aria-hidden="true" />
			</button>
		</div>
		{#if phase === "confirm"}
			<p id="renew-cookie-description">
				This replaces the saved session cookie. Any open browser for this
				account will need to be reopened.
			</p>
		{:else}
			<div
				id="renew-cookie-status"
				class="flex items-center gap-2 py-2 text-xs *:shrink-0"
				role="status"
				aria-live="polite">
				{#if pending}
					<CircleNotchIcon size={16} class="spinner" aria-hidden="true" />
					<span>Renewing the cookie and saving it…</span>
				{:else if error}
					<span>Renewal could not be confirmed.</span>
				{:else}
					<CheckIcon size={16} class="text-success" aria-hidden="true" />
					<span>The renewed cookie is saved.</span>
				{/if}
			</div>
		{/if}
		{#if error}
			<div class="mt-3 notice-error" role="alert">{error}</div>
		{/if}
		{#if warning}
			<p class="text-warning" role="status">{warning}</p>
		{/if}
		<div class="dialog-actions">
			{#if phase === "confirm"}
				<button class="btn" type="button" onclick={onClose}>Cancel</button>
				<button
					class="btn-warning"
					type="button"
					disabled={store.busy}
					onclick={() => void renew()}>Renew cookie</button>
			{:else}
				<button class="btn" type="button" disabled={pending} onclick={onClose}
					>{pending ? "Renewing…" : "Close"}</button>
			{/if}
		</div>
	</div>
</div>

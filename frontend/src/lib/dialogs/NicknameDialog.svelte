<script lang="ts">
	import { notifyErrors } from "../notifications/error-notifications.svelte"
	import { untrack } from "svelte"
	import xIcon from "@phosphor-icons/core/regular/x.svg?raw"
	import Icon from "../shared/Icon.svelte"
	import Dialog from "../shared/Dialog.svelte"
	import type { Account } from "../accounts/account-model"
	import type { SocialStore, SocialSubject } from "../social/social-store.svelte"

	let {
		store,
		account,
		person,
		nickname,
		onClose,
	}: {
		store: SocialStore
		account: Account
		person: SocialSubject
		nickname: string
		onClose: () => void
	} = $props()

	const maximumLength = 20,
		displayName = $derived(person.displayName || person.username)
	let typedNickname = $state(untrack(() => nickname))
	const text = $derived(typedNickname.trim()),
		saving = $derived(store.savingNickname),
		removing = $derived(!!nickname && !text)

	function close(): void {
		if (!saving) onClose()
	}

	async function save(): Promise<void> {
		if (await store.setNickname(person, text)) onClose()
	}

	notifyErrors("Nickname not saved", () => store.nicknameError)
</script>

<Dialog
	class="max-w-sm text-left"
	aria-labelledby="nickname-title"
	aria-describedby="nickname-description">
	<div class="dialog-heading">
		<div>
			<h2 id="nickname-title">
				{nickname ? "Edit nickname" : "Add a nickname"}
			</h2>
			<p id="nickname-description">
				Saved to Roblox for @{account.username}. Only that account sees it, on
				the Roblox website and app.
			</p>
		</div>
		<button
			type="button"
			aria-label="Close dialog"
			disabled={saving}
			onclick={close}>
			<Icon svg={xIcon} size={14} aria-hidden="true" />
		</button>
	</div>
	<form
		novalidate
		class="contents"
		onsubmit={(event) => {
			event.preventDefault()
			void save()
		}}>
		<div class="dialog-field">
			<label for="nickname-field">Nickname for {displayName}</label>
			<input
				id="nickname-field"
				class="field"
				type="text"
				autocomplete="off"
				placeholder={displayName}
				maxlength={maximumLength}
				aria-describedby="nickname-help"
				disabled={saving}
				bind:value={typedNickname} />
			<small
				id="nickname-help"
				class="flex justify-between gap-2 text-xs text-fg-muted">
				<span>Leave it empty to show their display name.</span>
				<span class="font-mono text-2xs tabular-nums"
					>{Array.from(typedNickname).length}/{maximumLength}</span>
			</small>
		</div>
		<div class="dialog-actions">
			<button class="button" type="button" disabled={saving} onclick={close}
				>Cancel</button>
			<button
				class="button-primary"
				type="submit"
				disabled={saving || text === nickname}>
				{#if removing}{saving ? "Removing…" : "Remove nickname"}{:else}{saving
						? "Saving…"
						: "Save nickname"}{/if}
			</button>
		</div>
	</form>
</Dialog>

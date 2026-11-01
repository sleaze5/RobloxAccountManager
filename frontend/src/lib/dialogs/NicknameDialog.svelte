<script lang="ts">
	import { untrack } from "svelte"
	import xIcon from "@phosphor-icons/core/regular/x.svg?raw"
	import Icon from "../shared/Icon.svelte"
	import type { Account } from "../accounts/account-model"
	import type { SocialStore, SocialSubject } from "../social/social-store.svelte"
	import {
		modalBackdropIn,
		modalBackdropOut,
		modalCardIn,
		modalCardOut,
		modalFocus,
	} from "../shared/presence"

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
	let value = $state(untrack(() => nickname))
	const text = $derived(value.trim()),
		saving = $derived(store.updating === "nickname"),
		removing = $derived(!!nickname && !text)

	function close(): void {
		if (!saving) onClose()
	}

	async function save(): Promise<void> {
		if (await store.setNickname(person, text)) onClose()
	}
</script>

<div class="dialog-backdrop" in:modalBackdropIn out:modalBackdropOut>
	<div
		class="dialog-card max-w-sm text-left"
		role="dialog"
		aria-modal="true"
		aria-labelledby="nickname-title"
		aria-describedby="nickname-description"
		use:modalFocus
		in:modalCardIn
		out:modalCardOut>
		<div class="dialog-heading">
			<div>
				<h2 id="nickname-title">
					{nickname ? "Edit nickname" : "Add a nickname"}
				</h2>
				<p id="nickname-description">
					Saved to Roblox for @{account.username}. Only that account sees it,
					on the Roblox website and app.
				</p>
			</div>
			<button
				type="button"
				aria-label="Close nickname dialog"
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
					bind:value />
				<small
					id="nickname-help"
					class="flex justify-between gap-2 text-xs text-fg-muted">
					<span>Leave it empty to show their display name.</span>
					<span class="font-mono text-2xs tabular-nums"
						>{Array.from(value).length}/{maximumLength}</span>
				</small>
			</div>
			{#if store.nicknameError}
				<div class="mt-3 notice-error" role="alert">{store.nicknameError}</div>
			{/if}
			<div class="dialog-actions">
				<button class="btn" type="button" disabled={saving} onclick={close}
					>Cancel</button>
				<button
					class="btn-primary"
					type="submit"
					disabled={saving || text === nickname}>
					{#if removing}{saving
							? "Removing…"
							: "Remove nickname"}{:else}{saving
							? "Saving…"
							: "Save nickname"}{/if}
				</button>
			</div>
		</form>
	</div>
</div>

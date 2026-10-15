<script lang="ts" module>
	export type AccountEditKind = "displayName" | "email" | "password" | "pin"
</script>

<script lang="ts">
	import EyeIcon from "phosphor-svelte/lib/EyeIcon"
	import EyeSlashIcon from "phosphor-svelte/lib/EyeSlashIcon"
	import XIcon from "phosphor-svelte/lib/XIcon"
	import type { Account } from "../accounts/account-model"
	import { accountBackend } from "../backend/bridge"
	import type { NotificationCenter } from "../notifications/notification-center.svelte"
	import {
		modalBackdropIn,
		modalBackdropOut,
		modalCardIn,
		modalCardOut,
		modalFocus,
		replayValidationNudge,
	} from "../shared/presence"
	import { errorMessage } from "../shared/errors"

	let {
		account,
		kind,
		notifications,
		onSaved,
		onClose,
	}: {
		account: Account
		kind: AccountEditKind
		notifications: NotificationCenter
		onSaved: () => void
		onClose: () => void
	} = $props()

	const titles: Record<AccountEditKind, string> = {
			displayName: "Change display name",
			email: "Change email",
			password: "Change Roblox password",
			pin: "Unlock PIN",
		},
		successes: Record<AccountEditKind, string> = {
			displayName: "Display name changed",
			email: "Email changed",
			password: "Password changed",
			pin: "PIN unlocked",
		}

	let phase = $state<"edit" | "confirm">("edit"),
		pending = $state(false),
		value = $state(""),
		password = $state(""),
		newPassword = $state(""),
		confirmation = $state(""),
		visible = $state(false),
		error = $state(""),
		card = $state<HTMLDivElement>()
	const ready = $derived(
		{
			displayName: value.trim() !== "",
			email: value.trim() !== "" && password !== "",
			password:
				password !== "" && newPassword !== "" && newPassword === confirmation,
			pin: /^[0-9]{4}$/.test(value),
		}[kind],
	)

	function close(): void {
		if (!pending) onClose()
	}

	function submit(): void {
		if (!ready || pending) {
			replayValidationNudge(card)
		} else if (phase === "edit" && kind !== "pin") {
			phase = "confirm"
		} else {
			void save()
		}
	}

	function change(): Promise<void> {
		switch (kind) {
			case "displayName":
				return accountBackend.ChangeDisplayName(account.id, value)
			case "email":
				return accountBackend.ChangeAccountEmail(account.id, password, value)
			case "password":
				return accountBackend.ChangeAccountPassword(
					account.id,
					password,
					newPassword,
				)
			case "pin":
				return accountBackend.UnlockAccountPin(account.id, value)
		}
	}

	async function save(): Promise<void> {
		pending = true
		error = ""
		try {
			await change()
			notifications.show({
				id: `account-edit-${account.id}-${kind}`,
				title: successes[kind],
				tone: "success",
				message: `@${account.username}`,
			})
			onSaved()
			onClose()
		} catch (cause) {
			error = errorMessage(
				cause,
				"Roblox did not confirm the change. Refresh to check.",
			)
			pending = false
			phase = "edit"
			replayValidationNudge(card)
		}
	}
</script>

<svelte:window
	onkeydown={(event) => {
		if (event.key === "Escape") close()
	}} />

<div class="modal-backdrop" in:modalBackdropIn out:modalBackdropOut>
	<div
		class="modal-card confirm-card"
		role={phase === "confirm" ? "alertdialog" : "dialog"}
		aria-modal="true"
		aria-labelledby="account-edit-title"
		bind:this={card}
		use:modalFocus
		in:modalCardIn
		out:modalCardOut>
		<form
			novalidate
			onsubmit={(event) => {
				event.preventDefault()
				submit()
			}}>
			<div class="modal-heading compact-heading">
				<div>
					<h2 id="account-edit-title">
						{phase === "confirm" ? `${titles[kind]}?` : titles[kind]}
					</h2>
					<p>@{account.username}</p>
				</div>
				<button
					type="button"
					aria-label={`Close ${titles[kind].toLowerCase()}`}
					disabled={pending}
					onclick={close}>
					<XIcon size={14} aria-hidden="true" />
				</button>
			</div>

			{#if phase === "confirm"}
				{#if kind === "displayName"}
					<p>The new display name will be “{value.trim()}”.</p>
					<div class="modal-warning">
						You can't change the display name again for 7 days.
					</div>
				{:else if kind === "email"}
					<p>The new email will be {value.trim()}.</p>
					<div class="modal-note">
						Roblox sends a verification email to the new address.
					</div>
				{:else}
					<div class="modal-warning">
						{account.hasSavedPassword
							? "The saved password in your vault will also be updated."
							: "The new password will not be saved in the vault. Store it somewhere safe before you continue."}
					</div>
				{/if}
			{:else}
				{#if kind === "displayName"}
					<label class="modal-field">
						<span>New display name</span>
						<input
							bind:value
							autocomplete="off"
							spellcheck="false"
							disabled={pending} />
					</label>
					<div class="modal-warning">
						After you change it, you can't change it again for 7 days.
					</div>
				{:else if kind === "email"}
					<label class="modal-field">
						<span>New email</span>
						<input
							bind:value
							type="email"
							autocomplete="off"
							spellcheck="false"
							disabled={pending} />
					</label>
				{:else if kind === "pin"}
					<label class="modal-field">
						<span>PIN</span>
						<input
							bind:value
							inputmode="numeric"
							maxlength="4"
							autocomplete="off"
							disabled={pending} />
					</label>
				{/if}
				{#if kind === "email" || kind === "password"}
					<label class="modal-field">
						<span
							>{kind === "email" ? "Password" : "Current password"}</span>
						<input
							bind:value={password}
							type={visible ? "text" : "password"}
							autocomplete="off"
							disabled={pending} />
					</label>
				{/if}
				{#if kind === "password"}
					<label class="modal-field">
						<span>New password</span>
						<input
							bind:value={newPassword}
							type={visible ? "text" : "password"}
							autocomplete="off"
							disabled={pending} />
					</label>
					<label class="modal-field">
						<span>Confirm new password</span>
						<input
							bind:value={confirmation}
							type={visible ? "text" : "password"}
							autocomplete="off"
							aria-invalid={confirmation !== "" &&
								confirmation !== newPassword}
							disabled={pending} />
					</label>
				{/if}
				{#if kind === "email" || kind === "password"}
					<button
						class="visibility-control"
						type="button"
						onclick={() => (visible = !visible)}>
						{#if visible}<EyeSlashIcon size={16} aria-hidden="true" /> Hide passwords{:else}<EyeIcon
								size={16}
								aria-hidden="true" /> Show passwords{/if}
					</button>
				{/if}
			{/if}

			{#if error}
				<div class="modal-error" role="alert">{error}</div>
			{/if}

			<div class="modal-actions">
				{#if phase === "confirm"}
					<button
						class="control-button text-button"
						type="button"
						disabled={pending}
						onclick={() => (phase = "edit")}>Back</button>
				{:else}
					<button
						class="control-button text-button"
						type="button"
						disabled={pending}
						onclick={close}>Cancel</button>
				{/if}
				<button
					class="primary-action"
					type="submit"
					disabled={!ready || pending}>
					{#if pending}
						{kind === "pin" ? "Unlocking…" : "Saving…"}
					{:else if phase === "confirm" || kind === "pin"}
						{kind === "pin" ? "Unlock" : titles[kind]}
					{:else}
						Continue
					{/if}
				</button>
			</div>
		</form>
	</div>
</div>

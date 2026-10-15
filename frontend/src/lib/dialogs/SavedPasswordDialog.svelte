<script lang="ts">
	import CheckCircleIcon from "phosphor-svelte/lib/CheckCircleIcon"
	import CircleNotchIcon from "phosphor-svelte/lib/CircleNotchIcon"
	import EyeIcon from "phosphor-svelte/lib/EyeIcon"
	import EyeSlashIcon from "phosphor-svelte/lib/EyeSlashIcon"
	import InfoIcon from "phosphor-svelte/lib/InfoIcon"
	import ShieldCheckIcon from "phosphor-svelte/lib/ShieldCheckIcon"
	import WarningCircleIcon from "phosphor-svelte/lib/WarningCircleIcon"
	import XIcon from "phosphor-svelte/lib/XIcon"
	import { onDestroy, onMount, tick, untrack } from "svelte"
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
		notifications,
		onClose,
	}: {
		account: Account
		notifications: NotificationCenter
		onClose: () => void
	} = $props()

	const accountId = untrack(() => account.id),
		replacing = untrack(() => account.hasSavedPassword),
		title = replacing ? "Update saved password" : "Save password",
		validationHint =
			"Checks the password with Roblox. Too many checks can temporarily block validation."
	let saved = $state(""),
		password = $state(""),
		visible = $state(false),
		loading = $state(replacing),
		pending = $state<"validate" | "save" | null>(null),
		validation = $state<"unchecked" | "valid" | "invalid">("unchecked"),
		error = $state(""),
		card = $state<HTMLDivElement>(),
		input = $state<HTMLInputElement>()
	let active = true,
		request:
			| ReturnType<typeof accountBackend.GetAccountPassword>
			| ReturnType<typeof accountBackend.ValidateAccountPassword>
			| ReturnType<typeof accountBackend.SaveAccountPassword>
			| null = null
	const busy = $derived(loading || pending !== null)

	async function focusInput(): Promise<void> {
		await tick()
		if (active) input?.focus()
	}

	async function load(): Promise<void> {
		try {
			request = accountBackend.GetAccountPassword(accountId)
			const value = await request
			if (!active) return
			saved = value
			password = value
		} catch (cause) {
			if (!active) return
			error = errorMessage(cause, "Could not load the saved password.")
		} finally {
			request = null
			loading = false
		}
	}

	function close(): void {
		if (pending) return
		onClose()
	}

	async function validate(): Promise<void> {
		if (busy || !password) return
		pending = "validate"
		error = ""
		try {
			request = accountBackend.ValidateAccountPassword(accountId, password)
			const valid = await request
			if (!active) return
			validation = valid ? "valid" : "invalid"
			if (!valid) replayValidationNudge(card)
		} catch (cause) {
			if (!active) return
			error = errorMessage(
				cause,
				"Could not validate the password. Try again later.",
			)
		} finally {
			request = null
			pending = null
		}
		await focusInput()
	}

	async function save(): Promise<void> {
		if (busy || !password || password === saved) return
		pending = "save"
		error = ""
		try {
			request = accountBackend.SaveAccountPassword(accountId, password)
			await request
			if (!active) return
			notifications.show({
				id: `account-password-saved-${accountId}`,
				title: "Password saved",
				tone: "success",
				message: `@${account.username}`,
			})
			onClose()
			return
		} catch (cause) {
			if (!active) return
			error = errorMessage(cause, "Could not save the password. Try again.")
			replayValidationNudge(card)
		} finally {
			request = null
			pending = null
		}
		await focusInput()
	}

	onMount(async () => {
		if (replacing) await load()
		await focusInput()
	})

	onDestroy(() => {
		active = false
		void request?.cancel()
		saved = ""
		password = ""
	})
</script>

<svelte:window
	onkeydown={(event) => {
		if (event.key === "Escape") close()
	}} />

<div class="modal-backdrop" in:modalBackdropIn out:modalBackdropOut>
	<div
		class="modal-card confirm-card"
		role="dialog"
		aria-modal="true"
		aria-labelledby="saved-password-title"
		bind:this={card}
		use:modalFocus
		in:modalCardIn
		out:modalCardOut>
		<form
			novalidate
			onsubmit={(event) => {
				event.preventDefault()
				void save()
			}}>
			<div class="modal-heading compact-heading">
				<div>
					<h2 id="saved-password-title">{title}</h2>
					<p>@{account.username}</p>
				</div>
				<button
					type="button"
					aria-label={`Close ${title.toLowerCase()}`}
					disabled={pending !== null}
					onclick={close}>
					<XIcon size={16} aria-hidden="true" />
				</button>
			</div>

			<div class="password-field">
				<label class="modal-field">
					<span>Roblox password</span>
					<input
						bind:this={input}
						bind:value={password}
						type={visible ? "text" : "password"}
						placeholder={loading ? "Loading..." : ""}
						autocomplete="off"
						spellcheck="false"
						disabled={busy}
						oninput={() => {
							validation = "unchecked"
							error = ""
						}} />
				</label>
				<button
					class="icon-action"
					type="button"
					aria-label={visible ? "Hide password" : "Show password"}
					data-tooltip={visible ? "Hide password" : "Show password"}
					onclick={() => (visible = !visible)}>
					{#if visible}<EyeSlashIcon
							size={16}
							aria-hidden="true" />{:else}<EyeIcon
							size={16}
							aria-hidden="true" />{/if}
				</button>
			</div>

			<div class="password-validation">
				{#if validation === "valid"}
					<span class="password-valid" role="status"
						><CheckCircleIcon size={16} aria-hidden="true" />Password is
						correct</span>
				{:else if validation === "invalid"}
					<span class="password-invalid" role="status"
						><WarningCircleIcon size={16} aria-hidden="true" />Password is
						incorrect</span>
				{:else}
					<button
						class="visibility-control"
						type="button"
						disabled={busy || !password}
						onclick={() => void validate()}>
						{#if pending === "validate"}<CircleNotchIcon
								class="spinner"
								size={16}
								aria-hidden="true" /> Validating...{:else}<ShieldCheckIcon
								size={16}
								aria-hidden="true" /> Validate with Roblox{/if}
					</button>
					<button
						class="automatic-unlock-info"
						type="button"
						aria-label={validationHint}
						data-tooltip={validationHint}>
						<InfoIcon size={15} aria-hidden="true" />
					</button>
				{/if}
			</div>

			{#if error}<div class="modal-error" role="alert">{error}</div>{/if}

			<div class="modal-actions">
				<button type="button" disabled={pending !== null} onclick={close}
					>Cancel</button>
				<button
					class="primary-action"
					type="submit"
					disabled={busy || !password || password === saved}>
					{pending === "save" ? "Saving..." : "Save"}
				</button>
			</div>
		</form>
	</div>
</div>

<style>
	.password-field {
		position: relative;
	}

	.password-field input {
		padding-right: calc(var(--control-height-sm) + var(--space-2));
	}

	.password-field > button {
		position: absolute;
		right: var(--space-1);
		bottom: var(--space-1);
		width: var(--control-height-sm);
		height: var(--control-height-sm);
	}

	.password-validation {
		display: flex;
		min-height: var(--control-height-sm);
		align-items: center;
		gap: var(--space-1);
		margin-top: var(--space-2);
		color: var(--color-text);
		font-size: var(--text-12);
	}

	.password-validation > span {
		display: flex;
		align-items: center;
		gap: var(--space-1);
	}

	.password-validation > .visibility-control {
		margin-top: 0;
	}

	.password-validation > .password-valid {
		color: var(--color-success);
	}

	.password-validation > .password-invalid {
		color: var(--color-warning);
	}
</style>

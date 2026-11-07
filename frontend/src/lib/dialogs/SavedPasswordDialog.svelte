<script lang="ts">
	import checkCircleIcon from "@phosphor-icons/core/regular/check-circle.svg?raw"
	import circleNotchIcon from "@phosphor-icons/core/regular/circle-notch.svg?raw"
	import eyeIcon from "@phosphor-icons/core/regular/eye.svg?raw"
	import eyeSlashIcon from "@phosphor-icons/core/regular/eye-slash.svg?raw"
	import infoIcon from "@phosphor-icons/core/regular/info.svg?raw"
	import shieldCheckIcon from "@phosphor-icons/core/regular/shield-check.svg?raw"
	import warningCircleIcon from "@phosphor-icons/core/regular/warning-circle.svg?raw"
	import xIcon from "@phosphor-icons/core/regular/x.svg?raw"
	import Icon from "../shared/Icon.svelte"
	import { onDestroy, onMount, tick, untrack } from "svelte"
	import type { Account } from "../accounts/account-model"
	import { accountBackend } from "../backend/bridge"
	import type { NotificationCenter } from "../notifications/notification-center.svelte"
	import { replayValidationNudge } from "../shared/presence"
	import { errorMessage } from "../shared/errors"
	import Dialog from "../shared/Dialog.svelte"

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

<Dialog class="max-w-sm text-left" aria-labelledby="saved-password-title" bind:card>
	<form
		novalidate
		onsubmit={(event) => {
			event.preventDefault()
			void save()
		}}>
		<div class="dialog-heading mb-2">
			<div>
				<h2 id="saved-password-title">{title}</h2>
				<p>@{account.username}</p>
			</div>
			<button
				type="button"
				aria-label={`Close ${title.toLowerCase()}`}
				disabled={pending !== null}
				onclick={close}>
				<Icon svg={xIcon} size={14} aria-hidden="true" />
			</button>
		</div>

		<div class="relative">
			<label class="dialog-field">
				<span>Roblox password</span>
				<input
					bind:this={input}
					bind:value={password}
					class="pr-8"
					type={visible ? "text" : "password"}
					placeholder={loading ? "Loading…" : ""}
					autocomplete="off"
					spellcheck="false"
					disabled={busy}
					oninput={() => {
						validation = "unchecked"
						error = ""
					}} />
			</label>
			<button
				class="absolute right-1 bottom-1 button-ghost size-6"
				type="button"
				aria-label={visible ? "Hide password" : "Show password"}
				data-tooltip={visible ? "Hide password" : "Show password"}
				onclick={() => (visible = !visible)}>
				{#if visible}<Icon
						svg={eyeSlashIcon}
						size={16}
						aria-hidden="true" />{:else}<Icon
						svg={eyeIcon}
						size={16}
						aria-hidden="true" />{/if}
			</button>
		</div>

		<div class="mt-2 flex min-h-6 items-center gap-1 text-xs text-fg">
			{#if validation === "valid"}
				<span class="flex items-center gap-1 text-success" role="status"
					><Icon svg={checkCircleIcon} size={16} aria-hidden="true" />Password
					is correct</span>
			{:else if validation === "invalid"}
				<span class="flex items-center gap-1 text-warning" role="status"
					><Icon
						svg={warningCircleIcon}
						size={16}
						aria-hidden="true" />Password is incorrect</span>
			{:else}
				<button
					class="flex items-center gap-1 py-1 text-xs text-fg-muted transition-colors duration-100 *:animate-pop"
					type="button"
					disabled={busy || !password}
					onclick={() => void validate()}>
					{#if pending === "validate"}<Icon
							svg={circleNotchIcon}
							class="spinner"
							size={16}
							aria-hidden="true" /> Validating…{:else}<Icon
							svg={shieldCheckIcon}
							size={16}
							aria-hidden="true" /> Validate with Roblox{/if}
				</button>
				<button
					class="flex size-5 cursor-help items-center justify-center text-fg-faint transition-colors duration-100 hover:text-fg focus-visible:text-fg"
					type="button"
					aria-label={validationHint}
					data-tooltip={validationHint}>
					<Icon svg={infoIcon} size={14} aria-hidden="true" />
				</button>
			{/if}
		</div>

		{#if error}<div class="mt-3 notice-error" role="alert">{error}</div>{/if}

		<div class="dialog-actions">
			<button
				class="button"
				type="button"
				disabled={pending !== null}
				onclick={close}>Cancel</button>
			<button
				class="button-primary"
				type="submit"
				disabled={busy || !password || password === saved}>
				{pending === "save" ? "Saving…" : "Save"}
			</button>
		</div>
	</form>
</Dialog>

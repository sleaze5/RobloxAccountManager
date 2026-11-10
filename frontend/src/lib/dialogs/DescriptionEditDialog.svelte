<script lang="ts">
	import { tick, untrack } from "svelte"
	import xIcon from "@phosphor-icons/core/regular/x.svg?raw"
	import Icon from "../shared/Icon.svelte"
	import type { Account } from "../accounts/account-model"
	import { accountBackend } from "../backend/bridge"
	import type { NotificationCenter } from "../notifications/notification-center.svelte"
	import { replayValidationNudge } from "../shared/presence"
	import { errorMessage } from "../shared/errors"
	import { TextFilterCheck } from "../shared/text-filter-check.svelte"
	import TextFilterStatus from "../shared/TextFilterStatus.svelte"
	import TextFilterValidateButton from "../shared/TextFilterValidateButton.svelte"
	import Dialog from "../shared/Dialog.svelte"

	let {
		account,
		description,
		notifications,
		onSaved,
		onClose,
	}: {
		account: Account
		description: string
		notifications: NotificationCenter
		onSaved: (description: string) => void
		onClose: () => void
	} = $props()

	let value = $state(untrack(() => description)),
		pending = $state<"validate" | "submit" | "save" | null>(null),
		error = $state(""),
		card = $state<HTMLDivElement>(),
		field = $state<HTMLTextAreaElement>()
	const text = $derived(value.trim()),
		changed = $derived(text !== description.trim()),
		check = new TextFilterCheck(() => text)

	function close(): void {
		if (!pending) onClose()
	}

	async function validate(mode: "validate" | "submit"): Promise<boolean> {
		const validatedText = text
		pending = mode
		error = ""
		try {
			return await check.run(validatedText, () =>
				accountBackend.ValidateAccountDescription(account.id, validatedText),
			)
		} catch (cause) {
			error = errorMessage(
				cause,
				"Could not validate the description. Try again.",
			)
			return false
		} finally {
			pending = null
		}
	}

	async function edit(): Promise<void> {
		check.clear()
		await tick()
		field?.focus()
	}

	async function save(): Promise<void> {
		pending = "save"
		error = ""
		try {
			const saved = await accountBackend.ChangeAccountDescription(
				account.id,
				text,
			)
			notifications.show({
				id: `account-edit-${account.id}-description`,
				title: "Description saved",
				tone: "success",
				message: `@${account.username}`,
			})
			onSaved(saved)
			onClose()
		} catch (cause) {
			error = errorMessage(
				cause,
				"Roblox did not confirm the change. Refresh to check.",
			)
			pending = null
			replayValidationNudge(card)
		}
	}

	async function submit(): Promise<void> {
		if (!changed || pending) {
			replayValidationNudge(card)
		} else if (text === "" || check.current || (await validate("submit"))) {
			await save()
		}
	}
</script>

<svelte:window
	onkeydown={(event) => {
		if (event.key === "Escape") close()
	}} />

<Dialog class="max-w-md text-left" aria-labelledby="description-edit-title" bind:card>
	<form
		novalidate
		onsubmit={(event) => {
			event.preventDefault()
			void submit()
		}}>
		<div class="dialog-heading mb-2">
			<div>
				<h2 id="description-edit-title">Edit description</h2>
				<p>@{account.username}</p>
			</div>
			<button
				type="button"
				aria-label="Close edit description"
				disabled={pending !== null}
				onclick={close}>
				<Icon svg={xIcon} size={14} aria-hidden="true" />
			</button>
		</div>

		<label class="dialog-field">
			<span>Description</span>
			{#if check.filteredPreview === undefined}
				<textarea bind:this={field} bind:value disabled={pending !== null}
				></textarea>
			{:else}
				<textarea
					value={check.filteredPreview}
					readonly
					disabled={pending !== null}></textarea>
			{/if}
		</label>

		<div aria-live="polite">
			{#if changed && text === ""}
				<div class="mt-3 notice-warning">
					This removes the description from @{account.username}'s profile.
				</div>
			{:else}
				<TextFilterStatus {check} noun="text" />
			{/if}
		</div>

		{#if error}
			<div class="mt-3 notice-error" role="alert">{error}</div>
		{/if}

		<div class="dialog-actions">
			<button
				class="button"
				type="button"
				disabled={pending !== null}
				onclick={close}>Cancel</button>
			<TextFilterValidateButton
				{check}
				{text}
				isValidating={pending === "validate"}
				disabled={pending !== null}
				onValidate={() => void validate("validate")}
				onEdit={() => void edit()} />
			<button
				class={check.isTagged ? "button-warning" : "button-primary"}
				type="submit"
				aria-busy={pending === "submit" || pending === "save"}
				disabled={!changed || pending !== null}>
				{#if pending === "submit"}
					Validating…
				{:else if pending === "save"}
					Saving…
				{:else if check.current && !check.current.filter}
					Save without validating
				{:else if check.isTagged}
					Save anyway
				{:else}
					Save description
				{/if}
			</button>
		</div>
	</form>
</Dialog>

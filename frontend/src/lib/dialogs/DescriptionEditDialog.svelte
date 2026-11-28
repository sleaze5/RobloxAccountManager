<script lang="ts">
	import { notifyErrors } from "../notifications/error-notifications.svelte"
	import { tick, untrack } from "svelte"
	import xIcon from "@phosphor-icons/core/regular/x.svg?raw"
	import Icon from "../shared/Icon.svelte"
	import type { Account } from "../accounts/account-model"
	import { accountBackend } from "../backend/bridge"
	import type { NotificationCenter } from "../notifications/notification-center.svelte"
	import { replayValidationNudge } from "../shared/transitions"
	import { errorMessage } from "../shared/errors"
	import { TextFilterCheck } from "../shared/text-filter-check.svelte"
	import TextFilterStatus from "../shared/TextFilterStatus.svelte"
	import TextFilterCheckButton from "../shared/TextFilterCheckButton.svelte"
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

	let typedDescription = $state(untrack(() => description)),
		pending = $state<"check" | "submit" | "save" | null>(null),
		error = $state(""),
		card = $state<HTMLDivElement>(),
		field = $state<HTMLTextAreaElement>()
	const text = $derived(typedDescription.trim()),
		changed = $derived(text !== description.trim()),
		check = new TextFilterCheck(() => text)

	function close(): void {
		if (!pending) onClose()
	}

	async function checkDescription(mode: "check" | "submit"): Promise<boolean> {
		const validatedText = text
		pending = mode
		error = ""
		try {
			return await check.run(validatedText, () =>
				accountBackend.CheckAccountDescription(account.id, validatedText),
			)
		} catch (cause) {
			error = errorMessage(cause, "Could not check the description. Try again.")
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
		} else if (text === "" || check.current || (await checkDescription("submit"))) {
			await save()
		}
	}

	notifyErrors("Description not saved", () => error)
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
				aria-label="Close dialog"
				disabled={pending !== null}
				onclick={close}>
				<Icon svg={xIcon} size={14} aria-hidden="true" />
			</button>
		</div>

		<label class="dialog-field">
			<span>Description</span>
			{#if check.filteredPreview === undefined}
				<textarea
					class="py-1.5 font-sans text-sm"
					bind:this={field}
					bind:value={typedDescription}
					disabled={pending !== null}></textarea>
			{:else}
				<textarea
					class="py-1.5 font-sans text-sm"
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

		<div class="dialog-actions">
			<button
				class="button"
				type="button"
				disabled={pending !== null}
				onclick={close}>Cancel</button>
			<TextFilterCheckButton
				{check}
				{text}
				isChecking={pending === "check"}
				disabled={pending !== null}
				onCheck={() => void checkDescription("check")}
				onEdit={() => void edit()} />
			<button
				class={check.isTagged ? "button-warning" : "button-primary"}
				type="submit"
				aria-busy={pending === "submit" || pending === "save"}
				disabled={!changed || pending !== null}>
				{#if pending === "submit"}
					Checking…
				{:else if pending === "save"}
					Saving…
				{:else if check.current && !check.current.filter}
					Save without checking
				{:else if check.isTagged}
					Save anyway
				{:else}
					Save description
				{/if}
			</button>
		</div>
	</form>
</Dialog>

<script lang="ts">
	import { tick, untrack } from "svelte"
	import CheckCircleIcon from "phosphor-svelte/lib/CheckCircleIcon"
	import XIcon from "phosphor-svelte/lib/XIcon"
	import type { Account } from "../accounts/account-model"
	import { accountBackend, TextModeration } from "../backend/bridge"
	import type { TextFilter } from "../backend/bridge"
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
		validated = $state<{ text: string; filter: TextFilter | null } | null>(null),
		error = $state(""),
		card = $state<HTMLDivElement>(),
		field = $state<HTMLTextAreaElement>()
	const text = $derived(value.trim()),
		changed = $derived(text !== description.trim()),
		result = $derived(validated?.text === text ? validated : null),
		clean = $derived(
			result?.filter?.moderation === TextModeration.TextModerationNone,
		),
		tagged = $derived(
			!!result?.filter &&
				result.filter.moderation !== TextModeration.TextModerationNone,
		),
		preview = $derived(tagged ? result?.filter?.filteredText : undefined)

	function close(): void {
		if (!pending) onClose()
	}

	async function validate(mode: "validate" | "submit"): Promise<boolean> {
		const validatedText = text
		pending = mode
		error = ""
		try {
			const filter = await accountBackend.ValidateAccountDescription(
				account.id,
				validatedText,
			)
			validated = { text: validatedText, filter }
			return filter.moderation === TextModeration.TextModerationNone
		} catch (cause) {
			validated = { text: validatedText, filter: null }
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
		validated = null
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
		} else if (text === "" || result || (await validate("submit"))) {
			await save()
		}
	}
</script>

<svelte:window
	onkeydown={(event) => {
		if (event.key === "Escape") close()
	}} />

<div class="modal-backdrop" in:modalBackdropIn out:modalBackdropOut>
	<div
		class="modal-card location-card"
		role="dialog"
		aria-modal="true"
		aria-labelledby="description-edit-title"
		bind:this={card}
		use:modalFocus
		in:modalCardIn
		out:modalCardOut>
		<form
			novalidate
			onsubmit={(event) => {
				event.preventDefault()
				void submit()
			}}>
			<div class="modal-heading compact-heading">
				<div>
					<h2 id="description-edit-title">Edit description</h2>
					<p>@{account.username}</p>
				</div>
				<button
					type="button"
					aria-label="Close edit description"
					disabled={pending !== null}
					onclick={close}>
					<XIcon size={14} aria-hidden="true" />
				</button>
			</div>

			<label class="modal-field">
				<span>Description</span>
				{#if preview === undefined}
					<textarea bind:this={field} bind:value disabled={pending !== null}
					></textarea>
				{:else}
					<textarea value={preview} readonly disabled={pending !== null}
					></textarea>
				{/if}
			</label>

			<div aria-live="polite">
				{#if changed && text === ""}
					<div class="modal-warning">
						This removes the description from @{account.username}'s profile.
					</div>
				{:else if clean}
					<span class="visually-hidden"
						>Roblox's filter won't tag this text.</span>
				{:else if result?.filter?.moderation === TextModeration.TextModerationPartial}
					<span class="visually-hidden"
						>Roblox will tag part of this text.</span>
				{:else if result?.filter?.moderation === TextModeration.TextModerationFull}
					<span class="visually-hidden"
						>Roblox will tag all of this text.</span>
				{/if}
			</div>

			{#if error}
				<div class="modal-error" role="alert">{error}</div>
			{/if}

			<div class="modal-actions">
				<button
					class="control-button text-button"
					type="button"
					disabled={pending !== null}
					onclick={close}>Cancel</button>
				<button
					class="control-button text-button"
					type="button"
					aria-busy={pending === "validate"}
					disabled={pending !== null ||
						(preview === undefined && (text === "" || result !== null))}
					onclick={() =>
						void (preview === undefined ? validate("validate") : edit())}>
					{#if pending === "validate"}
						Validating…
					{:else if preview !== undefined}
						Edit
					{:else}
						{#if clean}<CheckCircleIcon
								class="validation-passed"
								size={14}
								aria-hidden="true" />{/if}
						Validate
					{/if}
				</button>
				<button
					class={tagged ? "warning-button" : "primary-action"}
					type="submit"
					aria-busy={pending === "submit" || pending === "save"}
					disabled={!changed || pending !== null}>
					{#if pending === "submit"}
						Validating…
					{:else if pending === "save"}
						Saving…
					{:else if result && !result.filter}
						Save without validating
					{:else if tagged}
						Save anyway
					{:else}
						Save description
					{/if}
				</button>
			</div>
		</form>
	</div>
</div>

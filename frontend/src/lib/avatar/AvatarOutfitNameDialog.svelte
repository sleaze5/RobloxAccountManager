<script lang="ts">
	import { notifyErrors } from "../notifications/error-notifications.svelte"
	import xIcon from "@phosphor-icons/core/regular/x.svg?raw"
	import type { AvatarOutfit } from "../backend/bridge"
	import { untrack } from "svelte"
	import Dialog from "../shared/Dialog.svelte"
	import Icon from "../shared/Icon.svelte"
	import { replayValidationNudge } from "../shared/presence"
	import type { AvatarState } from "./avatar-state.svelte"

	let {
		avatar,
		renamedOutfit,
		onClose,
	}: {
		avatar: AvatarState
		// renamedOutfit is null when the dialog saves a new outfit.
		renamedOutfit: AvatarOutfit | null
		onClose: () => void
	} = $props()
	const elementIdPrefix = $props.id()

	const maxNameLength = 100

	let name = $state(untrack(() => renamedOutfit?.name ?? "")),
		nameInput = $state<HTMLInputElement | undefined>(undefined),
		isTouched = $state(false)

	const trimmedName = $derived(name.trim()),
		nameError = $derived(
			trimmedName === ""
				? "Enter a name."
				: trimmedName.length > maxNameLength
					? `Use ${maxNameLength} characters or fewer.`
					: "",
		),
		isWorking = $derived(avatar.isChangingOutfit)

	async function submit(): Promise<void> {
		isTouched = true
		if (nameError) {
			replayValidationNudge(nameInput)
			return
		}
		const isDone = renamedOutfit
			? await avatar.renameOutfit(renamedOutfit, trimmedName)
			: await avatar.createOutfit(trimmedName)
		if (isDone) onClose()
	}

	function close(): void {
		if (!isWorking) onClose()
	}

	notifyErrors("Outfit action not completed", () => avatar.outfitActionError)
</script>

<svelte:window
	onkeydown={(event) => {
		if (event.key === "Escape") {
			event.preventDefault()
			close()
		}
	}} />

<Dialog
	class="max-w-sm"
	aria-labelledby={`${elementIdPrefix}-avatar-outfit-name-title`}
	onBackdropClick={close}>
	<form
		novalidate
		onsubmit={(event) => {
			event.preventDefault()
			void submit()
		}}>
		<div class="dialog-heading">
			<div>
				<h2 id={`${elementIdPrefix}-avatar-outfit-name-title`}>
					{renamedOutfit ? "Rename outfit" : "Save as outfit"}
				</h2>
				<p>
					{renamedOutfit
						? "The outfit keeps its items and body."
						: "Save the current avatar as an outfit to put it on later."}
				</p>
			</div>
			<button
				type="button"
				aria-label="Close dialog"
				disabled={isWorking}
				onclick={close}>
				<Icon svg={xIcon} size={14} aria-hidden="true" />
			</button>
		</div>
		<label class="dialog-field">
			<span>Outfit name</span>
			<input
				bind:this={nameInput}
				bind:value={name}
				type="text"
				autocomplete="off"
				maxlength={maxNameLength}
				aria-invalid={isTouched && !!nameError}
				aria-describedby={`${elementIdPrefix}-avatar-outfit-name-error`}
				disabled={isWorking}
				onblur={() => (isTouched = name !== (renamedOutfit?.name ?? ""))} />
		</label>
		{#if isTouched && nameError}
			<p
				id={`${elementIdPrefix}-avatar-outfit-name-error`}
				class="mt-1 text-xs text-danger-muted">
				{nameError}
			</p>
		{/if}
		<div class="dialog-actions">
			<button class="button" type="button" disabled={isWorking} onclick={close}
				>Cancel</button>
			<button
				class="button-primary"
				type="submit"
				aria-busy={isWorking}
				disabled={isWorking}>
				{#if renamedOutfit}
					{isWorking ? "Renaming…" : "Rename outfit"}
				{:else}
					{isWorking ? "Saving…" : "Save outfit"}
				{/if}
			</button>
		</div>
	</form>
</Dialog>

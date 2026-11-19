<script lang="ts">
	import xIcon from "@phosphor-icons/core/regular/x.svg?raw"
	import Dialog from "../shared/Dialog.svelte"
	import Icon from "../shared/Icon.svelte"

	let {
		title,
		description,
		confirmLabel,
		progressLabel,
		isWorking,
		error,
		onConfirm,
		onCancel,
	}: {
		title: string
		// description names the affected item and the consequence.
		description: string
		confirmLabel: string
		progressLabel: string
		isWorking: boolean
		error: string
		onConfirm: () => void
		onCancel: () => void
	} = $props()

	const dialogId = $props.id()

	function cancel(): void {
		if (!isWorking) onCancel()
	}
</script>

<svelte:window
	onkeydown={(event) => {
		if (event.key === "Escape") {
			event.preventDefault()
			cancel()
		}
	}} />

<Dialog
	class="max-w-sm text-left"
	role="alertdialog"
	aria-labelledby={`${dialogId}-title`}
	aria-describedby={`${dialogId}-description`}>
	<div class="dialog-heading mb-2">
		<div>
			<h2 id={`${dialogId}-title`}>{title}</h2>
			<p id={`${dialogId}-description`}>{description}</p>
		</div>
		<button
			type="button"
			aria-label="Close dialog"
			disabled={isWorking}
			onclick={cancel}>
			<Icon svg={xIcon} size={14} aria-hidden="true" />
		</button>
	</div>
	{#if error}
		<div class="mt-3 notice-error" role="alert">{error}</div>
	{/if}
	<div class="dialog-actions">
		<button class="button" type="button" disabled={isWorking} onclick={cancel}
			>Cancel</button>
		<button
			class="button-danger"
			type="button"
			aria-busy={isWorking}
			disabled={isWorking}
			onclick={onConfirm}>{isWorking ? progressLabel : confirmLabel}</button>
	</div>
</Dialog>

<script lang="ts">
	import xIcon from "@phosphor-icons/core/regular/x.svg?raw"
	import Dialog from "../shared/Dialog.svelte"
	import Icon from "../shared/Icon.svelte"

	let {
		title,
		description,
		actionLabel,
		progressLabel,
		tone,
		isWorking,
		error,
		onConfirm,
		onCancel,
	}: {
		title: string
		description: string
		actionLabel: string
		progressLabel: string
		tone: "danger" | "warning"
		isWorking: boolean
		error: string
		onConfirm: () => void
		onCancel: () => void
	} = $props()

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
	class="max-w-sm"
	role="alertdialog"
	aria-labelledby="avatar-confirmation-title"
	aria-describedby="avatar-confirmation-description"
	onBackdropClick={cancel}>
	<div class="dialog-heading">
		<div>
			<h2 id="avatar-confirmation-title">{title}</h2>
			<p id="avatar-confirmation-description">{description}</p>
		</div>
		<button type="button" aria-label="Close" disabled={isWorking} onclick={cancel}>
			<Icon svg={xIcon} size={14} aria-hidden="true" />
		</button>
	</div>
	{#if error}
		<p class="notice-error" role="alert">{error}</p>
	{/if}
	<div class="dialog-actions">
		<button class="button" type="button" disabled={isWorking} onclick={cancel}
			>Cancel</button>
		<button
			class={tone === "danger" ? "button-danger" : "button-warning"}
			type="button"
			aria-busy={isWorking}
			disabled={isWorking}
			onclick={onConfirm}>{isWorking ? progressLabel : actionLabel}</button>
	</div>
</Dialog>

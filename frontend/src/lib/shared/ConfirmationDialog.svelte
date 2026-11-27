<script lang="ts">
	import xIcon from "@phosphor-icons/core/regular/x.svg?raw"
	import type { Snippet } from "svelte"
	import type { Account } from "../accounts/account-model"
	import Dialog from "./Dialog.svelte"
	import Icon from "./Icon.svelte"
	import RobloxAccountLine from "./RobloxAccountLine.svelte"

	let {
		title,
		description,
		confirmLabel,
		progressLabel,
		tone = "danger",
		account,
		isWorking,
		onConfirm,
		onCancel,
		children,
	}: {
		title: string
		// description names the affected item and the consequence.
		description: string
		confirmLabel: string
		progressLabel: string
		// tone is primary for a change that the user can reverse, such as a
		// change on Roblox.
		tone?: "danger" | "warning" | "primary"
		// account names the saved account that makes the change on Roblox.
		account?: Account
		isWorking: boolean
		onConfirm: () => void
		onCancel: () => void
		// children shows the details of the change, such as each changed
		// setting.
		children?: Snippet
	} = $props()

	const confirmClass = $derived(
		tone === "danger"
			? "button-danger"
			: tone === "warning"
				? "button-warning"
				: "button-primary",
	)

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
	class={[children ? "max-w-md" : "max-w-sm", "text-left"]}
	role="alertdialog"
	aria-labelledby={`${dialogId}-title`}
	aria-describedby={`${dialogId}-description`}
	onBackdropClick={cancel}>
	<div class="dialog-heading mb-2">
		<div class="min-w-0">
			<h2 id={`${dialogId}-title`} class="wrap-anywhere">{title}</h2>
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
	{#if account}<RobloxAccountLine {account} />{/if}
	{#if children}{@render children()}{/if}
	<div class="dialog-actions">
		<button class="button" type="button" disabled={isWorking} onclick={cancel}
			>Cancel</button>
		<button
			class={confirmClass}
			type="button"
			aria-busy={isWorking}
			disabled={isWorking}
			onclick={onConfirm}>{isWorking ? progressLabel : confirmLabel}</button>
	</div>
</Dialog>

<script lang="ts">
	import checkCircleIcon from "@phosphor-icons/core/regular/check-circle.svg?raw"
	import checkIcon from "@phosphor-icons/core/regular/check.svg?raw"
	import copyIcon from "@phosphor-icons/core/regular/copy.svg?raw"
	import eyeIcon from "@phosphor-icons/core/regular/eye.svg?raw"
	import eyeSlashIcon from "@phosphor-icons/core/regular/eye-slash.svg?raw"
	import xIcon from "@phosphor-icons/core/regular/x.svg?raw"
	import Icon from "../shared/Icon.svelte"
	import { accountBackend, type ExportInfo } from "../backend/bridge"
	import {
		modalBackdropIn,
		modalBackdropOut,
		modalCardIn,
		modalCardOut,
		modalFocus,
		replayValidationNudge,
	} from "../shared/presence"
	import { errorMessage } from "../shared/errors"
	import { checkPassword, passwordValid } from "./password-rules"

	let {
			directory,
			onCopyPath,
			onClose,
		}: {
			directory: string
			onCopyPath: () => void
			onClose: () => void
		} = $props(),
		passcode = $state(""),
		confirmation = $state(""),
		visible = $state(false),
		exporting = $state(false),
		error = $state(""),
		exported = $state<ExportInfo | null>(null),
		card = $state<HTMLDivElement>()
	const rules = $derived(checkPassword(passcode)),
		requirements = $derived<[boolean, string][]>([
			[rules.length, "8-128 characters long"],
			[rules.uppercase, "At least one upper-case letter"],
			[rules.lowercase, "At least one lower-case letter"],
			[rules.digit, "At least one number"],
			[rules.special, "At least one special character"],
			[passcode !== "" && passcode === confirmation, "Passcodes match"],
		])

	function close(): void {
		if (!exporting) onClose()
	}

	async function exportData(): Promise<void> {
		exporting = true
		error = ""
		try {
			exported = await accountBackend.ExportVault(passcode, confirmation)
			passcode = confirmation = ""
		} catch (cause) {
			error = errorMessage(cause, "The data could not be exported. Try again.")
			replayValidationNudge(card)
		} finally {
			exporting = false
		}
	}
</script>

<svelte:window
	onkeydown={(event) => {
		if (event.key === "Escape") {
			event.preventDefault()
			close()
		}
	}} />

<div
	class="dialog-backdrop"
	in:modalBackdropIn
	out:modalBackdropOut
	onclick={(event) => {
		if (event.target === event.currentTarget) close()
	}}
	role="presentation">
	<div
		class="dialog-card max-w-md"
		role="dialog"
		aria-modal="true"
		aria-labelledby="export-vault-title"
		aria-busy={exporting}
		bind:this={card}
		use:modalFocus
		in:modalCardIn
		out:modalCardOut>
		<div class="dialog-heading">
			<div>
				<h2 id="export-vault-title">
					{exported ? "Data exported" : "Export vault data"}
				</h2>
				<p>
					{exported
						? "Keep the file and its passcode in separate places."
						: "Save accounts, saved passwords, tags, favorite places, and settings to an encrypted file."}
				</p>
			</div>
			<button
				type="button"
				aria-label="Close export dialog"
				disabled={exporting}
				onclick={close}>
				<Icon svg={xIcon} size={14} aria-hidden="true" />
			</button>
		</div>
		{#if exported}
			<div
				class="flex animate-enter items-center gap-2 rounded-md border border-line bg-fill p-2"
				role="status">
				<span class="animate-bloom text-success">
					<Icon svg={checkCircleIcon} size={16} aria-hidden="true" />
				</span>
				<span class="flex min-w-0 flex-1 flex-col">
					<span class="font-mono text-2xs wrap-anywhere text-fg"
						>{exported.name}</span>
					<span class="font-mono text-2xs wrap-anywhere text-fg-faint"
						>{directory}</span>
				</span>
				<button
					class="btn-icon"
					type="button"
					aria-label="Copy folder path"
					data-tooltip="Copy folder path"
					onclick={onCopyPath}>
					<Icon svg={copyIcon} size={16} aria-hidden="true" />
				</button>
			</div>
			<p class="mt-3 text-xs text-fg-muted">
				To import it on another computer, copy it into that app's export folder.
			</p>
			<div class="dialog-actions">
				<button class="btn-primary" type="button" onclick={onClose}
					>Done</button>
			</div>
		{:else}
			<form
				novalidate
				onsubmit={(event) => {
					event.preventDefault()
					void exportData()
				}}>
				<div class="notice-warning">
					Anyone with this file and its passcode can sign in to your accounts.
					The passcode cannot be recovered.
				</div>
				<label class="dialog-field">
					<span>Passcode</span>
					<input
						bind:value={passcode}
						type={visible ? "text" : "password"}
						autocomplete="new-password"
						disabled={exporting}
						required />
				</label>
				<label class="dialog-field">
					<span>Confirm passcode</span>
					<input
						bind:value={confirmation}
						type={visible ? "text" : "password"}
						autocomplete="new-password"
						disabled={exporting}
						required />
				</label>
				<button
					class="mt-2 flex items-center gap-1 py-1 text-xs text-fg-muted transition-colors duration-100 *:animate-pop"
					type="button"
					onclick={() => (visible = !visible)}>
					{#if visible}<Icon
							svg={eyeSlashIcon}
							size={16}
							aria-hidden="true" /> Hide passcodes{:else}<Icon
							svg={eyeIcon}
							size={16}
							aria-hidden="true" /> Show passcodes{/if}
				</button>
				<ul class="checklist" data-touched={passcode !== "" || undefined}>
					{#each requirements as [valid, label] (label)}
						<li data-valid={valid || undefined}>
							{#if valid}<Icon
									svg={checkIcon}
									size={14}
									aria-hidden="true" />{:else}<Icon
									svg={xIcon}
									size={14}
									aria-hidden="true" />{/if}{label}
						</li>
					{/each}
				</ul>
				{#if error}
					<div class="mt-3 notice-error" role="alert">{error}</div>
				{/if}
				<div class="dialog-actions">
					<button
						class="btn"
						type="button"
						disabled={exporting}
						onclick={close}>Cancel</button>
					<button
						class="btn-primary"
						type="submit"
						disabled={!passwordValid(passcode) ||
							passcode !== confirmation ||
							exporting}
						>{exporting ? "Exporting…" : "Export data"}</button>
				</div>
			</form>
		{/if}
	</div>
</div>

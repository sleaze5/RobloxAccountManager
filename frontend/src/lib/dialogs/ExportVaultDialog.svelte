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
	import type { NotificationCenter } from "../notifications/notification-center.svelte"
	import { checkPassword, passwordValid } from "./password-rules"
	import { copyPath, fileName } from "./vault-transfer"

	let {
			notifications,
			onClose,
		}: {
			notifications: NotificationCenter
			onClose: () => void
		} = $props(),
		password = $state(""),
		confirmation = $state(""),
		hint = $state(""),
		visible = $state(false),
		exporting = $state(false),
		error = $state(""),
		exported = $state<ExportInfo | null>(null),
		card = $state<HTMLDivElement>()
	const rules = $derived(checkPassword(password)),
		requirements = $derived<[boolean, string][]>([
			[rules.length, "8-128 characters long"],
			[rules.uppercase, "At least one upper-case letter"],
			[rules.lowercase, "At least one lower-case letter"],
			[rules.digit, "At least one number"],
			[rules.special, "At least one special character"],
			[password !== "" && password === confirmation, "Passwords match"],
		]),
		hintHasPassword = $derived(
			hint !== "" &&
				password !== "" &&
				hint.toLowerCase().includes(password.toLowerCase()),
		)

	function close(): void {
		if (!exporting) onClose()
	}

	async function exportData(): Promise<void> {
		exporting = true
		error = ""
		try {
			const saved = await accountBackend.ExportVault(password, confirmation, hint)
			if (!saved.path) return
			exported = saved
			password = confirmation = hint = ""
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
		class="dialog-card max-w-lg"
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
						? "Keep the file and its export password in separate places."
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
						>{exported.path.slice(
							0,
							Math.max(
								exported.path.length -
									fileName(exported.path).length -
									1,
								0,
							),
						)}</span>
				</span>
				<button
					class="btn-icon"
					type="button"
					aria-label="Copy file path"
					data-tooltip="Copy file path"
					onclick={() => void copyPath(exported?.path ?? "", notifications)}>
					<Icon svg={copyIcon} size={16} aria-hidden="true" />
				</button>
			</div>
			<p class="mt-3 text-xs text-fg-muted">
				To import it on another computer, choose Import data there and open this
				file.
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
					Anyone with this file and its export password can sign in to your
					accounts. The export password cannot be recovered.
				</div>
				<div class="grid grid-cols-2 gap-3">
					<label class="dialog-field">
						<span>Export password</span>
						<input
							bind:value={password}
							type={visible ? "text" : "password"}
							autocomplete="new-password"
							disabled={exporting}
							required />
					</label>
					<label class="dialog-field">
						<span>Confirm export password</span>
						<input
							bind:value={confirmation}
							type={visible ? "text" : "password"}
							autocomplete="new-password"
							disabled={exporting}
							required />
					</label>
				</div>
				<button
					class="mt-2 flex items-center gap-1 py-1 text-xs text-fg-muted transition-colors duration-100 *:animate-pop"
					type="button"
					aria-pressed={visible}
					onclick={() => (visible = !visible)}>
					{#if visible}<Icon
							svg={eyeSlashIcon}
							size={16}
							aria-hidden="true" /> Hide passwords{:else}<Icon
							svg={eyeIcon}
							size={16}
							aria-hidden="true" /> Show passwords{/if}
				</button>
				<ul class="checklist" data-touched={password !== "" || undefined}>
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
				<label class="dialog-field">
					<span>Password hint (optional)</span>
					<input
						bind:value={hint}
						maxlength="120"
						autocomplete="off"
						aria-invalid={hintHasPassword || undefined}
						aria-describedby={hintHasPassword
							? "export-hint-error"
							: undefined}
						disabled={exporting} />
				</label>
				{#if hintHasPassword}
					<p id="export-hint-error" class="mt-1 text-xs text-danger-muted">
						The hint cannot contain the export password.
					</p>
				{:else}
					<p class="mt-1 text-2xs text-fg-muted">
						Anyone with the file can read the hint.
					</p>
				{/if}
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
						disabled={!passwordValid(password) ||
							password !== confirmation ||
							hintHasPassword ||
							exporting}
						>{exporting ? "Exporting…" : "Export data"}</button>
				</div>
			</form>
		{/if}
	</div>
</div>

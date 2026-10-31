<script lang="ts">
	import XIcon from "phosphor-svelte/lib/XIcon"
	import { accountBackend } from "../backend/bridge"
	import type { StorageCandidate } from "../backend/bridge"
	import {
		modalBackdropIn,
		modalBackdropOut,
		modalCardIn,
		modalCardOut,
		modalFocus,
	} from "../shared/presence"
	import { errorMessage } from "../shared/errors"

	let {
			target,
			targetLabel,
			onClose,
		}: { target: StorageCandidate; targetLabel: string; onClose: () => void } =
			$props(),
		busy = $state(false),
		error = $state("")

	async function move(): Promise<void> {
		busy = true
		error = ""
		try {
			await accountBackend.MoveAppData(target.mode)
		} catch (cause) {
			error = errorMessage(cause, "The data could not be moved.")
			busy = false
		}
	}

	function handleKeydown(event: KeyboardEvent): void {
		if (event.key === "Escape" && !busy) onClose()
	}
</script>

<svelte:window onkeydown={handleKeydown} />

<div class="dialog-backdrop" in:modalBackdropIn out:modalBackdropOut>
	<div
		class="dialog-card max-w-sm text-left"
		role="alertdialog"
		aria-modal="true"
		aria-labelledby="move-data-title"
		aria-describedby="move-data-description"
		use:modalFocus
		in:modalCardIn
		out:modalCardOut>
		<div class="dialog-heading">
			<div>
				<h2 id="move-data-title">Move to {targetLabel} storage?</h2>
				<p id="move-data-description">
					The app restarts and moves the vault, backups, and settings to this
					folder:
				</p>
			</div>
			<button
				type="button"
				aria-label="Close move data dialog"
				disabled={busy}
				onclick={onClose}>
				<XIcon size={14} aria-hidden="true" />
			</button>
		</div>
		<div class="mt-2 font-mono text-2xs wrap-anywhere text-fg-faint">
			{target.directory}
		</div>
		<ul class="mt-3 flex list-disc flex-col gap-1 pl-5 text-xs text-warning">
			<li>Every file is verified before the old copy is deleted.</li>
			<li>Open browser sessions close. The browser downloads again if needed.</li>
		</ul>
		{#if error}<div class="mt-3 notice-error" role="alert">{error}</div>{/if}
		<div class="dialog-actions">
			<button class="btn" type="button" disabled={busy} onclick={onClose}
				>Cancel</button
			><button
				class="btn-primary"
				type="button"
				disabled={busy}
				onclick={() => void move()}
				>{busy ? "Restarting…" : "Move and restart"}</button>
		</div>
	</div>
</div>

<script lang="ts">
	import { accountBackend } from "../backend/bridge"
	import type { StorageCandidate } from "../backend/bridge"
	import {
		modalBackdropIn,
		modalBackdropOut,
		modalCardIn,
		modalCardOut,
		modalFocus,
	} from "../shared/presence"

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
			error =
				cause instanceof Error ? cause.message : "The data could not be moved."
			busy = false
		}
	}

	function handleKeydown(event: KeyboardEvent): void {
		if (event.key === "Escape" && !busy) onClose()
	}
</script>

<svelte:window onkeydown={handleKeydown} />

<div class="modal-backdrop" in:modalBackdropIn out:modalBackdropOut>
	<div
		class="modal-card confirm-card"
		role="alertdialog"
		aria-modal="true"
		aria-labelledby="move-data-title"
		aria-describedby="move-data-description"
		use:modalFocus
		in:modalCardIn
		out:modalCardOut>
		<div class="modal-heading">
			<div>
				<h2 id="move-data-title">Move to {targetLabel} storage?</h2>
				<p id="move-data-description">
					The app restarts and moves the vault, backups, and settings to this
					folder:
				</p>
			</div>
		</div>
		<div class="location-option-path">{target.directory}</div>
		<ul class="confirmation-effects">
			<li>Every file is verified before the old copy is deleted.</li>
			<li>Open browser sessions close. The browser downloads again if needed.</li>
		</ul>
		{#if error}<div class="modal-error" role="alert">{error}</div>{/if}
		<div class="modal-actions">
			<button type="button" disabled={busy} onclick={onClose}>Cancel</button
			><button
				class="danger-button"
				type="button"
				disabled={busy}
				onclick={() => void move()}
				>{busy ? "Restarting..." : "Move and restart"}</button>
		</div>
	</div>
</div>

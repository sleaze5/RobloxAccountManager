<script lang="ts">
	import type { AccountStore } from "../accounts/account-store.svelte"
	import type { ShutdownEffects } from "../backend/bridge"
	import {
		modalBackdropIn,
		modalBackdropOut,
		modalCardIn,
		modalCardOut,
		modalFocus,
	} from "../shared/presence"

	let {
		store,
		effects,
		onClose,
	}: { store: AccountStore; effects: ShutdownEffects; onClose: () => void } = $props()
	const effectLines = $derived(
		[
			effects.browserSessions > 0
				? `Close ${effects.browserSessions} browser ${effects.browserSessions === 1 ? "session" : "sessions"}.`
				: "",
			effects.browserDownloads ? "Stop browser downloads." : "",
			effects.accountCandidates > 0
				? `Discard ${effects.accountCandidates} account ${effects.accountCandidates === 1 ? "candidate" : "candidates"}.`
				: "",
			effects.accountChecks ? "Cancel account checks." : "",
		].filter(Boolean),
	)

	async function lock(): Promise<void> {
		await store.lockVault()
		if (!store.vault.unlocked) onClose()
	}
</script>

<div class="modal-backdrop" in:modalBackdropIn out:modalBackdropOut>
	<section
		class="modal-card confirm-card"
		use:modalFocus
		in:modalCardIn
		out:modalCardOut
		aria-labelledby="lock-title">
		<div class="modal-heading">
			<div>
				<h2 id="lock-title">Lock vault?</h2>
				<p>
					You will need your master password to unlock it.{store.vault
						.automaticUnlock
						? " Automatic unlock will be disabled."
						: ""}
				</p>
			</div>
		</div>
		{#if store.error}<div class="modal-error" role="alert">{store.error}</div>{/if}
		{#if effectLines.length > 0}<ul class="confirmation-effects">
				{#each effectLines as effect}<li>{effect}</li>{/each}
			</ul>{/if}
		<div class="modal-actions">
			<button type="button" disabled={store.busy} onclick={onClose}>Cancel</button
			><button
				class="danger-button"
				type="button"
				disabled={store.busy}
				onclick={() => void lock()}
				>{store.busy ? "Locking..." : "Lock vault"}</button>
		</div>
	</section>
</div>

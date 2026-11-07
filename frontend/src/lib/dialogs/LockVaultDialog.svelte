<script lang="ts">
	import xIcon from "@phosphor-icons/core/regular/x.svg?raw"
	import Icon from "../shared/Icon.svelte"
	import Dialog from "../shared/Dialog.svelte"
	import type { AccountStore } from "../accounts/account-store.svelte"
	import type { ShutdownEffects } from "../backend/bridge"

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

<Dialog class="max-w-sm text-left" role="alertdialog" aria-labelledby="lock-title">
	<div class="dialog-heading">
		<div>
			<h2 id="lock-title">Lock vault?</h2>
			<p>
				You will need your master password to unlock it.{store.vault
					.automaticUnlock
					? " Automatic unlock will be disabled."
					: ""}
			</p>
		</div>
		<button
			type="button"
			aria-label="Close lock vault dialog"
			disabled={store.busy}
			onclick={onClose}>
			<Icon svg={xIcon} size={14} aria-hidden="true" />
		</button>
	</div>
	{#if store.error}<div class="mt-3 notice-error" role="alert">
			{store.error}
		</div>{/if}
	{#if effectLines.length > 0}<ul
			class="mt-3 flex list-disc flex-col gap-1 pl-5 text-xs text-warning">
			{#each effectLines as effect}<li>{effect}</li>{/each}
		</ul>{/if}
	<div class="dialog-actions">
		<button class="button" type="button" disabled={store.busy} onclick={onClose}
			>Cancel</button
		><button
			class="button-primary"
			type="button"
			disabled={store.busy}
			onclick={() => void lock()}
			>{store.busy ? "Locking…" : "Lock vault"}</button>
	</div>
</Dialog>

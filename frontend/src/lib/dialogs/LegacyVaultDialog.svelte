<script lang="ts">
	import arrowSquareOutIcon from "@phosphor-icons/core/regular/arrow-square-out.svg?raw"
	import lockKeyIcon from "@phosphor-icons/core/regular/lock-key.svg?raw"
	import { Browser } from "@wailsio/runtime"
	import Icon from "../shared/Icon.svelte"
	import type { AccountStore } from "../accounts/account-store.svelte"
	import {
		modalBackdropIn,
		modalBackdropOut,
		modalCardIn,
		modalCardOut,
		modalFocus,
		replayValidationNudge,
	} from "../shared/presence"
	import { repository } from "../../../package.json"

	let { store }: { store: AccountStore } = $props(),
		password = $state(""),
		confirmation = $state(""),
		manual = $state(false),
		card = $state<HTMLDivElement>()
	const bridgeReleaseURL = `${repository.url.replace(/^git\+/, "").replace(/\.git$/, "")}/releases/tag/v0.9.2`,
		migrating = $derived(store.vault.migrationAvailable && !manual)

	async function migrate(): Promise<void> {
		if (await store.migrateLegacyVault(password)) password = ""
		else replayValidationNudge(card)
	}

	async function setAside(): Promise<void> {
		if (await store.setAsideLegacyVault(confirmation)) confirmation = ""
	}
</script>

<div
	class="dialog-backdrop z-50 bg-scrim-strong"
	in:modalBackdropIn
	out:modalBackdropOut>
	<div
		class="relative dialog-card max-w-md text-left"
		role="dialog"
		aria-modal="true"
		aria-labelledby="legacy-vault-title"
		bind:this={card}
		use:modalFocus
		in:modalCardIn
		out:modalCardOut>
		<div
			class="mx-auto mb-3 grid size-8 place-items-center rounded-md border border-line-strong bg-raised text-fg">
			<Icon svg={lockKeyIcon} size={16} aria-hidden="true" />
		</div>
		{#if migrating}
			<h2 id="legacy-vault-title" class="text-center">
				Move your vault to the new format
			</h2>
			<p class="text-center">
				Version 0.9.2 saved your data before the update. Enter your master
				password to move it into a new vault.
			</p>
			{#if store.error}
				<div class="mt-3 notice-error" role="alert">{store.error}</div>
			{/if}
			<form
				novalidate
				onsubmit={(event) => {
					event.preventDefault()
					void migrate()
				}}>
				<label class="dialog-field">
					<span>Master password</span>
					<input
						bind:value={password}
						type="password"
						autocomplete="current-password"
						required />
				</label>
				{#if store.vault.passwordHint}
					<div class="mt-3 notice-note">Hint: {store.vault.passwordHint}</div>
				{/if}
				<button
					class="mt-3 btn-primary h-8 w-full"
					type="submit"
					disabled={!password || store.busy}>
					{store.busy ? "Moving data…" : "Move data"}
				</button>
			</form>
			<button
				class="mt-2 flex items-center gap-1 py-1 text-xs text-fg-muted transition-colors duration-100"
				type="button"
				disabled={store.busy}
				onclick={() => (manual = true)}>Other options</button>
		{:else}
			<h2 id="legacy-vault-title" class="text-center">
				Vault from an older version
			</h2>
			<p class="text-center">
				This vault uses the format from versions before 0.10.0. This version
				cannot open it.
			</p>
			<ul class="mt-3 flex list-disc flex-col gap-1 pl-5 text-xs text-fg-muted">
				<li>
					Download version 0.9.2, replace this app with it, and update from
					Settings > About. Your data moves over automatically.
				</li>
				<li>
					Or, if you have an export file, set the old vault aside below and
					create a new vault. Then open Add account, choose Import export
					file, and select your file.
				</li>
			</ul>
			<button
				class="mt-3 btn w-full"
				type="button"
				onclick={() => void Browser.OpenURL(bridgeReleaseURL)}>
				<Icon svg={arrowSquareOutIcon} size={16} aria-hidden="true" />
				Download version 0.9.2
			</button>
			{#if store.error}
				<div class="mt-3 notice-error" role="alert">{store.error}</div>
			{/if}
			<div class="mt-3 notice-warning">
				Set the old vault aside only if you have an export file. It moves to the
				recovery folder in storage/vault and is not deleted.
			</div>
			<form
				novalidate
				onsubmit={(event) => {
					event.preventDefault()
					void setAside()
				}}>
				<label class="dialog-field">
					<span>Type CONFIRM to confirm</span>
					<input bind:value={confirmation} autocomplete="off" />
				</label>
				<button
					class="mt-3 btn-warning w-full"
					type="submit"
					disabled={confirmation !== "CONFIRM" || store.busy}>
					{store.busy ? "Setting aside…" : "Set aside old vault"}
				</button>
			</form>
			{#if store.vault.migrationAvailable}
				<button
					class="mt-2 flex items-center gap-1 py-1 text-xs text-fg-muted transition-colors duration-100"
					type="button"
					disabled={store.busy}
					onclick={() => (manual = false)}>Back to moving your data</button>
			{/if}
		{/if}
	</div>
</div>

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
	} from "../shared/presence"
	import { repository } from "../../../package.json"

	let { store }: { store: AccountStore } = $props(),
		confirmation = $state("")
	const exportReleaseURL = `${repository.url.replace(/^git\+/, "").replace(/\.git$/, "")}/releases/tag/v0.9.1`

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
		use:modalFocus
		in:modalCardIn
		out:modalCardOut>
		<div
			class="mx-auto mb-3 grid size-8 place-items-center rounded-md border border-line-strong bg-raised text-fg">
			<Icon svg={lockKeyIcon} size={16} aria-hidden="true" />
		</div>
		<h2 id="legacy-vault-title" class="text-center">Vault from an older version</h2>
		<p class="text-center">
			This vault uses the format from versions before 0.10.0. This version cannot
			open it.
		</p>
		<ol class="mt-3 flex list-decimal flex-col gap-1 pl-5 text-xs text-fg-muted">
			<li>
				If you have not exported your data, download version 0.9.1, replace this
				app with it, and export your data.
			</li>
			<li>
				Set the old vault aside. It moves to the recovery folder in
				storage/vault and is not deleted.
			</li>
			<li>
				Create a new vault. Then open Add account, choose Import export file,
				and select your file.
			</li>
		</ol>
		<button
			class="mt-3 btn w-full"
			type="button"
			onclick={() => void Browser.OpenURL(exportReleaseURL)}>
			<Icon svg={arrowSquareOutIcon} size={16} aria-hidden="true" />
			Download version 0.9.1
		</button>
		{#if store.error}
			<div class="mt-3 notice-error" role="alert">{store.error}</div>
		{/if}
		<div class="mt-3 notice-warning">
			Set the old vault aside only after you export your data.
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
	</div>
</div>

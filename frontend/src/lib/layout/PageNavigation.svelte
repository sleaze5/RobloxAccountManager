<script lang="ts">
	import FileSearch from "@lucide/svelte/icons/file-search"
	import Gamepad2 from "@lucide/svelte/icons/gamepad-2"
	import LoaderCircle from "@lucide/svelte/icons/loader-circle"
	import LockKeyhole from "@lucide/svelte/icons/lock-keyhole"
	import Settings from "@lucide/svelte/icons/settings"
	import Users from "@lucide/svelte/icons/users"
	import type { AccountStore } from "../accounts/account-store.svelte"

	let {
		store,
		activePage,
		onPage,
		onSettings,
		onLock,
	}: {
		store: AccountStore
		activePage: "accounts" | "games" | "logs-explorer"
		onPage: (page: "accounts" | "games" | "logs-explorer") => void
		onSettings: () => void
		onLock: () => void
	} = $props()
</script>

<header class="page-tabs">
	<nav aria-label="Page navigation">
		<button
			class:active={activePage === "accounts"}
			type="button"
			aria-current={activePage === "accounts" ? "page" : undefined}
			onclick={() => onPage("accounts")}>
			<Users size={15} aria-hidden="true" />
			<span>Accounts</span>
		</button>
		<button
			class:active={activePage === "games"}
			type="button"
			aria-current={activePage === "games" ? "page" : undefined}
			onclick={() => onPage("games")}>
			<Gamepad2 size={15} aria-hidden="true" />
			<span>Games</span>
		</button>
		<button
			class:active={activePage === "logs-explorer"}
			type="button"
			aria-current={activePage === "logs-explorer" ? "page" : undefined}
			onclick={() => onPage("logs-explorer")}>
			<FileSearch size={15} aria-hidden="true" />
			<span>Logs Explorer</span>
		</button>
	</nav>
	<div class="page-actions">
		<button
			class="settings-action"
			type="button"
			aria-label="Lock account vault"
			data-tooltip="Lock account vault"
			data-tooltip-side="bottom-end"
			disabled={!store.vault.unlocked || store.busy}
			onclick={onLock}>
			{#if store.busy}<LoaderCircle
					class="spinner"
					size={15} />{:else}<LockKeyhole size={15} />{/if}
		</button>
		<button
			type="button"
			aria-label="Settings"
			data-tooltip="Settings"
			data-tooltip-side="bottom-end"
			onclick={onSettings}>
			<Settings size={15} aria-hidden="true" />
		</button>
	</div>
</header>

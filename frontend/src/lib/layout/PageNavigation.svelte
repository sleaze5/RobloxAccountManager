<script lang="ts">
	import FileMagnifyingGlassIcon from "phosphor-svelte/lib/FileMagnifyingGlassIcon"
	import GameControllerIcon from "phosphor-svelte/lib/GameControllerIcon"
	import CircleNotchIcon from "phosphor-svelte/lib/CircleNotchIcon"
	import LockKeyIcon from "phosphor-svelte/lib/LockKeyIcon"
	import GearIcon from "phosphor-svelte/lib/GearIcon"
	import UsersIcon from "phosphor-svelte/lib/UsersIcon"
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
			<UsersIcon size={17} aria-hidden="true" />
			<span>Accounts</span>
		</button>
		<button
			class:active={activePage === "games"}
			type="button"
			aria-current={activePage === "games" ? "page" : undefined}
			onclick={() => onPage("games")}>
			<GameControllerIcon size={17} aria-hidden="true" />
			<span>Games</span>
		</button>
		<button
			class:active={activePage === "logs-explorer"}
			type="button"
			aria-current={activePage === "logs-explorer" ? "page" : undefined}
			onclick={() => onPage("logs-explorer")}>
			<FileMagnifyingGlassIcon size={17} aria-hidden="true" />
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
			{#if store.busy}<CircleNotchIcon
					class="spinner"
					size={17}
					aria-hidden="true" />{:else}<LockKeyIcon
					size={17}
					aria-hidden="true" />{/if}
		</button>
		<button
			type="button"
			aria-label="Settings"
			data-tooltip="Settings"
			data-tooltip-side="bottom-end"
			onclick={onSettings}>
			<GearIcon size={17} aria-hidden="true" />
		</button>
	</div>
</header>

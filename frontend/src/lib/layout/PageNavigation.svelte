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
	const tabClass =
			"relative flex shrink-0 items-center justify-center gap-2 border-r border-line-subtle px-4 text-xs font-medium whitespace-nowrap transition-colors duration-100 hover:text-fg",
		actionClass =
			"flex w-8 items-center justify-center text-fg-muted transition-colors duration-100 hover:text-fg"
</script>

<header
	class="flex h-9 shrink-0 items-stretch border-b border-line-subtle bg-surface pr-3">
	<nav class="flex items-stretch" aria-label="Page navigation">
		<button
			class={[
				tabClass,
				activePage === "accounts" ? "text-fg shadow-tab" : "text-fg-muted",
			]}
			type="button"
			aria-current={activePage === "accounts" ? "page" : undefined}
			onclick={() => onPage("accounts")}>
			<UsersIcon size={16} aria-hidden="true" />
			<span>Accounts</span>
		</button>
		<button
			class={[
				tabClass,
				activePage === "games" ? "text-fg shadow-tab" : "text-fg-muted",
			]}
			type="button"
			aria-current={activePage === "games" ? "page" : undefined}
			onclick={() => onPage("games")}>
			<GameControllerIcon size={16} aria-hidden="true" />
			<span>Games</span>
		</button>
		<button
			class={[
				tabClass,
				activePage === "logs-explorer" ? "text-fg shadow-tab" : "text-fg-muted",
			]}
			type="button"
			aria-current={activePage === "logs-explorer" ? "page" : undefined}
			onclick={() => onPage("logs-explorer")}>
			<FileMagnifyingGlassIcon size={16} aria-hidden="true" />
			<span>Logs Explorer</span>
		</button>
	</nav>
	<div class="ml-auto flex">
		<button
			class={actionClass}
			type="button"
			aria-label="Lock account vault"
			data-tooltip="Lock account vault"
			data-tooltip-side="bottom-end"
			disabled={!store.vault.unlocked || store.busy}
			onclick={onLock}>
			{#if store.busy}<CircleNotchIcon
					class="spinner"
					size={16}
					aria-hidden="true" />{:else}<LockKeyIcon
					class="animate-pop"
					size={16}
					aria-hidden="true" />{/if}
		</button>
		<button
			class={actionClass}
			type="button"
			aria-label="Settings"
			data-tooltip="Settings"
			data-tooltip-side="bottom-end"
			onclick={onSettings}>
			<GearIcon class="animate-pop" size={16} aria-hidden="true" />
		</button>
	</div>
</header>

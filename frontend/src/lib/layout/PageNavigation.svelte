<script lang="ts">
	import circleNotchIcon from "@phosphor-icons/core/regular/circle-notch.svg?raw"
	import fileMagnifyingGlassIcon from "@phosphor-icons/core/regular/file-magnifying-glass.svg?raw"
	import gameControllerIcon from "@phosphor-icons/core/regular/game-controller.svg?raw"
	import gearIcon from "@phosphor-icons/core/regular/gear.svg?raw"
	import handshakeIcon from "@phosphor-icons/core/regular/handshake.svg?raw"
	import lockKeyIcon from "@phosphor-icons/core/regular/lock-key.svg?raw"
	import usersIcon from "@phosphor-icons/core/regular/users.svg?raw"
	import Icon from "../shared/Icon.svelte"
	import type { AccountStore } from "../accounts/account-store.svelte"

	let {
		store,
		activePage,
		onPage,
		onSettings,
		onLock,
	}: {
		store: AccountStore
		activePage: "accounts" | "social" | "games" | "logs-explorer"
		onPage: (page: "accounts" | "social" | "games" | "logs-explorer") => void
		onSettings: () => void
		onLock: () => void
	} = $props()
	const actionClass =
		"flex w-8 items-center justify-center text-fg-muted transition-colors duration-100 hover:text-fg"
</script>

<header
	class="flex h-9 shrink-0 items-stretch border-b border-line-subtle bg-surface pr-3">
	<nav class="flex items-stretch" aria-label="Page navigation">
		<button
			class="tab"
			type="button"
			aria-current={activePage === "accounts" ? "page" : undefined}
			onclick={() => onPage("accounts")}>
			<Icon svg={usersIcon} size={16} aria-hidden="true" />
			<span>Accounts</span>
		</button>
		<button
			class="tab"
			type="button"
			aria-current={activePage === "social" ? "page" : undefined}
			onclick={() => onPage("social")}>
			<Icon svg={handshakeIcon} size={16} aria-hidden="true" />
			<span>Social</span>
		</button>
		<button
			class="tab"
			type="button"
			aria-current={activePage === "games" ? "page" : undefined}
			onclick={() => onPage("games")}>
			<Icon svg={gameControllerIcon} size={16} aria-hidden="true" />
			<span>Games</span>
		</button>
		<button
			class="tab"
			type="button"
			aria-current={activePage === "logs-explorer" ? "page" : undefined}
			onclick={() => onPage("logs-explorer")}>
			<Icon svg={fileMagnifyingGlassIcon} size={16} aria-hidden="true" />
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
			{#if store.busy}<Icon
					svg={circleNotchIcon}
					class="spinner"
					size={16}
					aria-hidden="true" />{:else}<Icon
					svg={lockKeyIcon}
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
			<Icon svg={gearIcon} class="animate-pop" size={16} aria-hidden="true" />
		</button>
	</div>
</header>

<script lang="ts">
	import AtSign from "@lucide/svelte/icons/at-sign"
	import ChevronRight from "@lucide/svelte/icons/chevron-right"
	import Focus from "@lucide/svelte/icons/focus"
	import Gamepad2 from "@lucide/svelte/icons/gamepad-2"
	import Globe2 from "@lucide/svelte/icons/earth"
	import Hash from "@lucide/svelte/icons/hash"
	import KeyRound from "@lucide/svelte/icons/key-round"
	import RefreshCw from "@lucide/svelte/icons/refresh-cw"
	import Tags from "@lucide/svelte/icons/tags"
	import Trash2 from "@lucide/svelte/icons/trash-2"
	import UserRound from "@lucide/svelte/icons/user-round"
	import X from "@lucide/svelte/icons/x"
	import { browserStore } from "../browser/browser-store.svelte"
	import type { AccountStore } from "./account-store.svelte"
	import type { Account } from "./account-model"
	import TagPicker from "./TagPicker.svelte"
	import { AccountCopyField, LaunchMethod, PresenceType } from "../backend/bridge"
	import type { WorkspaceState } from "../layout/workspace-state.svelte"

	let {
		store,
		workspace,
		account,
		submenuLeft,
		onManageTags,
		onRemove,
		onRenewCookie,
		onBrowser,
	}: {
		store: AccountStore
		workspace: WorkspaceState
		account: Account
		submenuLeft: boolean
		onManageTags: (accountId: number) => void
		onRemove: (accountId: number) => void
		onRenewCookie: (accountId: number) => void
		onBrowser: (accountId: number | null) => void
	} = $props()
	const sessionId = $derived(browserStore.accountSessionId(account.id)),
		browserOpen = $derived(!!sessionId)
	const copyFields = [
		{
			field: AccountCopyField.CopyDisplayName,
			label: "Display name",
			icon: UserRound,
		},
		{ field: AccountCopyField.CopyUsername, label: "Username", icon: AtSign },
		{ field: AccountCopyField.CopyUserID, label: "User ID", icon: Hash },
	]

	async function copyField(field: AccountCopyField): Promise<void> {
		if (await store.copyAccountFields([account.id], field)) {
			workspace.closeAccountMenus()
		}
	}

	function fillJoinOptions(): void {
		const presence = account.presence
		if (store.launching || !presence.placeId || presence.placeId <= 0) return
		store.launchInput = {
			...store.launchInput,
			method: LaunchMethod.MethodPlace,
			placeId: String(presence.placeId),
			jobId: presence.gameId ?? "",
			link: "",
			user: "",
			resolveCurrentGame: false,
		}
		workspace.launchOpen = true
		workspace.closeAccountMenus()
	}
</script>

<div class="tag-submenu-host">
	<button class="tag-submenu-trigger" type="button">
		<Tags size={14} aria-hidden="true" />
		<span>Edit tags</span>
		<ChevronRight size={13} class="submenu-chevron" aria-hidden="true" />
	</button>
	<div
		class:open-left={submenuLeft}
		class="tag-submenu account-menu"
		data-account-menu>
		<TagPicker
			{store}
			accounts={[account]}
			onManageTags={() => onManageTags(account.id)} />
	</div>
</div>

{#if browserOpen}
	<button
		class="danger-action"
		type="button"
		onclick={() => void browserStore.close(sessionId)}
		><X size={14} aria-hidden="true" /><span>Close browser</span></button>
	<button type="button" onclick={() => void browserStore.focus(sessionId)}
		><Focus size={14} aria-hidden="true" /><span>Focus browser</span></button>
{:else}
	<button type="button" onclick={() => onBrowser(account.id)}
		><Globe2 size={14} aria-hidden="true" /><span>Open in browser</span></button>
{/if}

<div role="group" aria-label="Copy account details">
	<div class="account-menu-heading" aria-hidden="true">Copy</div>
	{#each copyFields as item (item.field)}
		<button
			type="button"
			aria-label={`Copy ${item.label}`}
			disabled={store.copying}
			onclick={() => void copyField(item.field)}>
			<item.icon size={14} aria-hidden="true" /><span>{item.label}</span>
		</button>
	{/each}
</div>

{#if account.presence.userPresenceType === PresenceType.PresenceTypeInGame}
	<div role="group" aria-label="Game">
		<div class="account-menu-heading" aria-hidden="true">Game</div>
		<button
			class="game-action"
			type="button"
			disabled={store.launching ||
				!account.presence.placeId ||
				account.presence.placeId <= 0}
			data-tooltip={!account.presence.placeId || account.presence.placeId <= 0
				? "Roblox has not shared this account's place."
				: undefined}
			onclick={fillJoinOptions}>
			<Gamepad2 size={14} aria-hidden="true" /><span>Fill join options</span>
		</button>
	</div>
{/if}

<div role="group" aria-label="Session">
	<div class="account-menu-heading" aria-hidden="true">Session</div>
	<button
		class="warning-action"
		type="button"
		disabled={store.copying}
		onclick={() => void copyField(AccountCopyField.CopyCookie)}>
		<KeyRound size={14} aria-hidden="true" /><span>Copy cookie</span>
	</button>
	<button
		class="warning-action"
		type="button"
		disabled={store.busy}
		onclick={() => onRenewCookie(account.id)}>
		<RefreshCw size={14} aria-hidden="true" /><span>Renew cookie</span>
	</button>
</div>
<div class="account-menu-separator"></div>
<button
	class="danger-action"
	type="button"
	disabled={store.busy}
	onclick={() => onRemove(account.id)}>
	<Trash2 size={14} aria-hidden="true" />
	<span>Remove account</span>
</button>

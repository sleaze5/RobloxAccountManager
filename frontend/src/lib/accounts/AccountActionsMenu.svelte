<script lang="ts">
	import AtIcon from "phosphor-svelte/lib/AtIcon"
	import CaretRightIcon from "phosphor-svelte/lib/CaretRightIcon"
	import CornersOutIcon from "phosphor-svelte/lib/CornersOutIcon"
	import GameControllerIcon from "phosphor-svelte/lib/GameControllerIcon"
	import GlobeHemisphereWestIcon from "phosphor-svelte/lib/GlobeHemisphereWestIcon"
	import HashIcon from "phosphor-svelte/lib/HashIcon"
	import KeyIcon from "phosphor-svelte/lib/KeyIcon"
	import ArrowsClockwiseIcon from "phosphor-svelte/lib/ArrowsClockwiseIcon"
	import TagIcon from "phosphor-svelte/lib/TagIcon"
	import TrashIcon from "phosphor-svelte/lib/TrashIcon"
	import UserCircleIcon from "phosphor-svelte/lib/UserCircleIcon"
	import XIcon from "phosphor-svelte/lib/XIcon"
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
			icon: UserCircleIcon,
		},
		{ field: AccountCopyField.CopyUsername, label: "Username", icon: AtIcon },
		{ field: AccountCopyField.CopyUserID, label: "User ID", icon: HashIcon },
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
		<TagIcon size={16} aria-hidden="true" />
		<span>Edit tags</span>
		<CaretRightIcon size={15} class="submenu-chevron" aria-hidden="true" />
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
		><XIcon size={16} aria-hidden="true" /><span>Close browser</span></button>
	<button type="button" onclick={() => void browserStore.focus(sessionId)}
		><CornersOutIcon size={16} aria-hidden="true" /><span>Focus browser</span
		></button>
{:else}
	<button type="button" onclick={() => onBrowser(account.id)}
		><GlobeHemisphereWestIcon size={16} aria-hidden="true" /><span
			>Open in browser</span
		></button>
{/if}

<div role="group" aria-label="Copy account details">
	<div class="account-menu-heading" aria-hidden="true">Copy</div>
	{#each copyFields as item (item.field)}
		<button
			type="button"
			aria-label={`Copy ${item.label}`}
			disabled={store.copying}
			onclick={() => void copyField(item.field)}>
			<item.icon size={16} aria-hidden="true" /><span>{item.label}</span>
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
			<GameControllerIcon size={16} aria-hidden="true" /><span
				>Fill join options</span>
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
		<KeyIcon size={16} aria-hidden="true" /><span>Copy cookie</span>
	</button>
	<button
		class="warning-action"
		type="button"
		disabled={store.busy}
		onclick={() => onRenewCookie(account.id)}>
		<ArrowsClockwiseIcon size={16} aria-hidden="true" /><span>Renew cookie</span>
	</button>
</div>
<div class="account-menu-separator"></div>
<button
	class="danger-action"
	type="button"
	disabled={store.busy}
	onclick={() => onRemove(account.id)}>
	<TrashIcon size={16} aria-hidden="true" />
	<span>Remove account</span>
</button>

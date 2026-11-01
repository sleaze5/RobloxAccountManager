<script lang="ts">
	import arrowsClockwiseIcon from "@phosphor-icons/core/regular/arrows-clockwise.svg?raw"
	import atIcon from "@phosphor-icons/core/regular/at.svg?raw"
	import caretRightIcon from "@phosphor-icons/core/regular/caret-right.svg?raw"
	import cornersOutIcon from "@phosphor-icons/core/regular/corners-out.svg?raw"
	import gameControllerIcon from "@phosphor-icons/core/regular/game-controller.svg?raw"
	import globeHemisphereWestIcon from "@phosphor-icons/core/regular/globe-hemisphere-west.svg?raw"
	import hashIcon from "@phosphor-icons/core/regular/hash.svg?raw"
	import keyIcon from "@phosphor-icons/core/regular/key.svg?raw"
	import tagIcon from "@phosphor-icons/core/regular/tag.svg?raw"
	import trashIcon from "@phosphor-icons/core/regular/trash.svg?raw"
	import userCircleIcon from "@phosphor-icons/core/regular/user-circle.svg?raw"
	import xIcon from "@phosphor-icons/core/regular/x.svg?raw"
	import Icon from "../shared/Icon.svelte"
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
			icon: userCircleIcon,
		},
		{ field: AccountCopyField.CopyUsername, label: "Username", icon: atIcon },
		{ field: AccountCopyField.CopyUserID, label: "User ID", icon: hashIcon },
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

<div class="relative">
	<button class="menu-item" type="button">
		<Icon svg={tagIcon} size={16} aria-hidden="true" />
		<span>Edit tags</span>
		<Icon
			svg={caretRightIcon}
			data-nudge="right"
			size={16}
			class="ml-auto shrink-0"
			aria-hidden="true" />
	</button>
	<div class="submenu" data-side={submenuLeft ? "left" : undefined} data-account-menu>
		<TagPicker
			{store}
			accounts={[account]}
			onManageTags={() => onManageTags(account.id)} />
	</div>
</div>

{#if browserOpen}
	<button
		class="menu-item-danger"
		type="button"
		onclick={() => void browserStore.close(sessionId)}
		><Icon svg={xIcon} size={16} aria-hidden="true" /><span>Close browser</span
		></button>
	<button
		class="menu-item"
		type="button"
		onclick={() => void browserStore.focus(sessionId)}
		><Icon svg={cornersOutIcon} size={16} aria-hidden="true" /><span
			>Focus browser</span
		></button>
{:else}
	<button class="menu-item" type="button" onclick={() => onBrowser(account.id)}
		><Icon svg={globeHemisphereWestIcon} size={16} aria-hidden="true" /><span
			>Open in browser</span
		></button>
{/if}

<div role="group" aria-label="Copy account details">
	<div class="menu-heading" aria-hidden="true">Copy</div>
	{#each copyFields as item (item.field)}
		<button
			class="menu-item"
			type="button"
			aria-label={`Copy ${item.label}`}
			disabled={store.copying}
			onclick={() => void copyField(item.field)}>
			<Icon svg={item.icon} size={16} aria-hidden="true" /><span
				>{item.label}</span>
		</button>
	{/each}
</div>

{#if account.presence.userPresenceType === PresenceType.PresenceTypeInGame}
	<div role="group" aria-label="Game">
		<div class="menu-heading" aria-hidden="true">Game</div>
		<button
			class="menu-item"
			type="button"
			disabled={store.launching ||
				!account.presence.placeId ||
				account.presence.placeId <= 0}
			data-tooltip={!account.presence.placeId || account.presence.placeId <= 0
				? "Roblox has not shared this account's place"
				: undefined}
			onclick={fillJoinOptions}>
			<Icon svg={gameControllerIcon} size={16} aria-hidden="true" /><span
				>Fill join options</span>
		</button>
	</div>
{/if}

<div role="group" aria-label="Session">
	<div class="menu-heading" aria-hidden="true">Session</div>
	<button
		class="menu-item-warning"
		type="button"
		disabled={store.copying}
		onclick={() => void copyField(AccountCopyField.CopyCookie)}>
		<Icon svg={keyIcon} size={16} aria-hidden="true" /><span>Copy cookie</span>
	</button>
	<button
		class="menu-item-warning"
		type="button"
		disabled={store.busy}
		onclick={() => onRenewCookie(account.id)}>
		<Icon svg={arrowsClockwiseIcon} data-turn size={16} aria-hidden="true" /><span
			>Renew cookie</span>
	</button>
</div>
<div class="menu-separator"></div>
<button
	class="menu-item-danger"
	type="button"
	disabled={store.busy}
	onclick={() => onRemove(account.id)}>
	<Icon svg={trashIcon} size={16} aria-hidden="true" />
	<span>Remove account</span>
</button>

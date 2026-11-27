<script lang="ts">
	import arrowCounterClockwiseIcon from "@phosphor-icons/core/regular/arrow-counter-clockwise.svg?raw"
	import arrowsClockwiseIcon from "@phosphor-icons/core/regular/arrows-clockwise.svg?raw"
	import atIcon from "@phosphor-icons/core/regular/at.svg?raw"
	import cornersOutIcon from "@phosphor-icons/core/regular/corners-out.svg?raw"
	import globeHemisphereWestIcon from "@phosphor-icons/core/regular/globe-hemisphere-west.svg?raw"
	import hashIcon from "@phosphor-icons/core/regular/hash.svg?raw"
	import keyIcon from "@phosphor-icons/core/regular/key.svg?raw"
	import linkIcon from "@phosphor-icons/core/regular/link.svg?raw"
	import listPlusIcon from "@phosphor-icons/core/regular/list-plus.svg?raw"
	import tagIcon from "@phosphor-icons/core/regular/tag.svg?raw"
	import textTIcon from "@phosphor-icons/core/regular/text-t.svg?raw"
	import trashIcon from "@phosphor-icons/core/regular/trash.svg?raw"
	import xIcon from "@phosphor-icons/core/regular/x.svg?raw"
	import MenuItem from "../shared/MenuItem.svelte"
	import MenuSection from "../shared/MenuSection.svelte"
	import { browserStore } from "../browser/browser-store.svelte"
	import type { AccountStore } from "./account-store.svelte"
	import type { Account } from "./account-model"
	import TagPicker from "./TagPicker.svelte"
	import {
		AccountCopyField,
		LaunchMethod,
		PresenceType,
		SessionState,
	} from "../backend/bridge"
	import { canReactivate, hasReactivation } from "./moderation.svelte"
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
		browserOpen = $derived(!!sessionId),
		reactivatable = $derived(canReactivate(account.moderation, Date.now())),
		placeId = $derived(account.presence.placeId ?? 0),
		inGame = $derived(
			account.presence.userPresenceType === PresenceType.PresenceTypeInGame,
		)
	const copyFields = $derived([
		{
			field: AccountCopyField.CopyDisplayName,
			label: "Display name",
			icon: textTIcon,
			detail: "",
		},
		{
			field: AccountCopyField.CopyUsername,
			label: "Username",
			icon: atIcon,
			detail: `@${account.username}`,
		},
		{
			field: AccountCopyField.CopyUserID,
			label: "User ID",
			icon: hashIcon,
			detail: account.userId,
		},
	])

	async function copyField(field: AccountCopyField): Promise<void> {
		if (await store.copyAccountFields([account.id], field)) {
			workspace.closeAccountMenus()
		}
	}

	async function copyProfileLink(): Promise<void> {
		if (await store.copyAccountProfileLinks([account.id])) {
			workspace.closeAccountMenus()
		}
	}

	function fillJoinOptions(): void {
		const presence = account.presence
		if (store.launching || placeId <= 0) return
		store.launchInput = {
			...store.launchInput,
			method: LaunchMethod.MethodPlace,
			placeId: String(placeId),
			jobId: presence.gameId ?? "",
			link: "",
			user: "",
			resolveCurrentGame: false,
		}
		workspace.launchOpen = true
		workspace.closeAccountMenus()
	}
</script>

{#if inGame}
	<MenuItem
		label="Fill launch options"
		icon={listPlusIcon}
		disabled={store.launching || placeId <= 0}
		tooltip={placeId <= 0
			? "Roblox has not shared this account's place"
			: undefined}
		onSelect={fillJoinOptions} />
{/if}
<div class="relative">
	<MenuItem label="Edit tags" icon={tagIcon} hasSubmenu />
	<div class="submenu" data-side={submenuLeft ? "left" : undefined} data-account-menu>
		<TagPicker
			{store}
			accounts={[account]}
			onManageTags={() => onManageTags(account.id)} />
	</div>
</div>
{#if browserOpen}
	<MenuItem
		label="Focus browser"
		icon={cornersOutIcon}
		onSelect={() => void browserStore.focus(sessionId)} />
	<MenuItem
		label="Close browser"
		icon={xIcon}
		onSelect={() => void browserStore.close(sessionId)} />
{:else}
	<MenuItem
		label="Open in browser"
		icon={globeHemisphereWestIcon}
		onSelect={() => onBrowser(account.id)} />
{/if}

<MenuSection heading="Copy" accessibleLabel="Copy account details">
	{#each copyFields as fieldOption (fieldOption.field)}
		<MenuItem
			label={fieldOption.label}
			icon={fieldOption.icon}
			detail={fieldOption.detail}
			accessibleLabel={`Copy ${fieldOption.label.toLowerCase()}`}
			disabled={store.copying}
			onSelect={() => void copyField(fieldOption.field)} />
	{/each}
	<MenuItem
		label="Profile link"
		icon={linkIcon}
		accessibleLabel="Copy profile link"
		disabled={store.copying}
		onSelect={() => void copyProfileLink()} />
</MenuSection>

<MenuSection heading="Session">
	{#if account.state === SessionState.StateModerated && hasReactivation(account.moderation)}
		<MenuItem
			label="Reactivate account"
			icon={arrowCounterClockwiseIcon}
			disabled={!reactivatable || store.reactivatingAccountId !== null}
			tooltip={reactivatable ? undefined : "Available when the ban ends"}
			onSelect={() => {
				workspace.closeAccountMenus()
				void store.reactivateAccount(account.id)
			}} />
	{/if}
	<MenuItem
		label="Copy cookie"
		icon={keyIcon}
		tone="warning"
		disabled={store.copying}
		onSelect={() => void copyField(AccountCopyField.CopyCookie)} />
	<MenuItem
		label="Renew cookie"
		icon={arrowsClockwiseIcon}
		tone="warning"
		disabled={store.busy}
		onSelect={() => onRenewCookie(account.id)} />
</MenuSection>
<div class="menu-separator"></div>
<MenuItem
	label="Remove account"
	icon={trashIcon}
	tone="danger"
	disabled={store.busy}
	onSelect={() => onRemove(account.id)} />

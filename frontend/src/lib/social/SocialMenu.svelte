<script lang="ts">
	import arrowSquareOutIcon from "@phosphor-icons/core/regular/arrow-square-out.svg?raw"
	import atIcon from "@phosphor-icons/core/regular/at.svg?raw"
	import hashIcon from "@phosphor-icons/core/regular/hash.svg?raw"
	import listPlusIcon from "@phosphor-icons/core/regular/list-plus.svg?raw"
	import magnifyingGlassIcon from "@phosphor-icons/core/regular/magnifying-glass.svg?raw"
	import prohibitIcon from "@phosphor-icons/core/regular/prohibit.svg?raw"
	import shieldCheckIcon from "@phosphor-icons/core/regular/shield-check.svg?raw"
	import shieldSlashIcon from "@phosphor-icons/core/regular/shield-slash.svg?raw"
	import textTIcon from "@phosphor-icons/core/regular/text-t.svg?raw"
	import userCircleMinusIcon from "@phosphor-icons/core/regular/user-circle-minus.svg?raw"
	import userCirclePlusIcon from "@phosphor-icons/core/regular/user-circle-plus.svg?raw"
	import userListIcon from "@phosphor-icons/core/regular/user-list.svg?raw"
	import userMinusIcon from "@phosphor-icons/core/regular/user-minus.svg?raw"
	import userPlusIcon from "@phosphor-icons/core/regular/user-plus.svg?raw"
	import Icon from "../shared/Icon.svelte"
	import {
		FriendStatus,
		PresenceType,
		SocialList,
		TrustedStatus,
	} from "../backend/bridge"
	import type { SocialUser } from "../backend/bridge"
	import { floatMenuAtPoint, menuIn, menuKeyboard, menuOut } from "../shared/presence"
	import { personName } from "./social-store.svelte"
	import type { SocialMenu, SocialStore } from "./social-store.svelte"

	let {
		store,
		menu,
		onFillLaunch,
		onSearchPlace,
		onUnfriend,
		onRemoveTrusted,
	}: {
		store: SocialStore
		menu: SocialMenu
		onFillLaunch: (placeId: number, jobId: string) => void
		onSearchPlace: (placeId: number) => void
		onUnfriend: (person: SocialUser) => void
		onRemoveTrusted: (person: SocialUser) => void
	} = $props()
	const person = $derived(menu.person),
		presence = $derived(person.presence),
		placeId = $derived(presence?.placeId ?? 0),
		inExperience = $derived(
			presence?.userPresenceType === PresenceType.PresenceTypeInGame ||
				presence?.userPresenceType === PresenceType.PresenceTypeInStudio,
		),
		joinable = $derived(
			presence?.userPresenceType === PresenceType.PresenceTypeInGame &&
				placeId > 0 &&
				!!presence.gameId,
		),
		relationship = $derived(store.menuRelationship?.relationship ?? null),
		unavailable = $derived(store.menuRelationship?.unavailable ?? []),
		isFriend = $derived(relationship?.friend === FriendStatus.FriendStatusFriends),
		isTrusted = $derived(
			isFriend && relationship?.trusted === TrustedStatus.TrustedStatusTrusted,
		),
		busy = $derived(relationship === null || store.updating !== null)

	function closeOnTab(event: KeyboardEvent): void {
		if (event.key === "Tab") store.closeMenu()
	}

	function handleWindowClick(event: MouseEvent): void {
		if (
			event.target instanceof Element &&
			event.target.closest("[data-social-menu]")
		) {
			return
		}
		store.closeMenu()
	}
</script>

<svelte:window
	onclick={handleWindowClick}
	onkeydown={(event) => {
		if (event.key === "Escape") store.closeMenu()
	}} />

<div
	class="fixed menu"
	role="menu"
	aria-label={`${personName(person)} actions`}
	tabindex="-1"
	data-social-menu
	onkeydown={closeOnTab}
	use:floatMenuAtPoint={menu}
	use:menuKeyboard
	in:menuIn
	out:menuOut>
	{#if person.request}
		<button
			class="menu-item"
			type="button"
			role="menuitem"
			disabled={store.responding !== null}
			onclick={() => void store.respond(person, true)}
			><Icon svg={userPlusIcon} size={16} aria-hidden="true" /><span
				>Accept request</span
			></button>
		<button
			class="menu-item"
			type="button"
			role="menuitem"
			disabled={store.responding !== null}
			onclick={() => void store.respond(person, false)}
			><Icon svg={prohibitIcon} size={16} aria-hidden="true" /><span
				>Decline request</span
			></button>
	{/if}
	{#if joinable}
		<button
			class="menu-item"
			type="button"
			role="menuitem"
			onclick={() => {
				const jobId = presence?.gameId ?? ""
				onFillLaunch(placeId, jobId)
				store.closeMenu()
			}}
			><Icon svg={listPlusIcon} size={16} aria-hidden="true" /><span
				>Fill launch options</span
			></button>
	{/if}
	{#if inExperience && placeId > 0}
		<button
			class="menu-item"
			type="button"
			role="menuitem"
			onclick={() => {
				onSearchPlace(placeId)
				store.closeMenu()
			}}
			><Icon svg={magnifyingGlassIcon} size={16} aria-hidden="true" /><span
				>Find place in Games</span
			></button>
	{/if}
	<button
		class="menu-item"
		type="button"
		role="menuitem"
		onclick={() => {
			store.explore(person, SocialList.SocialFriends)
			store.closeMenu()
		}}
		><Icon svg={userListIcon} size={16} aria-hidden="true" /><span
			>View friends</span
		></button>
	<button
		class="menu-item"
		type="button"
		role="menuitem"
		onclick={() => void store.openProfile(person)}
		><Icon svg={arrowSquareOutIcon} size={16} aria-hidden="true" /><span
			>Open Roblox profile</span
		></button>

	{#if !relationship?.self}
		<div role="group" aria-label="Connection" aria-busy={relationship === null}>
			<div class="menu-heading" aria-hidden="true">Connection</div>
			<button
				class="menu-item"
				type="button"
				role="menuitem"
				disabled={busy || unavailable.includes("follow")}
				onclick={() => {
					void store.setFollowing(person, !relationship?.following)
					store.closeMenu()
				}}
				><Icon
					svg={relationship?.following
						? userCircleMinusIcon
						: userCirclePlusIcon}
					size={16}
					aria-hidden="true" /><span
					>{relationship?.following ? "Unfollow" : "Follow"}</span
				></button>
			{#if !person.request && !isFriend}
				{#if relationship?.friend === FriendStatus.FriendStatusRequestReceived}
					<button
						class="menu-item"
						type="button"
						role="menuitem"
						disabled={store.responding !== null}
						onclick={() => void store.respond(person, true)}
						><Icon svg={userPlusIcon} size={16} aria-hidden="true" /><span
							>Accept friend request</span
						></button>
				{:else}
					<button
						class="menu-item"
						type="button"
						role="menuitem"
						disabled={busy ||
							unavailable.includes("friendship") ||
							relationship?.friend ===
								FriendStatus.FriendStatusRequestSent}
						onclick={() => void store.sendFriendRequest(person)}
						><Icon svg={userPlusIcon} size={16} aria-hidden="true" /><span
							>{relationship?.friend ===
							FriendStatus.FriendStatusRequestSent
								? "Friend request sent"
								: "Add friend"}</span
						></button>
				{/if}
			{/if}
			{#if relationship === null || (isFriend && !isTrusted)}
				{#if relationship?.trusted === TrustedStatus.TrustedStatusRequestReceived}
					<button
						class="menu-item"
						type="button"
						role="menuitem"
						disabled={busy}
						onclick={() => void store.acceptTrustedFriendRequest(person)}
						><Icon
							svg={shieldCheckIcon}
							size={16}
							aria-hidden="true" /><span>Accept trusted request</span
						></button>
				{:else}
					<button
						class="menu-item"
						type="button"
						role="menuitem"
						disabled={busy ||
							unavailable.includes("trusted") ||
							relationship?.trusted ===
								TrustedStatus.TrustedStatusRequestSent}
						onclick={() => void store.sendTrustedFriendRequest(person)}
						><Icon
							svg={shieldCheckIcon}
							size={16}
							aria-hidden="true" /><span
							>{relationship?.trusted ===
							TrustedStatus.TrustedStatusRequestSent
								? "Trusted request sent"
								: "Add trusted friend"}</span
						></button>
				{/if}
			{/if}
			{#if store.menuRelationshipFailed}
				<p class="px-2 py-1 text-2xs text-fg-muted" role="status">
					Could not check this user. Reopen the menu to retry.
				</p>
			{/if}
		</div>
	{/if}

	<div role="group" aria-label="Copy user details">
		<div class="menu-heading" aria-hidden="true">Copy</div>
		<button
			class="menu-item"
			type="button"
			role="menuitem"
			aria-label="Copy display name"
			onclick={() => void store.copy(personName(person), "Display name")}
			><Icon svg={textTIcon} size={16} aria-hidden="true" /><span
				>Display name</span
			></button>
		<button
			class="menu-item"
			type="button"
			role="menuitem"
			aria-label="Copy username"
			disabled={!person.username}
			onclick={() => void store.copy(person.username, "Username")}
			><Icon svg={atIcon} size={16} aria-hidden="true" /><span>Username</span
			></button>
		<button
			class="menu-item"
			type="button"
			role="menuitem"
			aria-label="Copy user ID"
			onclick={() => void store.copy(person.userId, "User ID")}
			><Icon svg={hashIcon} size={16} aria-hidden="true" /><span>User ID</span
			></button>
	</div>
	{#if isFriend}
		<div class="menu-separator"></div>
		{#if isTrusted}
			<button
				class="menu-item-danger"
				type="button"
				role="menuitem"
				onclick={() => {
					onRemoveTrusted(person)
					store.closeMenu()
				}}
				><Icon svg={shieldSlashIcon} size={16} aria-hidden="true" /><span
					>Remove trusted friend</span
				></button>
		{/if}
		<button
			class="menu-item-danger"
			type="button"
			role="menuitem"
			onclick={() => {
				onUnfriend(person)
				store.closeMenu()
			}}
			><Icon svg={userMinusIcon} size={16} aria-hidden="true" /><span
				>Unfriend</span
			></button>
	{/if}
</div>

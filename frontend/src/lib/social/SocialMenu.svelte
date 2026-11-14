<script lang="ts">
	import arrowSquareOutIcon from "@phosphor-icons/core/regular/arrow-square-out.svg?raw"
	import atIcon from "@phosphor-icons/core/regular/at.svg?raw"
	import hashIcon from "@phosphor-icons/core/regular/hash.svg?raw"
	import listPlusIcon from "@phosphor-icons/core/regular/list-plus.svg?raw"
	import magnifyingGlassIcon from "@phosphor-icons/core/regular/magnifying-glass.svg?raw"
	import textTIcon from "@phosphor-icons/core/regular/text-t.svg?raw"
	import userListIcon from "@phosphor-icons/core/regular/user-list.svg?raw"
	import Icon from "../shared/Icon.svelte"
	import { PresenceType, SocialList } from "../backend/bridge"
	import type { SocialUser } from "../backend/bridge"
	import { floatMenuAtPoint, menuIn, menuKeyboard, menuOut } from "../shared/presence"
	import {
		relationshipActions,
		relationshipMenuItemClass,
		requestRelationship,
	} from "./relationship"
	import type { RelationshipActionId } from "./relationship"
	import { personName } from "./social-store.svelte"
	import type { SocialMenu, SocialStore } from "./social-store.svelte"

	let {
		store,
		menu,
		onFillLaunch,
		onSearchPlace,
		onRelationshipAction,
	}: {
		store: SocialStore
		menu: SocialMenu
		onFillLaunch: (placeId: number, jobId: string) => void
		onSearchPlace: (placeId: number) => void
		onRelationshipAction: (person: SocialUser, action: RelationshipActionId) => void
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
		relationship = $derived(
			store.menuRelationship?.relationship ?? requestRelationship(person),
		),
		actions = $derived(
			relationship
				? relationshipActions(
						relationship,
						store.menuRelationship?.unavailable ?? [],
					)
				: [],
		)

	// Closing the menu clears the menu prop that person derives from, so the
	// action must read person first.
	function run(action: RelationshipActionId): void {
		onRelationshipAction(person, action)
		store.closeMenu()
	}

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
	{#if !relationship?.self}
		<div role="group" aria-label="Connection" aria-busy={relationship === null}>
			<div class="menu-heading" aria-hidden="true">Connection</div>
			{#each actions as action (action.id)}
				<button
					class={relationshipMenuItemClass(action.tone)}
					type="button"
					role="menuitem"
					disabled={store.pendingAction !== null}
					onclick={() => run(action.id)}
					><Icon svg={action.icon} size={16} aria-hidden="true" /><span
						>{action.label}</span
					></button>
			{/each}
			{#if relationship === null && !store.menuRelationshipFailed}
				<p class="px-2 py-1 text-2xs text-fg-muted" role="status">
					Checking your connection…
				</p>
			{:else if store.menuRelationshipFailed}
				<p class="px-2 py-1 text-2xs text-fg-muted" role="status">
					Could not check this user. Reopen the menu to retry.
				</p>
			{/if}
		</div>
	{/if}
</div>

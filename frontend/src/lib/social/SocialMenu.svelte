<script lang="ts">
	import { getProfilePreviews } from "../shared/profile-preview"
	import atIcon from "@phosphor-icons/core/regular/at.svg?raw"
	import globeHemisphereWestIcon from "@phosphor-icons/core/regular/globe-hemisphere-west.svg?raw"
	import hashIcon from "@phosphor-icons/core/regular/hash.svg?raw"
	import linkIcon from "@phosphor-icons/core/regular/link.svg?raw"
	import listPlusIcon from "@phosphor-icons/core/regular/list-plus.svg?raw"
	import magnifyingGlassIcon from "@phosphor-icons/core/regular/magnifying-glass.svg?raw"
	import textTIcon from "@phosphor-icons/core/regular/text-t.svg?raw"
	import userListIcon from "@phosphor-icons/core/regular/user-list.svg?raw"
	import { unsavedChangesGuard } from "../layout/unsaved-changes-guard"
	import { PresenceType, SocialList } from "../backend/bridge"
	import type { SocialUser } from "../backend/bridge"
	import MenuItem from "../shared/MenuItem.svelte"
	import MenuSection from "../shared/MenuSection.svelte"
	import PointMenu from "../shared/PointMenu.svelte"
	import {
		getRobloxPageOpener,
		openInBrowserUnavailableReason,
		robloxUserProfileUrl,
	} from "../shared/roblox-pages"
	import {
		relationshipActions,
		relationshipMenuItemTone,
		requestRelationship,
	} from "./relationship"
	import type { RelationshipActionId } from "./relationship"
	import { personName } from "./social-store.svelte"
	import type { SocialMenu, SocialStore } from "./social-store.svelte"

	let {
		store,
		menu,
		onFillLaunch,
		onRelationshipAction,
	}: {
		store: SocialStore
		menu: SocialMenu
		onFillLaunch: (placeId: number, serverId: string) => void
		onRelationshipAction: (person: SocialUser, action: RelationshipActionId) => void
	} = $props()
	const previews = getProfilePreviews(),
		robloxPages = getRobloxPageOpener()
	const person = $derived(menu.person),
		profileUrl = $derived(robloxUserProfileUrl(person.userId)),
		presence = $derived(person.presence),
		placeId = $derived(presence?.placeId ?? 0),
		isInGame = $derived(
			presence?.userPresenceType === PresenceType.PresenceTypeInGame ||
				presence?.userPresenceType === PresenceType.PresenceTypeInStudio,
		),
		joinable = $derived(
			presence?.userPresenceType === PresenceType.PresenceTypeInGame &&
				placeId > 0 &&
				!!presence.serverId,
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

	// Closing the menu clears the menu prop that person derives from, so each
	// action reads what it needs first.
	function run(action: RelationshipActionId): void {
		onRelationshipAction(person, action)
		store.closeMenu()
	}
</script>

<PointMenu
	label={`${personName(person)} actions`}
	point={menu}
	owner="data-social-menu"
	onClose={() => store.closeMenu()}>
	{#if joinable}
		<MenuItem
			label="Fill launch options"
			icon={listPlusIcon}
			onSelect={() => {
				const serverId = presence?.serverId ?? ""
				onFillLaunch(placeId, serverId)
				store.closeMenu()
			}} />
	{/if}
	{#if isInGame && placeId > 0}
		<MenuItem
			label="Show game"
			icon={magnifyingGlassIcon}
			onSelect={() => {
				previews.show({ kind: "Place", placeId })
				store.closeMenu()
			}} />
	{/if}
	<MenuItem
		label="View friends"
		icon={userListIcon}
		onSelect={() => {
			const exploredPerson = person
			store.closeMenu()
			unsavedChangesGuard.leave(() =>
				store.explore(exploredPerson, SocialList.SocialFriends),
			)
		}} />
	<MenuItem
		label="Open in browser"
		icon={globeHemisphereWestIcon}
		disabled={!robloxPages.canOpen}
		tooltip={robloxPages.canOpen ? undefined : openInBrowserUnavailableReason}
		onSelect={() => {
			const pageUrl = profileUrl
			store.closeMenu()
			robloxPages.open(pageUrl)
		}} />

	<MenuSection heading="Copy" accessibleLabel="Copy user details">
		<MenuItem
			label="Display name"
			icon={textTIcon}
			accessibleLabel="Copy display name"
			onSelect={() => void store.copy(personName(person), "Display name")} />
		<MenuItem
			label="Username"
			icon={atIcon}
			detail={person.username ? `@${person.username}` : ""}
			accessibleLabel="Copy username"
			disabled={!person.username}
			onSelect={() => void store.copy(person.username, "Username")} />
		<MenuItem
			label="User ID"
			icon={hashIcon}
			detail={String(person.userId)}
			accessibleLabel="Copy user ID"
			onSelect={() => void store.copy(person.userId, "User ID")} />
		<MenuItem
			label="Profile link"
			icon={linkIcon}
			accessibleLabel="Copy profile link"
			onSelect={() => void store.copy(profileUrl, "Profile link")} />
	</MenuSection>
	{#if !relationship?.self}
		<MenuSection heading="Connection" isBusy={relationship === null}>
			{#each actions as action (action.id)}
				<MenuItem
					label={action.label}
					icon={action.icon}
					tone={relationshipMenuItemTone(action.tone)}
					disabled={store.pendingAction !== null}
					onSelect={() => run(action.id)} />
			{/each}
			{#if relationship === null && !store.menuRelationshipFailed}
				<p class="px-2 py-1 text-2xs text-fg-muted" role="status">
					Checking your connection…
				</p>
			{:else if store.menuRelationshipFailed}
				<p class="px-2 py-1 text-2xs text-fg-muted" role="status">
					Connection unavailable
				</p>
			{/if}
		</MenuSection>
	{/if}
</PointMenu>

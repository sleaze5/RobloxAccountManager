<script lang="ts">
	import type { GroupSummary } from "../backend/bridge"
	import globeHemisphereWestIcon from "@phosphor-icons/core/regular/globe-hemisphere-west.svg?raw"
	import hashIcon from "@phosphor-icons/core/regular/hash.svg?raw"
	import linkIcon from "@phosphor-icons/core/regular/link.svg?raw"
	import signOutIcon from "@phosphor-icons/core/regular/sign-out.svg?raw"
	import starIcon from "@phosphor-icons/core/regular/star.svg?raw"
	import starFillIcon from "@phosphor-icons/core/fill/star-fill.svg?raw"
	import textTIcon from "@phosphor-icons/core/regular/text-t.svg?raw"
	import MenuItem from "../shared/MenuItem.svelte"
	import MenuSection from "../shared/MenuSection.svelte"
	import PointMenu from "../shared/PointMenu.svelte"
	import {
		getRobloxPageOpener,
		openInBrowserUnavailableReason,
		robloxGroupUrl,
	} from "../shared/roblox-pages"
	import type { GroupMenu, GroupsStore } from "./groups-store.svelte"

	let {
		store,
		menu,
		onLeave,
		onChangePrimary,
	}: {
		store: GroupsStore
		menu: GroupMenu
		onLeave: (groupIds: number[]) => void
		// onChangePrimary asks to make the group primary, or to remove it as
		// the primary group when isPrimary is true.
		onChangePrimary: (group: GroupSummary, isPrimary: boolean) => void
	} = $props()
	const robloxPages = getRobloxPageOpener()
	const group = $derived(menu.group),
		membership = $derived(store.membershipOf(group.groupId)),
		isPrimary = $derived(store.myGroups?.primaryGroupId === group.groupId)

	// Closing the menu clears the menu prop that group derives from, so each
	// action reads the group first.
	function changePrimaryGroup(): void {
		const changedGroup = group,
			wasPrimary = isPrimary
		store.closeMenu()
		onChangePrimary(changedGroup, wasPrimary)
	}
</script>

<PointMenu
	label={`${group.name} actions`}
	point={menu}
	owner="data-group-menu"
	onClose={() => store.closeMenu()}>
	<!-- The group profile shows the primary group and leave buttons. -->
	{#if membership && !menu.isFromProfile}
		<MenuItem
			label={isPrimary ? "Remove primary group" : "Make primary group"}
			icon={isPrimary ? starFillIcon : starIcon}
			disabled={store.pendingAction !== null}
			onSelect={changePrimaryGroup} />
	{/if}
	<MenuItem
		label="Open in browser"
		icon={globeHemisphereWestIcon}
		disabled={!robloxPages.canOpen}
		tooltip={robloxPages.canOpen ? undefined : openInBrowserUnavailableReason}
		onSelect={() => {
			const pageUrl = robloxGroupUrl(group.groupId)
			store.closeMenu()
			robloxPages.open(pageUrl)
		}} />

	<MenuSection heading="Copy" accessibleLabel="Copy group details">
		<MenuItem
			label="Name"
			icon={textTIcon}
			accessibleLabel="Copy group name"
			onSelect={() => void store.copyField(group, "name")} />
		<MenuItem
			label="Group ID"
			icon={hashIcon}
			detail={String(group.groupId)}
			accessibleLabel="Copy group ID"
			onSelect={() => void store.copyField(group, "groupId")} />
		<MenuItem
			label="Group link"
			icon={linkIcon}
			accessibleLabel="Copy group link"
			onSelect={() => void store.copyField(group, "link")} />
	</MenuSection>
	{#if store.canLeave(group.groupId) && !menu.isFromProfile}
		<div class="menu-separator"></div>
		<MenuItem
			label="Leave group"
			icon={signOutIcon}
			tone="danger"
			disabled={store.isLeavingGroups || store.pendingAction !== null}
			onSelect={() => {
				const { groupId } = group
				store.closeMenu()
				onLeave([groupId])
			}} />
	{/if}
</PointMenu>

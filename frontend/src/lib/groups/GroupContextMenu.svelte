<script lang="ts">
	import globeIcon from "@phosphor-icons/core/regular/globe.svg?raw"
	import hashIcon from "@phosphor-icons/core/regular/hash.svg?raw"
	import linkIcon from "@phosphor-icons/core/regular/link.svg?raw"
	import signOutIcon from "@phosphor-icons/core/regular/sign-out.svg?raw"
	import textTIcon from "@phosphor-icons/core/regular/text-t.svg?raw"
	import Icon from "../shared/Icon.svelte"
	import { floatMenuAtPoint, menuIn, menuKeyboard, menuOut } from "../shared/presence"
	import type { GroupMenu, GroupsStore } from "./groups-store.svelte"

	let {
		store,
		menu,
		onOpenInBrowser,
		onLeave,
	}: {
		store: GroupsStore
		menu: GroupMenu
		onOpenInBrowser: (pageUrl: string) => void
		onLeave: (groupIds: number[]) => void
	} = $props()

	const groups = $derived(menu.groups),
		isMultiple = $derived(groups.length > 1),
		leavableGroupIds = $derived(
			groups
				.filter((group) => store.canLeave(group.groupId))
				.map((group) => group.groupId),
		)

	function groupLink(groupId: number): string {
		return `https://www.roblox.com/communities/${groupId}`
	}

	function copyEach(
		read: (groupId: number, name: string) => string,
		label: string,
	): void {
		void store.copy(
			groups.map((group) => read(group.groupId, group.name)).join("\n"),
			label,
		)
	}

	function handleWindowClick(event: MouseEvent): void {
		if (
			event.target instanceof Element &&
			event.target.closest("[data-group-menu]")
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
	role="group"
	aria-label={isMultiple
		? `${groups.length} groups actions`
		: `${groups[0].name} actions`}
	tabindex="-1"
	data-group-menu
	use:floatMenuAtPoint={menu}
	use:menuKeyboard
	in:menuIn
	out:menuOut>
	{#if !isMultiple}
		<button
			class="menu-item"
			type="button"
			onclick={() => {
				const pageUrl = groupLink(groups[0].groupId)
				store.closeMenu()
				onOpenInBrowser(pageUrl)
			}}
			><Icon svg={globeIcon} size={16} aria-hidden="true" /><span
				>Open in browser</span
			></button>
	{/if}
	<div role="group" aria-label="Copy group details">
		<div class="menu-heading" aria-hidden="true">Copy</div>
		<button
			class="menu-item"
			type="button"
			aria-label={isMultiple ? "Copy group names" : "Copy group name"}
			onclick={() =>
				copyEach(
					(_groupId, name) => name,
					isMultiple ? "Group names" : "Group name",
				)}
			><Icon svg={textTIcon} size={16} aria-hidden="true" /><span
				>{isMultiple ? "Names" : "Name"}</span
			></button>
		<button
			class="menu-item"
			type="button"
			aria-label={isMultiple ? "Copy group IDs" : "Copy group ID"}
			onclick={() =>
				copyEach(
					(groupId) => String(groupId),
					isMultiple ? "Group IDs" : "Group ID",
				)}
			><Icon svg={hashIcon} size={16} aria-hidden="true" /><span
				>{isMultiple ? "Group IDs" : "Group ID"}</span
			></button>
		<button
			class="menu-item"
			type="button"
			aria-label={isMultiple ? "Copy group links" : "Copy group link"}
			onclick={() =>
				copyEach(
					(groupId) => groupLink(groupId),
					isMultiple ? "Group links" : "Group link",
				)}
			><Icon svg={linkIcon} size={16} aria-hidden="true" /><span
				>{isMultiple ? "Group links" : "Group link"}</span
			></button>
	</div>
	{#if leavableGroupIds.length > 0}
		<div class="menu-separator"></div>
		<button
			class="menu-item-danger"
			type="button"
			disabled={store.isLeavingGroups || store.pendingAction !== null}
			onclick={() => {
				const groupIds = leavableGroupIds
				store.closeMenu()
				onLeave(groupIds)
			}}
			><Icon svg={signOutIcon} size={16} aria-hidden="true" /><span
				>{leavableGroupIds.length > 1
					? `Leave ${leavableGroupIds.length} groups`
					: "Leave group"}</span
			></button>
	{/if}
</div>

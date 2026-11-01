<script lang="ts">
	import arrowSquareOutIcon from "@phosphor-icons/core/regular/arrow-square-out.svg?raw"
	import atIcon from "@phosphor-icons/core/regular/at.svg?raw"
	import hashIcon from "@phosphor-icons/core/regular/hash.svg?raw"
	import listPlusIcon from "@phosphor-icons/core/regular/list-plus.svg?raw"
	import magnifyingGlassIcon from "@phosphor-icons/core/regular/magnifying-glass.svg?raw"
	import prohibitIcon from "@phosphor-icons/core/regular/prohibit.svg?raw"
	import textTIcon from "@phosphor-icons/core/regular/text-t.svg?raw"
	import userListIcon from "@phosphor-icons/core/regular/user-list.svg?raw"
	import userMinusIcon from "@phosphor-icons/core/regular/user-minus.svg?raw"
	import userPlusIcon from "@phosphor-icons/core/regular/user-plus.svg?raw"
	import Icon from "../shared/Icon.svelte"
	import { PresenceType, SocialList } from "../backend/bridge"
	import type { SocialUser } from "../backend/bridge"
	import { menuIn, menuOut, placeMenuAtPoint } from "../shared/presence"
	import { personName } from "./social-store.svelte"
	import type { SocialMenu, SocialStore } from "./social-store.svelte"

	let {
		store,
		menu,
		onFillLaunch,
		onSearchPlace,
		onUnfriend,
	}: {
		store: SocialStore
		menu: SocialMenu
		onFillLaunch: (placeId: number, jobId: string) => void
		onSearchPlace: (placeId: number) => void
		onUnfriend: (person: SocialUser) => void
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
		)

	function positionMenu(node: HTMLElement, position: SocialMenu) {
		const previousFocus = document.activeElement
		let current = position
		const reposition = () => {
			placeMenuAtPoint(node, current)
		}
		reposition()
		node.focus({ preventScroll: true })
		window.addEventListener("resize", reposition)
		return {
			update(next: SocialMenu) {
				current = next
				reposition()
			},
			destroy() {
				window.removeEventListener("resize", reposition)
				if (
					node.contains(document.activeElement) &&
					previousFocus instanceof HTMLElement &&
					previousFocus.isConnected
				) {
					previousFocus.focus({ preventScroll: true })
				}
			},
		}
	}

	function moveFocus(event: KeyboardEvent): void {
		const buttons = Array.from(
				(
					event.currentTarget as HTMLElement
				).querySelectorAll<HTMLButtonElement>("button:enabled"),
			),
			current = buttons.indexOf(document.activeElement as HTMLButtonElement)
		let next: number
		switch (event.key) {
			case "ArrowDown":
				next = (current + 1) % buttons.length
				break
			case "ArrowUp":
				next = (current - 1 + buttons.length) % buttons.length
				break
			case "Home":
				next = 0
				break
			case "End":
				next = buttons.length - 1
				break
			case "Tab":
				store.closeMenu()
				return
			default:
				return
		}
		event.preventDefault()
		buttons[next]?.focus()
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
	onkeydown={moveFocus}
	use:positionMenu={menu}
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
				store.closeMenu()
				onFillLaunch(placeId, jobId)
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
				store.closeMenu()
				onSearchPlace(placeId)
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
			store.closeMenu()
			store.explore(person, SocialList.SocialFriends)
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
	{#if store.ownFriends}
		<div class="menu-separator"></div>
		<button
			class="menu-item-danger"
			type="button"
			role="menuitem"
			onclick={() => {
				store.closeMenu()
				onUnfriend(person)
			}}
			><Icon svg={userMinusIcon} size={16} aria-hidden="true" /><span
				>Unfriend</span
			></button>
	{/if}
</div>

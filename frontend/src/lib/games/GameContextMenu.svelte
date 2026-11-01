<script lang="ts">
	import hashIcon from "@phosphor-icons/core/regular/hash.svg?raw"
	import listPlusIcon from "@phosphor-icons/core/regular/list-plus.svg?raw"
	import pencilLineIcon from "@phosphor-icons/core/regular/pencil-line.svg?raw"
	import starFillIcon from "@phosphor-icons/core/fill/star-fill.svg?raw"
	import starIcon from "@phosphor-icons/core/regular/star.svg?raw"
	import textTIcon from "@phosphor-icons/core/regular/text-t.svg?raw"
	import userCircleIcon from "@phosphor-icons/core/regular/user-circle.svg?raw"
	import usersThreeIcon from "@phosphor-icons/core/regular/users-three.svg?raw"
	import Icon from "../shared/Icon.svelte"
	import { menuIn, menuOut, placeMenuAtPoint } from "../shared/presence"
	import type { GameMenu, GamesStore } from "./games-store.svelte"

	let {
		store,
		menu,
		onFillLaunch,
	}: {
		store: GamesStore
		menu: GameMenu
		onFillLaunch: (placeId: number) => void
	} = $props()
	const place = $derived(menu.place),
		favorite = $derived(store.isFavorite(place.placeId)),
		group = $derived(place.creatorType === "Group"),
		owner = $derived(group ? "Group" : "Creator"),
		favoriteBusy = $derived(
			store.savingFavorite || store.savingNickname || !store.favoritesLoaded,
		)

	function positionMenu(node: HTMLElement, position: GameMenu) {
		const previousFocus = document.activeElement
		let current = position
		const reposition = () => {
			placeMenuAtPoint(node, current)
		}
		reposition()
		node.focus({ preventScroll: true })
		window.addEventListener("resize", reposition)
		return {
			update(next: GameMenu) {
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

	function handleWindowClick(event: MouseEvent): void {
		if (
			event.target instanceof Element &&
			event.target.closest("[data-game-menu]")
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
	aria-label={`${store.displayName(place)} actions`}
	tabindex="-1"
	data-game-menu
	use:positionMenu={menu}
	in:menuIn
	out:menuOut>
	<button
		class="menu-item"
		type="button"
		onclick={() => {
			const { placeId } = place
			store.closeMenu()
			onFillLaunch(placeId)
		}}
		><Icon svg={listPlusIcon} size={16} aria-hidden="true" /><span
			>Fill launch options</span
		></button>
	{#if favorite}
		<button
			class="menu-item"
			type="button"
			onclick={() => store.editNickname(place)}
			><Icon svg={pencilLineIcon} size={16} aria-hidden="true" /><span
				>{store.nicknameFor(place.placeId)
					? "Edit nickname"
					: "Add nickname"}</span
			></button>
	{:else}
		<button
			class="menu-item"
			type="button"
			disabled={favoriteBusy}
			onclick={() => void store.toggleFavorite(place)}
			><Icon svg={starIcon} size={16} aria-hidden="true" /><span
				>Add to favorites</span
			></button>
	{/if}

	<div role="group" aria-label="Copy game details">
		<div class="menu-heading" aria-hidden="true">Copy</div>
		<button
			class="menu-item"
			type="button"
			aria-label="Copy game name"
			onclick={() => void store.copy(place.name, "Game name")}
			><Icon svg={textTIcon} size={16} aria-hidden="true" /><span>Name</span
			></button>
		<button
			class="menu-item"
			type="button"
			aria-label="Copy place ID"
			onclick={() => void store.copy(place.placeId, "Place ID")}
			><Icon svg={hashIcon} size={16} aria-hidden="true" /><span>Place ID</span
			></button>
		<button
			class="menu-item"
			type="button"
			aria-label="Copy universe ID"
			disabled={!place.universeId}
			onclick={() => void store.copy(place.universeId, "Universe ID")}
			><Icon svg={hashIcon} size={16} aria-hidden="true" /><span>Universe ID</span
			></button>
		<button
			class="menu-item"
			type="button"
			aria-label={`Copy ${owner.toLowerCase()} name`}
			disabled={!place.creatorName}
			onclick={() => void store.copy(place.creatorName, `${owner} name`)}>
			{#if group}<Icon
					svg={usersThreeIcon}
					size={16}
					aria-hidden="true" />{:else}<Icon
					svg={userCircleIcon}
					size={16}
					aria-hidden="true" />{/if}
			<span>{owner} name</span>
		</button>
		<button
			class="menu-item"
			type="button"
			aria-label={`Copy ${owner.toLowerCase()} ID`}
			disabled={!place.creatorId}
			onclick={() => void store.copy(place.creatorId, `${owner} ID`)}
			><Icon svg={hashIcon} size={16} aria-hidden="true" /><span>{owner} ID</span
			></button>
	</div>
	{#if favorite}
		<div class="menu-separator"></div>
		<button
			class="menu-item-danger"
			type="button"
			disabled={favoriteBusy}
			onclick={() => void store.toggleFavorite(place)}
			><Icon svg={starFillIcon} size={16} aria-hidden="true" /><span
				>Remove from favorites</span
			></button>
	{/if}
</div>

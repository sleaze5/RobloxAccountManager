<script lang="ts">
	import HashIcon from "phosphor-svelte/lib/HashIcon"
	import ListPlusIcon from "phosphor-svelte/lib/ListPlusIcon"
	import PencilLineIcon from "phosphor-svelte/lib/PencilLineIcon"
	import StarIcon from "phosphor-svelte/lib/StarIcon"
	import TextTIcon from "phosphor-svelte/lib/TextTIcon"
	import UserCircleIcon from "phosphor-svelte/lib/UserCircleIcon"
	import UsersThreeIcon from "phosphor-svelte/lib/UsersThreeIcon"
	import { menuIn, menuOut } from "../shared/presence"
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
		owner = $derived(group ? "Group" : "Creator")

	function positionMenu(node: HTMLElement, position: GameMenu) {
		const previousFocus = document.activeElement
		let current = position
		const reposition = () => {
			const width = node.offsetWidth,
				height = node.offsetHeight,
				x = Math.max(8, Math.min(current.x, window.innerWidth - width - 8)),
				y = Math.max(42, Math.min(current.y, window.innerHeight - height - 8))
			node.style.left = `${x}px`
			node.style.top = `${y}px`
			node.style.setProperty(
				"--menu-origin",
				`${y < current.y ? "bottom" : "top"} ${x < current.x ? "right" : "left"}`,
			)
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
	class="account-context-menu account-menu game-context-menu"
	role="group"
	aria-label={`${store.displayName(place)} actions`}
	tabindex="-1"
	data-game-menu
	use:positionMenu={menu}
	in:menuIn
	out:menuOut>
	<button
		type="button"
		disabled={store.savingFavorite ||
			store.savingNickname ||
			!store.favoritesLoaded}
		onclick={() => void store.toggleFavorite(place)}>
		{#if favorite}<StarIcon weight="fill" size={16} aria-hidden="true" /><span
				>Remove from favorites</span
			>{:else}<StarIcon size={16} aria-hidden="true" /><span
				>Add to favorites</span
			>{/if}
	</button>
	{#if favorite}
		<button type="button" onclick={() => store.editNickname(place)}
			><PencilLineIcon size={16} aria-hidden="true" /><span
				>{store.nicknameFor(place.placeId)
					? "Edit nickname"
					: "Add nickname"}</span
			></button>
	{/if}
	<button
		type="button"
		onclick={() => {
			const { placeId } = place
			store.closeMenu()
			onFillLaunch(placeId)
		}}
		><ListPlusIcon size={16} aria-hidden="true" /><span>Fill launch options</span
		></button>

	<div role="group" aria-label="Copy game details">
		<div class="account-menu-heading" aria-hidden="true">Copy</div>
		<button
			type="button"
			aria-label="Copy game name"
			onclick={() => void store.copy(place.name, "Game name")}
			><TextTIcon size={16} aria-hidden="true" /><span>Name</span></button>
		<button
			type="button"
			aria-label="Copy place ID"
			onclick={() => void store.copy(place.placeId, "Place ID")}
			><HashIcon size={16} aria-hidden="true" /><span>Place ID</span></button>
		<button
			type="button"
			aria-label="Copy universe ID"
			onclick={() => void store.copy(place.universeId, "Universe ID")}
			><HashIcon size={16} aria-hidden="true" /><span>Universe ID</span></button>
		<button
			type="button"
			aria-label={`Copy ${owner.toLowerCase()} name`}
			disabled={!place.creatorName}
			onclick={() => void store.copy(place.creatorName, `${owner} name`)}>
			{#if group}<UsersThreeIcon
					size={16}
					aria-hidden="true" />{:else}<UserCircleIcon
					size={16}
					aria-hidden="true" />{/if}
			<span>{owner} name</span>
		</button>
		<button
			type="button"
			aria-label={`Copy ${owner.toLowerCase()} ID`}
			disabled={!place.creatorId}
			onclick={() => void store.copy(place.creatorId, `${owner} ID`)}
			><HashIcon size={16} aria-hidden="true" /><span>{owner} ID</span></button>
	</div>
</div>

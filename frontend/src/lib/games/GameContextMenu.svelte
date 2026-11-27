<script lang="ts">
	import globeHemisphereWestIcon from "@phosphor-icons/core/regular/globe-hemisphere-west.svg?raw"
	import hashIcon from "@phosphor-icons/core/regular/hash.svg?raw"
	import linkIcon from "@phosphor-icons/core/regular/link.svg?raw"
	import listPlusIcon from "@phosphor-icons/core/regular/list-plus.svg?raw"
	import pencilLineIcon from "@phosphor-icons/core/regular/pencil-line.svg?raw"
	import starFillIcon from "@phosphor-icons/core/fill/star-fill.svg?raw"
	import starIcon from "@phosphor-icons/core/regular/star.svg?raw"
	import textTIcon from "@phosphor-icons/core/regular/text-t.svg?raw"
	import userCircleIcon from "@phosphor-icons/core/regular/user-circle.svg?raw"
	import usersThreeIcon from "@phosphor-icons/core/regular/users-three.svg?raw"
	import MenuItem from "../shared/MenuItem.svelte"
	import MenuSection from "../shared/MenuSection.svelte"
	import PointMenu from "../shared/PointMenu.svelte"
	import {
		getRobloxPageOpener,
		openInBrowserUnavailableReason,
		robloxGameUrl,
	} from "../shared/roblox-pages"
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
	const robloxPages = getRobloxPageOpener()
	const place = $derived(menu.place),
		gameUrl = $derived(robloxGameUrl(place.placeId)),
		favorite = $derived(store.isFavorite(place.placeId)),
		group = $derived(place.creatorType === "Group"),
		owner = $derived(group ? "Group" : "Creator"),
		favoriteBusy = $derived(
			store.savingFavorite || store.savingNickname || !store.favoritesLoaded,
		)
</script>

<PointMenu
	label={`${store.displayName(place)} actions`}
	point={menu}
	owner="data-game-menu"
	onClose={() => store.closeMenu()}>
	<MenuItem
		label="Fill launch options"
		icon={listPlusIcon}
		onSelect={() => {
			const { placeId } = place
			store.closeMenu()
			onFillLaunch(placeId)
		}} />
	{#if favorite}
		<MenuItem
			label={store.nicknameFor(place.placeId) ? "Edit nickname" : "Add nickname"}
			icon={pencilLineIcon}
			onSelect={() => store.editNickname(place)} />
	{:else}
		<MenuItem
			label="Add to favorites"
			icon={starIcon}
			disabled={favoriteBusy}
			onSelect={() => void store.toggleFavorite(place)} />
	{/if}
	<MenuItem
		label="Open in browser"
		icon={globeHemisphereWestIcon}
		disabled={!robloxPages.canOpen}
		tooltip={robloxPages.canOpen ? undefined : openInBrowserUnavailableReason}
		onSelect={() => {
			const pageUrl = gameUrl
			store.closeMenu()
			robloxPages.open(pageUrl)
		}} />

	<MenuSection heading="Copy" accessibleLabel="Copy game details">
		<MenuItem
			label="Name"
			icon={textTIcon}
			accessibleLabel="Copy game name"
			onSelect={() => void store.copy(place.name, "Game name")} />
		<MenuItem
			label="Place ID"
			icon={hashIcon}
			detail={String(place.placeId)}
			accessibleLabel="Copy place ID"
			onSelect={() => void store.copy(place.placeId, "Place ID")} />
		<MenuItem
			label="Universe ID"
			icon={hashIcon}
			detail={place.universeId ? String(place.universeId) : ""}
			accessibleLabel="Copy universe ID"
			disabled={!place.universeId}
			onSelect={() => void store.copy(place.universeId, "Universe ID")} />
		<MenuItem
			label={`${owner} name`}
			icon={group ? usersThreeIcon : userCircleIcon}
			accessibleLabel={`Copy ${owner.toLowerCase()} name`}
			disabled={!place.creatorName}
			onSelect={() => void store.copy(place.creatorName, `${owner} name`)} />
		<MenuItem
			label={`${owner} ID`}
			icon={hashIcon}
			detail={place.creatorId ? String(place.creatorId) : ""}
			accessibleLabel={`Copy ${owner.toLowerCase()} ID`}
			disabled={!place.creatorId}
			onSelect={() => void store.copy(place.creatorId, `${owner} ID`)} />
		<MenuItem
			label="Game link"
			icon={linkIcon}
			accessibleLabel="Copy game link"
			onSelect={() => void store.copy(gameUrl, "Game link")} />
	</MenuSection>
	{#if favorite}
		<div class="menu-separator"></div>
		<MenuItem
			label="Remove from favorites"
			icon={starFillIcon}
			tone="danger"
			disabled={favoriteBusy}
			onSelect={() => void store.toggleFavorite(place)} />
	{/if}
</PointMenu>

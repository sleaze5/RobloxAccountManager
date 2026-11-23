<script lang="ts">
	import gameControllerIcon from "@phosphor-icons/core/regular/game-controller.svg?raw"
	import starFillIcon from "@phosphor-icons/core/fill/star-fill.svg?raw"
	import storefrontIcon from "@phosphor-icons/core/regular/storefront.svg?raw"
	import usersThreeIcon from "@phosphor-icons/core/regular/users-three.svg?raw"
	import { untrack } from "svelte"
	import type { Snippet } from "svelte"
	import type { GamePlace } from "../backend/bridge"
	import { inventoryCategoryOptionOf } from "../inventory/inventory-categories"
	import InventoryItemTile from "../inventory/InventoryItemTile.svelte"
	import { getCreatorNavigation } from "../shared/creator-navigation"
	import Icon from "../shared/Icon.svelte"
	import ItemTile from "../shared/ItemTile.svelte"
	import SkeletonTile from "../shared/SkeletonTile.svelte"
	import TileCarousel from "../shared/TileCarousel.svelte"
	import type { AccountStore } from "./account-store.svelte"
	import { profileShowcaseFor } from "./profile-showcase-state.svelte"
	import type {
		ProfileShowcase,
		ProfileShowcasePart,
	} from "./profile-showcase-state.svelte"

	let {
		accounts,
		accountId,
		userId,
		username,
		refreshCount,
		onSearchPlace,
		onShowCreations,
	}: {
		accounts: AccountStore
		// accountId is the account that reads the profile.
		accountId: number
		userId: number
		username: string
		// refreshCount changes when the profile refreshes, which reads the
		// showcase again.
		refreshCount: number
		onSearchPlace: (placeId: number) => void
		onShowCreations: (username: string) => void
	} = $props()

	const skeletonTileIndexes = Array.from({ length: 6 }, (_, index) => index),
		creatorNavigation = getCreatorNavigation(),
		headingID = $props.id()
	let showcase = $state.raw<ProfileShowcase | null>(null)

	// The effect tracks the IDs and the username alone, because the account
	// scope reads the account list, which refreshes often.
	$effect(() => {
		const shownAccountId = accountId,
			shownUserId = userId,
			shownUsername = username
		untrack(() => {
			showcase =
				shownAccountId > 0 && shownUserId > 0
					? profileShowcaseFor(
							shownAccountId,
							shownUserId,
							shownUsername,
							accounts,
						)
					: null
		})
	})

	$effect(() => {
		if (refreshCount > 0) untrack(() => showcase?.load())
	})
</script>

<!-- A part shows its tiles in a carousel. Until they load, or when there are
     none, it shows its heading over the state instead. -->
{#snippet part(
	id: ProfileShowcasePart,
	title: string,
	noun: string,
	count: number | null,
	tiles: Snippet,
)}
	{#if count}
		<TileCarousel id={`${headingID}-${id}`} label={title}>
			{#snippet heading()}
				{title}<span class="font-mono text-2xs font-normal text-fg-faint"
					>({count})</span>
			{/snippet}
			{@render tiles()}
		</TileCarousel>
	{:else}
		<section class="flex flex-col gap-2" aria-labelledby={`${headingID}-${id}`}>
			<h3
				id={`${headingID}-${id}`}
				class="flex h-6 items-center text-xs font-semibold text-fg-muted">
				{title}
			</h3>
			{#if count === 0}
				<p class="text-xs text-fg-muted">None</p>
			{:else if showcase?.failedParts.includes(id)}
				<p class="text-xs text-fg-muted">Unavailable</p>
			{:else}
				<ul class="flex gap-2 overflow-x-hidden" aria-hidden="true">
					{#each skeletonTileIndexes as index (index)}
						<li class="w-36 shrink-0"><SkeletonTile /></li>
					{/each}
				</ul>
				<span class="sr-only" role="status">Loading {noun}…</span>
			{/if}
		</section>
	{/if}
{/snippet}

{#snippet gameTiles(places: GamePlace[], showsCreator: boolean)}
	{#each places as place (place.placeId)}
		<li class="w-36 shrink-0">
			<ItemTile
				name={place.name}
				imageUrl={place.iconUrl}
				fallbackIcon={gameControllerIcon}
				detail={showsCreator ? place.creatorName : ""}
				actionLabel="Show in Games"
				onOpen={() => onSearchPlace(place.placeId)} />
		</li>
	{/each}
{/snippet}

{#if showcase}
	{@const groups = showcase.groups}
	{@const favoriteGames = showcase.favoriteGames}
	{@const createdGames = showcase.createdGames}
	{@const creations = showcase.creations}
	<div class="flex min-w-0 flex-col gap-4">
		{#snippet groupTiles()}
			{#each groups ?? [] as userGroup (userGroup.group.groupId)}
				<li class="w-36 shrink-0">
					<ItemTile
						name={userGroup.group.name}
						imageUrl={userGroup.group.iconUrl}
						fallbackIcon={usersThreeIcon}
						badges={userGroup.group.groupId === showcase?.primaryGroupId
							? [
									{
										label: "Primary group",
										icon: starFillIcon,
										tone: "text-fg",
									},
								]
							: []}
						detail={userGroup.role.name}
						actionLabel="Show in Groups"
						onOpen={() =>
							creatorNavigation.showGroup(userGroup.group.groupId)} />
				</li>
			{/each}
		{/snippet}
		{#snippet favoriteGameTiles()}{@render gameTiles(
				favoriteGames ?? [],
				true,
			)}{/snippet}
		{#snippet createdGameTiles()}{@render gameTiles(
				createdGames ?? [],
				false,
			)}{/snippet}
		{@render part("groups", "Groups", "groups", groups?.length ?? null, groupTiles)}
		{@render part(
			"favoriteGames",
			"Favorite games",
			"favorite games",
			favoriteGames?.length ?? null,
			favoriteGameTiles,
		)}
		{@render part(
			"createdGames",
			"Games",
			"games",
			createdGames?.length ?? null,
			createdGameTiles,
		)}
		{#if creations?.length}
			<TileCarousel id={`${headingID}-creations`} label="Marketplace creations">
				{#snippet heading()}
					Marketplace creations<span
						class="font-mono text-2xs font-normal text-fg-faint"
						>({creations.length}{showcase?.hasMoreCreations
							? "+"
							: ""})</span>
				{/snippet}
				{#snippet actions()}
					<button
						class="button"
						type="button"
						onclick={() => onShowCreations(username)}>
						<Icon svg={storefrontIcon} size={16} aria-hidden="true" />
						Show in Marketplace
					</button>
				{/snippet}
				{#each creations as categorizedItem, index (`${categorizedItem.item.itemId}:${index}`)}
					<li class="w-36 shrink-0">
						<InventoryItemTile
							item={categorizedItem.item}
							category={inventoryCategoryOptionOf(
								categorizedItem.category,
							)}
							isWorn={false}
							isOwned={false}
							actionLabel="Show in Marketplace"
							onOpen={() => onShowCreations(username)} />
					</li>
				{/each}
			</TileCarousel>
		{/if}
	</div>
{/if}

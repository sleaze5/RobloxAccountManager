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
	import { itemGridClass } from "../inventory/item-grid"
	import { getCreatorNavigation } from "../shared/creator-navigation"
	import Icon from "../shared/Icon.svelte"
	import ItemTile from "../shared/ItemTile.svelte"
	import SkeletonTile from "../shared/SkeletonTile.svelte"
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

	// A collapsed part shows two rows of the widest grid.
	const collapsedTileCount = 12,
		skeletonTileIndexes = Array.from({ length: 6 }, (_, index) => index),
		creatorNavigation = getCreatorNavigation(),
		headingID = $props.id()
	let showcase = $state.raw<ProfileShowcase | null>(null),
		expandedParts = $state<ProfileShowcasePart[]>([])

	// The effect tracks the IDs and the username alone, because the account
	// scope reads the account list, which refreshes often.
	$effect(() => {
		const shownAccountId = accountId,
			shownUserId = userId,
			shownUsername = username
		untrack(() => {
			expandedParts = []
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

	function shownOf<Item>(part: ProfileShowcasePart, items: Item[]): Item[] {
		return expandedParts.includes(part) ? items : items.slice(0, collapsedTileCount)
	}

	function toggle(part: ProfileShowcasePart): void {
		expandedParts = expandedParts.includes(part)
			? expandedParts.filter((expandedPart) => expandedPart !== part)
			: [...expandedParts, part]
	}
</script>

{#snippet part(
	id: ProfileShowcasePart,
	title: string,
	noun: string,
	count: number | null,
	tiles: Snippet,
)}
	<section class="flex flex-col gap-2" aria-labelledby={`${headingID}-${id}`}>
		<h3
			id={`${headingID}-${id}`}
			class="flex h-6 items-center gap-1.5 text-xs font-semibold text-fg-muted">
			{title}
			{#if count}<span class="font-mono text-2xs font-normal text-fg-faint"
					>({count})</span
				>{/if}
			{#if count && count > collapsedTileCount}
				<button
					class="ml-auto button-text"
					type="button"
					aria-expanded={expandedParts.includes(id)}
					onclick={() => toggle(id)}
					>{expandedParts.includes(id)
						? "Show fewer"
						: `Show all ${noun}`}</button>
			{/if}
		</h3>
		{#if showcase?.failedParts.includes(id) && count === null}
			<p class="text-xs text-fg-muted">Unavailable</p>
		{:else if count === null}
			<ul class={itemGridClass} aria-hidden="true">
				{#each skeletonTileIndexes as index (index)}
					<SkeletonTile />
				{/each}
			</ul>
			<span class="sr-only" role="status">Loading {noun}…</span>
		{:else if count === 0}
			<p class="text-xs text-fg-muted">None</p>
		{:else}
			{@render tiles()}
		{/if}
	</section>
{/snippet}

{#snippet gameTiles(
	id: ProfileShowcasePart,
	places: GamePlace[],
	showsCreator: boolean,
)}
	<ul
		class={itemGridClass}
		aria-label={id === "favoriteGames" ? "Favorite games" : "Games"}>
		{#each shownOf(id, places) as place (place.placeId)}
			<li class="min-w-0">
				<ItemTile
					name={place.name}
					imageUrl={place.iconUrl}
					fallbackIcon={gameControllerIcon}
					detail={showsCreator ? place.creatorName : ""}
					actionLabel="Show in Games"
					onOpen={() => onSearchPlace(place.placeId)} />
			</li>
		{/each}
	</ul>
{/snippet}

{#if showcase}
	{@const groups = showcase.groups}
	{@const favoriteGames = showcase.favoriteGames}
	{@const createdGames = showcase.createdGames}
	{@const creations = showcase.creations}
	<div class="@container flex flex-col gap-4">
		{#snippet groupTiles()}
			<ul class={itemGridClass} aria-label="Groups">
				{#each shownOf("groups", groups ?? []) as userGroup (userGroup.group.groupId)}
					<li class="min-w-0">
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
			</ul>
		{/snippet}
		{#snippet favoriteGameTiles()}{@render gameTiles(
				"favoriteGames",
				favoriteGames ?? [],
				true,
			)}{/snippet}
		{#snippet createdGameTiles()}{@render gameTiles(
				"createdGames",
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
			<section
				class="flex flex-col gap-2"
				aria-labelledby={`${headingID}-creations`}>
				<h3
					id={`${headingID}-creations`}
					class="flex h-6 items-center gap-1.5 text-xs font-semibold text-fg-muted">
					Marketplace creations
					<span class="font-mono text-2xs font-normal text-fg-faint"
						>({creations.length}{showcase.hasMoreCreations
							? "+"
							: ""})</span>
					<button
						class="ml-auto button"
						type="button"
						onclick={() => onShowCreations(username)}>
						<Icon svg={storefrontIcon} size={16} aria-hidden="true" />
						Show in Marketplace
					</button>
				</h3>
				<ul class={itemGridClass} aria-label="Marketplace creations">
					{#each creations.slice(0, collapsedTileCount) as categorizedItem, index (`${categorizedItem.item.itemId}:${index}`)}
						<li class="min-w-0">
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
				</ul>
			</section>
		{/if}
	</div>
{/if}

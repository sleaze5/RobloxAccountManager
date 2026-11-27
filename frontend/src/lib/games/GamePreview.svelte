<script lang="ts">
	import calendarDotsIcon from "@phosphor-icons/core/regular/calendar-dots.svg?raw"
	import clockCounterClockwiseIcon from "@phosphor-icons/core/regular/clock-counter-clockwise.svg?raw"
	import shapesIcon from "@phosphor-icons/core/regular/shapes.svg?raw"
	import starFillIcon from "@phosphor-icons/core/fill/star-fill.svg?raw"
	import starIcon from "@phosphor-icons/core/regular/star.svg?raw"
	import usersThreeIcon from "@phosphor-icons/core/regular/users-three.svg?raw"
	import warningIcon from "@phosphor-icons/core/regular/warning.svg?raw"
	import { accountBackend, GameSection } from "../backend/bridge"
	import type { Game } from "../backend/bridge"
	import CreatorByline from "../shared/CreatorByline.svelte"
	import { errorMessage } from "../shared/errors"
	import { notifyErrors } from "../notifications/error-notifications.svelte"
	import Icon from "../shared/Icon.svelte"
	import ProfilePreviewLayout from "../shared/ProfilePreviewLayout.svelte"
	import type { ProfilePreviewFrame } from "../shared/ProfilePreviewLayout.svelte"
	import ProfileStats from "../shared/ProfileStats.svelte"
	import Skeleton from "../shared/Skeleton.svelte"
	import Timestamp from "../shared/Timestamp.svelte"
	import { relativeTime } from "../shared/timestamp"
	import GameIcon from "./GameIcon.svelte"
	import { gameHeadlineStats } from "./game-stats"
	import { getGamesStore } from "./games-store.svelte"
	import GameSubplaces from "./GameSubplaces.svelte"
	import PlaceIDs from "./PlaceIDs.svelte"

	// GamePreview shows the information of a place: its numbers, description,
	// and main details. A subplace shows only its own details, because the
	// others belong to its universe.
	let { placeId, frame }: { placeId: number; frame: ProfilePreviewFrame } = $props()

	const games = getGamesStore(),
		numbers = new Intl.NumberFormat(),
		sectionHeadingClass = "text-xs font-semibold text-fg-muted"

	let game = $state.raw<Game | null>(null),
		loadError = $state(""),
		isLoadingVotes = $state(true),
		hasVotes = $state(false),
		loadCount = $state(0)

	// Favorites load with Games, which may not have opened yet.
	$effect(() => {
		if (!games.favoritesLoaded) void games.loadFavorites()
	})

	$effect(() => {
		void loadCount
		const request = accountBackend.GetGame(0, placeId)
		let votesRequest: ReturnType<typeof accountBackend.GetGameSection> | null =
				null,
			isCurrent = true
		game = null
		loadError = ""
		isLoadingVotes = true
		hasVotes = false
		void (async () => {
			try {
				const loadedGame = await request
				if (!isCurrent) return
				game = loadedGame
				votesRequest = accountBackend.GetGameSection(
					loadedGame.place.universeId,
					loadedGame.place.placeId,
					GameSection.GameSectionVotes,
				)
				const votes = await votesRequest
				if (!isCurrent) return
				game = {
					...loadedGame,
					upVotes: votes.upVotes,
					downVotes: votes.downVotes,
				}
				hasVotes = true
			} catch (error) {
				// Without the votes, the rating says that it has none.
				if (isCurrent && !game)
					loadError = errorMessage(
						error,
						"Could not load the game. Try again.",
					)
			} finally {
				if (isCurrent) isLoadingVotes = false
			}
		})()
		return () => {
			isCurrent = false
			void request.cancel()
			void votesRequest?.cancel()
		}
	})

	const place = $derived(game?.place ?? null),
		rootPlace = $derived(game?.rootPlace ?? null),
		listedGame = $derived(game?.isUnlisted ? undefined : game),
		stats = $derived(
			gameHeadlineStats(game, !game && !loadError, {
				isLoaded: hasVotes,
				isPending: isLoadingVotes,
			}),
		),
		isFavorite = $derived(!!place && games.isFavorite(place.placeId)),
		genre = $derived(
			listedGame &&
				[listedGame.genre, listedGame.subgenre].filter(Boolean).join("; "),
		)

	notifyErrors("Action not completed", () => games.error)
</script>

{#snippet quickActions()}
	{#if place}
		{@const favoritePlace = place}
		<button
			class="button-icon"
			type="button"
			aria-label="Favorite"
			aria-pressed={isFavorite}
			aria-busy={games.savingFavorite}
			data-tooltip={isFavorite ? "Remove from favorites" : "Add to favorites"}
			disabled={games.savingFavorite || !games.favoritesLoaded}
			onclick={() => void games.toggleFavorite(favoritePlace)}>
			{#if isFavorite}
				<Icon
					svg={starFillIcon}
					class="animate-pop text-accent-strong"
					size={16}
					aria-hidden="true" />
			{:else}
				<Icon svg={starIcon} size={16} aria-hidden="true" />
			{/if}
		</button>
	{/if}
{/snippet}

{#snippet dateFact(label: string, icon: string, valueMs: number)}
	<div>
		<dt><Icon svg={icon} size={16} aria-hidden="true" />{label}</dt>
		<dd>
			<Timestamp value={valueMs} />
			<small>{relativeTime(valueMs)}</small>
		</dd>
	</div>
{/snippet}

<ProfilePreviewLayout {frame} quickActions={place ? quickActions : undefined}>
	{#snippet image()}
		<GameIcon url={place?.iconUrl ?? ""} size="xl" />
	{/snippet}
	{#snippet title()}
		{#if place}{games.displayName(place)}{:else if loadError}Game unavailable{:else}<Skeleton
				className="w-48"
				label="Loading game…" />{/if}
	{/snippet}
	{#snippet summary()}
		<PlaceIDs
			{placeId}
			universeId={place?.universeId ?? 0}
			rootPlaceId={rootPlace?.placeId ?? 0} />
		{#if rootPlace}
			<p class="text-xs text-fg-muted">
				Subplace of <span class="font-medium text-fg"
					>{games.displayName(rootPlace)}</span>
			</p>
		{/if}
		{#if place && (place.creatorType === "User" || place.creatorType === "Group") && place.creatorId}
			<CreatorByline
				kind={place.creatorType}
				creatorId={place.creatorId}
				name={place.creatorName}
				hasVerifiedBadge={place.creatorVerified} />
		{:else if !place && !loadError}
			<span class="text-xs"><Skeleton className="w-32" label="" /></span>
		{/if}
		{#if !rootPlace}<ProfileStats label="Game numbers" {stats} />{/if}
	{/snippet}

	{#if loadError}
		<div class="empty-state py-6">
			<Icon svg={warningIcon} size={28} aria-hidden="true" />
			<p>{loadError}</p>
			<button class="button" type="button" onclick={() => loadCount++}
				>Try again</button>
		</div>
	{:else if game}
		{#if game.description}
			<section class="flex flex-col gap-3" aria-label="Description">
				<h3 class={sectionHeadingClass}>Description</h3>
				<p
					class="max-h-40 text-block overflow-y-auto select-text"
					data-variant="prose">
					{game.description}
				</p>
			</section>
		{/if}
		<section class="flex flex-col gap-3" aria-label="Details">
			<h3 class={sectionHeadingClass}>Details</h3>
			<dl class="facts">
				{#if !rootPlace && genre}
					<div>
						<dt>
							<Icon svg={shapesIcon} size={16} aria-hidden="true" />Genre
						</dt>
						<dd>{genre}</dd>
					</div>
				{/if}
				{#if !rootPlace && listedGame}
					<div>
						<dt>
							<Icon
								svg={usersThreeIcon}
								size={16}
								aria-hidden="true" />Server capacity
						</dt>
						<dd>
							{numbers.format(listedGame.maxPlayers)}
							{listedGame.maxPlayers === 1 ? "player" : "players"}
						</dd>
					</div>
				{/if}
				{#if game.updatedAtMs}{@render dateFact(
						"Updated",
						clockCounterClockwiseIcon,
						game.updatedAtMs,
					)}{/if}
				{#if game.createdAtMs}{@render dateFact(
						"Created",
						calendarDotsIcon,
						game.createdAtMs,
					)}{/if}
			</dl>
		</section>
		{#if !rootPlace}<GameSubplaces universeId={game.place.universeId} />{/if}
	{:else}
		<span class="h-24 w-full skeleton rounded-lg" role="status"
			><span class="sr-only">Loading game…</span></span>
	{/if}
</ProfilePreviewLayout>

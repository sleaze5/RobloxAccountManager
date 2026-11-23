<script lang="ts">
	import LoadMore from "../shared/LoadMore.svelte"
	import gameControllerIcon from "@phosphor-icons/core/regular/game-controller.svg?raw"
	import lockIcon from "@phosphor-icons/core/regular/lock.svg?raw"
	import warningIcon from "@phosphor-icons/core/regular/warning.svg?raw"
	import { onMount, untrack } from "svelte"
	import { accountBackend } from "../backend/bridge"
	import type { GroupGame, GroupGamePage } from "../backend/bridge"
	import { CursorPagedList } from "../shared/cursor-paged-list.svelte"
	import Icon from "../shared/Icon.svelte"
	import ProfileImage from "../shared/ProfileImage.svelte"
	import RefreshIcon from "../shared/RefreshIcon.svelte"
	import SkeletonTile from "../shared/SkeletonTile.svelte"
	import { relativeTime } from "../shared/timestamp"
	import GroupSubpageHeader from "./GroupSubpageHeader.svelte"
	import type { GroupsStore } from "./groups-store.svelte"

	let {
		store,
		onSearchPlace,
	}: {
		store: GroupsStore
		onSearchPlace: (placeId: number) => void
	} = $props()

	type GameFilter = "all" | "public" | "private"

	const group = untrack(() => store.profile!.details.group),
		compactNumbers = new Intl.NumberFormat("en-US", {
			notation: "compact",
			maximumFractionDigits: 1,
		}),
		// A game tile shows a long name and its visits, so the grid has fewer
		// and wider columns than the item grid of the Marketplace.
		gameGridClass =
			"grid grid-cols-2 gap-2 @md:grid-cols-3 @2xl:grid-cols-4 @4xl:grid-cols-5 @6xl:grid-cols-6",
		skeletonTileIndexes = Array.from({ length: 8 }, (_unused, index) => index),
		filterOptions: { value: GameFilter; label: string }[] = [
			{ value: "all", label: "All" },
			{ value: "public", label: "Public" },
			{ value: "private", label: "Private" },
		],
		games = untrack(
			() =>
				new CursorPagedList<GroupGamePage, GroupGame>(
					store.scope!,
					(cursor) =>
						accountBackend.GetGroupGames(
							store.accountId,
							group.groupId,
							group.name,
							cursor,
						),
					(page) => page.games,
					"Could not load the games. Try again.",
				),
		)
	let filter = $state<GameFilter>("all"),
		gameListElement = $state<HTMLElement | undefined>(undefined)

	const privateCount = $derived(games.items.filter((game) => game.isPrivate).length),
		shownGames = $derived(
			filter === "all"
				? games.items
				: games.items.filter(
						(game) => game.isPrivate === (filter === "private"),
					),
		)

	onMount(() => void games.refresh())

	function showFilter(nextFilter: GameFilter): void {
		filter = nextFilter
		gameListElement?.scrollTo({ top: 0 })
	}
</script>

<section class="settings-page min-w-0" aria-labelledby="group-games-title">
	<GroupSubpageHeader
		titleId="group-games-title"
		title="Games"
		groupName={group.name}
		onBack={() => (store.subpage = null)}>
		<button
			class="button-ghost"
			type="button"
			aria-label={games.isLoading ? "Refreshing games…" : "Refresh games"}
			data-tooltip="Refresh games"
			data-tooltip-side="bottom-end"
			aria-busy={games.isLoading}
			disabled={games.isLoading || games.isLoadingMore}
			onclick={() => void games.refresh()}>
			<RefreshIcon isRefreshing={games.isLoading && games.hasLoaded} />
		</button>
	</GroupSubpageHeader>
	<div
		class="flex min-h-10 shrink-0 items-center gap-2 border-b border-line-subtle px-3 py-1.5"
		role="toolbar"
		aria-label="Filter games">
		<div class="segmented" role="group" aria-label="Show games">
			{#each filterOptions as option (option.value)}
				<button
					type="button"
					aria-pressed={filter === option.value}
					disabled={!games.hasLoaded}
					onclick={() => showFilter(option.value)}>{option.label}</button>
			{/each}
		</div>
		{#if games.hasLoaded}
			<span class="animate-enter text-2xs text-fg-muted" aria-live="polite"
				>{games.items.length}{games.nextCursor ? "+" : ""} loaded · {privateCount}
				private</span>
		{/if}
	</div>
	<div
		class="min-h-0 flex-1 scrollbar-stable overflow-y-auto"
		data-load-more-root
		bind:this={gameListElement}>
		{#if games.error && games.hasLoaded}
			<p class="mx-3 mt-3 notice-error" role="alert">{games.error}</p>
		{/if}
		{#if !games.hasLoaded && games.error}
			<div class="empty-state h-full" role="alert">
				<Icon svg={warningIcon} size={28} aria-hidden="true" />
				<h2>Games unavailable</h2>
				<p>{games.error}</p>
				<button
					class="button"
					type="button"
					onclick={() => void games.refresh()}>Try again</button>
			</div>
		{:else if !games.hasLoaded}
			<div class="p-3" role="status" aria-busy="true">
				<span class="sr-only">Loading games…</span>
				<ul class={[gameGridClass, "animate-enter"]} aria-hidden="true">
					{#each skeletonTileIndexes as index (index)}
						<SkeletonTile />
					{/each}
				</ul>
			</div>
		{:else if shownGames.length === 0}
			<div class="empty-state h-full" role="status">
				<Icon svg={gameControllerIcon} size={28} aria-hidden="true" />
				<h2>{games.items.length === 0 ? "No games" : "No matching games"}</h2>
				<p>
					{games.items.length === 0
						? `${group.name} has no games.`
						: "No loaded game matches this filter."}
				</p>
				{#if games.items.length > 0}
					<button
						class="button"
						type="button"
						onclick={() => showFilter("all")}>Show all games</button>
				{/if}
			</div>
		{:else}
			<ul
				class={[gameGridClass, "animate-enter p-3"]}
				aria-label={`Games by ${group.name}`}>
				{#each shownGames as game (game.place.universeId)}
					{@const name = game.place.name || "Unnamed game"}
					<li class="min-w-0">
						<button
							class="group/tile flex h-full w-full min-w-0 flex-col gap-1 rounded-lg border border-line-subtle bg-surface p-1 text-left transition-colors duration-100 hover:border-line hover:bg-hover"
							type="button"
							data-tooltip="Show in Games"
							data-tooltip-strong={name}
							onclick={() => onSearchPlace(game.place.placeId)}>
							<span
								class="relative grid aspect-square w-full place-items-center overflow-hidden rounded-sm bg-raised">
								<span
									class="grid size-full place-items-center transition-transform duration-150 ease-out motion-safe:group-hover/tile:scale-105"
									aria-hidden="true">
									<ProfileImage
										url={game.place.iconUrl}
										fallbackIcon={gameControllerIcon} />
								</span>
								{#if game.isPrivate}
									<span
										class="absolute bottom-1 left-1 inline-flex h-5 items-center gap-1 rounded-sm border border-line-strong bg-overlay px-1 text-2xs font-medium text-fg"
										><Icon
											svg={lockIcon}
											class="text-warning"
											size={12}
											aria-hidden="true" />Private</span>
								{/if}
							</span>
							<span class="flex min-w-0 flex-col px-1 pb-0.5">
								<span class="truncate text-xs font-medium text-fg"
									>{name}</span>
								<span
									class="flex min-w-0 items-baseline justify-between gap-2 text-2xs">
									<span class="shrink-0 font-mono text-fg"
										>{compactNumbers.format(game.visits)}
										{game.visits === 1 ? "visit" : "visits"}</span>
									{#if game.updatedAtMs}
										<span class="truncate text-fg-muted"
											>{relativeTime(game.updatedAtMs)}</span>
									{/if}
								</span>
							</span>
						</button>
					</li>
				{/each}
			</ul>
			{#if games.nextCursor}
				<LoadMore
					pageKey={games.items.length}
					label="Load more games"
					loadingLabel="Loading more games…"
					isLoadingMore={games.isLoadingMore}
					hasFailed={!!games.error}
					onLoadMore={() => void games.loadMore()} />
			{/if}
		{/if}
	</div>
</section>

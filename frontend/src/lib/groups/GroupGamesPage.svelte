<script lang="ts">
	import circleNotchIcon from "@phosphor-icons/core/regular/circle-notch.svg?raw"
	import gameControllerIcon from "@phosphor-icons/core/regular/game-controller.svg?raw"
	import lockIcon from "@phosphor-icons/core/regular/lock.svg?raw"
	import warningIcon from "@phosphor-icons/core/regular/warning.svg?raw"
	import { onMount, untrack } from "svelte"
	import { accountBackend } from "../backend/bridge"
	import type { GroupGame, GroupGamePage } from "../backend/bridge"
	import GameIcon from "../games/GameIcon.svelte"
	import { CursorPagedList } from "../shared/cursor-paged-list.svelte"
	import Icon from "../shared/Icon.svelte"
	import { loadMoreWhenVisible } from "../shared/load-more-when-visible"
	import { moveListFocus } from "../shared/move-shortcut"
	import RefreshIcon from "../shared/RefreshIcon.svelte"
	import SkeletonList from "../shared/SkeletonList.svelte"
	import SkeletonMediaRow from "../shared/SkeletonMediaRow.svelte"
	import Timestamp from "../shared/Timestamp.svelte"
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
	let filter = $state<GameFilter>("all")

	const privateCount = $derived(games.items.filter((game) => game.isPrivate).length),
		shownGames = $derived(
			filter === "all"
				? games.items
				: games.items.filter(
						(game) => game.isPrivate === (filter === "private"),
					),
		)

	onMount(() => void games.refresh())
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
					onclick={() => (filter = option.value)}>{option.label}</button>
			{/each}
		</div>
		{#if games.hasLoaded}
			<span class="animate-enter text-2xs text-fg-muted" aria-live="polite"
				>{games.items.length}{games.nextCursor ? "+" : ""} loaded · {privateCount}
				private</span>
		{/if}
	</div>
	<div class="min-h-0 flex-1 scrollbar-stable overflow-y-auto" data-load-more-root>
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
			<SkeletonList label="Loading games…">
				{#snippet row(index)}<SkeletonMediaRow {index} />{/snippet}
			</SkeletonList>
		{:else if shownGames.length === 0}
			<div class="empty-state h-full" role="status">
				<Icon svg={gameControllerIcon} size={28} aria-hidden="true" />
				<h2>{games.items.length === 0 ? "No games" : "No matching games"}</h2>
				<p>
					{games.items.length === 0
						? `${group.name} has no games.`
						: "No loaded game matches this filter. Select All to list every game."}
				</p>
				{#if games.items.length > 0}
					<button
						class="button"
						type="button"
						onclick={() => (filter = "all")}>Show all games</button>
				{/if}
			</div>
		{:else}
			<ul
				class="animate-enter"
				role="group"
				aria-label={`Games by ${group.name}`}>
				{#each shownGames as game, index (game.place.universeId)}
					<li>
						<button
							class="list-row relative flex h-12 w-full items-center gap-2 border-b border-line-subtle px-3 text-left"
							type="button"
							data-list-item
							tabindex={index === 0 ? 0 : -1}
							data-tooltip="Show in Games"
							data-tooltip-strong={game.place.name}
							onclick={() => onSearchPlace(game.place.placeId)}
							onkeydown={moveListFocus}>
							<GameIcon url={game.place.iconUrl} />
							<span class="flex min-w-0 flex-1 flex-col leading-snug">
								<strong
									class="inline-flex min-w-0 items-center gap-1 text-xs font-semibold text-fg"
									><span class="truncate"
										>{game.place.name || "Unnamed game"}</span
									></strong>
								<span class="truncate font-mono text-2xs text-fg-muted"
									>Place ID {game.place.placeId} · Universe ID {game
										.place.universeId}</span>
							</span>
							{#if game.isPrivate}
								<span class="tag-pill-warning shrink-0">
									<Icon svg={lockIcon} size={12} aria-hidden="true" />
									<span>Private</span>
								</span>
							{/if}
							<span
								class="flex w-28 shrink-0 flex-col items-end text-2xs text-fg-muted @max-lg:hidden">
								<span class="font-mono text-fg"
									>{compactNumbers.format(game.visits)} visits</span>
								{#if game.updatedAtMs}
									<span
										>Updated <Timestamp
											value={game.updatedAtMs} /></span>
								{/if}
							</span>
						</button>
					</li>
				{/each}
			</ul>
			{#if games.nextCursor}
				<div
					class="flex min-h-12 items-center justify-center text-fg-faint"
					use:loadMoreWhenVisible={{
						pageKey: games.items.length,
						loadMore: () => {
							if (!games.error) void games.loadMore()
						},
					}}>
					{#if games.error}
						<button
							class="button"
							type="button"
							onclick={() => void games.loadMore()}
							>Load more games</button>
					{:else}
						<Icon
							svg={circleNotchIcon}
							class="spinner"
							size={16}
							aria-label="Loading more games…" />
					{/if}
				</div>
			{/if}
		{/if}
	</div>
</section>

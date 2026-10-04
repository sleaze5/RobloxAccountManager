<script lang="ts">
	import Activity from "@lucide/svelte/icons/activity"
	import AlertTriangle from "@lucide/svelte/icons/triangle-alert"
	import BadgeCheck from "@lucide/svelte/icons/badge-check"
	import CalendarDays from "@lucide/svelte/icons/calendar-days"
	import Copy from "@lucide/svelte/icons/copy"
	import Eye from "@lucide/svelte/icons/eye"
	import Footprints from "@lucide/svelte/icons/footprints"
	import GitCommitHorizontal from "@lucide/svelte/icons/git-commit-horizontal"
	import History from "@lucide/svelte/icons/rotate-ccw-clock"
	import LoaderCircle from "@lucide/svelte/icons/loader-circle"
	import Mic from "@lucide/svelte/icons/mic"
	import MoreHorizontal from "@lucide/svelte/icons/ellipsis"
	import Network from "@lucide/svelte/icons/network"
	import Pencil from "@lucide/svelte/icons/pencil"
	import PersonStanding from "@lucide/svelte/icons/person-standing"
	import RefreshCw from "@lucide/svelte/icons/refresh-cw"
	import Server from "@lucide/svelte/icons/server"
	import ShieldCheck from "@lucide/svelte/icons/shield-check"
	import Shapes from "@lucide/svelte/icons/shapes"
	import Shirt from "@lucide/svelte/icons/shirt"
	import Star from "@lucide/svelte/icons/star"
	import ThumbsDown from "@lucide/svelte/icons/thumbs-down"
	import ThumbsUp from "@lucide/svelte/icons/thumbs-up"
	import Ticket from "@lucide/svelte/icons/ticket"
	import Upload from "@lucide/svelte/icons/upload"
	import UsersRound from "@lucide/svelte/icons/users-round"
	import Video from "@lucide/svelte/icons/video"
	import { untrack } from "svelte"
	import CompactNumber from "../shared/CompactNumber.svelte"
	import Timestamp from "../shared/Timestamp.svelte"
	import { creatorByline, type GamesStore } from "./games-store.svelte"
	import GameIcon from "./GameIcon.svelte"

	let { store }: { store: GamesStore } = $props()
	const numbers = new Intl.NumberFormat(),
		percent = new Intl.NumberFormat(undefined, {
			style: "percent",
			maximumFractionDigits: 1,
		})
	const place = $derived(store.selected),
		game = $derived(store.game),
		favorite = $derived(place ? store.isFavorite(place.placeId) : false),
		nickname = $derived(place ? store.nicknameFor(place.placeId) : ""),
		menuOpen = $derived(!!place && store.menu?.place.placeId === place.placeId),
		rootPlace = $derived(game?.rootPlace ?? null),
		votes = $derived(game ? game.upVotes + game.downVotes : 0),
		likeShare = $derived(game && votes ? game.upVotes / votes : 0),
		genre = $derived(
			game &&
				([game.genre, game.subgenre].filter(Boolean).join("; ") ||
					"Not specified"),
		),
		contentRating = $derived(
			game &&
				[game.maturity || "Unknown", ...(game.maturityDescriptors ?? [])].join(
					"; ",
				),
		),
		activity = $derived([
			{ label: "Playing", icon: Activity, value: game?.playing },
			{ label: "Visits", icon: Eye, value: game?.visits },
			{ label: "Favorites", icon: Star, value: game?.favorites },
			{ label: "Server capacity", icon: UsersRound, value: game?.maxPlayers },
		]),
		supported = (value: boolean | undefined) =>
			value === undefined ? "Unavailable" : value ? "Supported" : "Not supported",
		versions = $derived([
			{
				label: "Saved version",
				icon: GitCommitHorizontal,
				value: game?.versions?.saved,
			},
			{
				label: "Published version",
				icon: Upload,
				value: game?.versions?.published,
			},
		]),
		restrictions = $derived(
			game
				? [
						{
							label: "Access",
							icon: Ticket,
							value:
								game.price > 0
									? `Paid access · R$ ${numbers.format(game.price)}`
									: "Free",
						},
						{
							label: "Avatar type",
							icon: PersonStanding,
							value: game.avatarType || "Unknown",
						},
						{
							label: "Animations",
							icon: Footprints,
							value: game.avatarRules
								? game.avatarRules.customAnimationsAllowed
									? "Custom allowed"
									: "Roblox defaults only"
								: "Unknown",
						},
						...(game.avatarRules?.itemOverrides
							? [
									{
										label: "Avatar items",
										icon: Shirt,
										value: `${numbers.format(game.avatarRules.itemOverrides)} replaced by the game`,
									},
								]
							: []),
						{
							label: "Private servers",
							icon: Server,
							value: game.privateServersAllowed
								? "Allowed"
								: "Not allowed",
						},
						...(rootPlace
							? []
							: [
									{
										label: "Copying",
										icon: Copy,
										value: game.copyingAllowed
											? "Allowed"
											: "Not allowed",
									},
								]),
					]
				: [],
		)
	let draft = $state(""),
		editing = $state(false)

	$effect(() => {
		if (store.nicknameRequest !== place?.placeId) return
		untrack(() => {
			store.nicknameRequest = null
			startEditing()
		})
	})

	function startEditing(): void {
		draft = nickname
		editing = true
	}

	function saveNickname(): void {
		if (!editing || !place) return
		editing = false
		void store.saveNickname(place, draft)
	}

	function focusInput(node: HTMLInputElement): void {
		node.focus()
		node.select()
	}
</script>

{#if place}
	<section
		class="game-details"
		aria-label={store.displayName(place)}
		aria-busy={store.loadingDetails}>
		<div class="game-details-content">
			<div class="profile-head">
				<div class="selected-identity">
					<GameIcon url={place.iconUrl} large />
					<div class="identity-copy">
						<div class="game-title">
							<h2 class="game-name">{place.name}</h2>
							{#if favorite}<Star
									class="favorite-name-star"
									size={13}
									fill="currentColor"
									aria-label="Favorite" />{/if}
							{#if favorite}
								<span
									class="game-nickname"
									class:empty={!nickname && !editing}>
									{#if editing}
										<input
											class="game-nickname-input"
											type="text"
											autocomplete="off"
											placeholder="Nickname"
											aria-label="Nickname"
											maxlength={50}
											bind:value={draft}
											use:focusInput
											onblur={saveNickname}
											onkeydown={(event) => {
												if (event.key === "Enter")
													saveNickname()
												else if (event.key === "Escape") {
													event.stopPropagation()
													editing = false
												}
											}} />
									{:else}
										{#if nickname}<span class="game-nickname-text"
												>{nickname}</span
											>{/if}
										<button
											class="icon-action game-nickname-edit"
											type="button"
											aria-label={nickname
												? "Edit nickname"
												: "Add nickname"}
											data-tooltip={nickname
												? "Edit nickname"
												: "Add nickname"}
											disabled={store.savingNickname}
											onclick={startEditing}
											><Pencil size={13} /></button>
									{/if}
								</span>
							{/if}
						</div>
						<div class="identity-meta">
							<span>ID: {place.placeId}</span>
							<span class="meta-separator" aria-hidden="true">-</span>
							<span>Universe ID: {place.universeId}</span>
						</div>
						<div class="identity-meta">
							<span class="game-creator"
								>{creatorByline(
									place,
								)}{#if place.creatorVerified}<BadgeCheck
										size={12}
										aria-label="Verified creator" />{/if}</span>
							{#if place.creatorId}
								<span class="meta-separator" aria-hidden="true">-</span>
								<span>{place.creatorType} ID: {place.creatorId}</span>
							{/if}
						</div>
					</div>
				</div>
				<div class="header-actions">
					<button
						class="control-button"
						type="button"
						aria-label={store.loadingDetails
							? "Refreshing game details"
							: "Refresh game details"}
						aria-busy={store.loadingDetails}
						data-tooltip="Refresh game details"
						disabled={store.loadingDetails}
						onclick={() => void store.refreshDetails()}>
						{#if store.loadingDetails}<LoaderCircle
								class="spinner"
								size={15}
								aria-hidden="true" />{:else}<RefreshCw
								size={15}
								aria-hidden="true" />{/if}
					</button>
					<button
						class="control-button"
						type="button"
						aria-label="Servers"
						data-tooltip="Servers"
						data-game-servers
						onclick={() => {
							store.closeMenu()
							store.detailsPage = "servers"
						}}>
						<Server size={15} aria-hidden="true" />
					</button>
					<button
						class:open={menuOpen}
						class="icon-action bordered"
						type="button"
						aria-label="More game actions"
						aria-haspopup="menu"
						aria-expanded={menuOpen}
						data-tooltip="Show more game actions"
						data-tooltip-side="bottom-end"
						data-game-menu
						onclick={(event) => {
							if (menuOpen) store.closeMenu()
							else store.openMenu(event, place)
						}}>
						<MoreHorizontal size={18} />
					</button>
				</div>
			</div>

			{#if favorite}
				<div class="identity-section">
					<span class="tag-pill favorite">
						<Star size={11} fill="currentColor" aria-hidden="true" />
						<span>Favorite</span>
					</span>
				</div>
			{/if}

			{#if rootPlace}
				<p class="game-subplace-note" role="note">
					<Network size={12} aria-hidden="true" />
					<span
						>Subplace of <button
							class="game-subplace-link"
							type="button"
							onclick={() => store.select(rootPlace)}
							>{store.displayName(rootPlace)}</button
						></span>
					<span class="game-subplace-hint"
						>{" · Main game stats are hidden"}</span>
				</p>
			{/if}

			{#if store.detailsError}
				<p class="game-details-error" role="alert">
					<AlertTriangle size={14} aria-hidden="true" />
					<span>{store.detailsError}</span>
				</p>
			{/if}

			<section class="game-section" aria-labelledby="game-information">
				<h3 id="game-information">Details</h3>
				<div class="game-summary">
					<dl class="game-facts">
						<div>
							<dt><Shapes size={12} aria-hidden="true" />Genre</dt>
							<dd class="game-fact-wrap">{genre || "—"}</dd>
						</div>
						<div>
							<dt>
								<ShieldCheck size={12} aria-hidden="true" />Content
								rating
							</dt>
							<dd class="game-fact-wrap">{contentRating || "—"}</dd>
						</div>
					</dl>

					{#if !rootPlace}
						<div
							class="game-rating"
							class:empty={!votes}
							role="group"
							aria-label={game
								? votes
									? `${percent.format(likeShare)} of ${numbers.format(votes)} votes are likes`
									: "No votes"
								: "Loading votes"}>
							<span class="game-rating-count">
								<ThumbsUp size={14} aria-hidden="true" />
								{#if game}<CompactNumber
										value={game.upVotes}
										unit={game.upVotes === 1
											? "like"
											: "likes"} />{:else}—{/if}
							</span>
							<span class="game-rating-bar">
								<span class="game-rating-track" aria-hidden="true"
									><span
										style:width={`${Math.min(likeShare, 0.5) * 200}%`}
									></span
									></span>
								<span class="game-rating-percent">
									{#if game && votes}
										{@const total = `${numbers.format(votes)} ${votes === 1 ? "vote" : "votes"}`}
										<button
											class="hover-value"
											type="button"
											aria-label={total}
											data-tooltip={total}
											data-tooltip-side="top"
											>{percent.format(likeShare)}</button>
									{:else}{game ? "No votes" : "—"}{/if}
								</span>
								<span class="game-rating-track" aria-hidden="true"
									><span
										style:width={`${Math.max(likeShare - 0.5, 0) * 200}%`}
									></span
									></span>
							</span>
							<span class="game-rating-count">
								<ThumbsDown size={14} aria-hidden="true" />
								{#if game}<CompactNumber
										value={game.downVotes}
										unit={game.downVotes === 1
											? "dislike"
											: "dislikes"} />{:else}—{/if}
							</span>
						</div>
					{/if}
				</div>

				<dl class="game-facts">
					{#if !rootPlace}
						{#each activity as fact (fact.label)}
							<div>
								<dt>
									<fact.icon
										size={12}
										aria-hidden="true" />{fact.label}
								</dt>
								<dd class="game-fact-number">
									{#if fact.value !== undefined}<CompactNumber
											value={fact.value} />{:else}—{/if}
								</dd>
							</div>
						{/each}
						<div>
							<dt><History size={12} aria-hidden="true" />Updated</dt>
							<dd>
								{#if game?.updatedAtMs}<Timestamp
										value={game.updatedAtMs} />{:else}—{/if}
							</dd>
						</div>
						<div>
							<dt>
								<CalendarDays size={12} aria-hidden="true" />Created
							</dt>
							<dd>
								{#if game?.createdAtMs}<Timestamp
										value={game.createdAtMs} />{:else}—{/if}
							</dd>
						</div>
					{/if}
					<div>
						<dt><Mic size={12} aria-hidden="true" />Voice chat</dt>
						<dd>{game ? supported(game.communication?.voiceChat) : "—"}</dd>
					</div>
					<div>
						<dt><Video size={12} aria-hidden="true" />Camera</dt>
						<dd>{game ? supported(game.communication?.camera) : "—"}</dd>
					</div>
					{#each restrictions as fact (fact.label)}
						<div>
							<dt>
								<fact.icon size={12} aria-hidden="true" />{fact.label}
							</dt>
							<dd>{fact.value}</dd>
						</div>
					{/each}
					{#each versions as fact (fact.label)}
						<div>
							<dt>
								<fact.icon size={12} aria-hidden="true" />{fact.label}
							</dt>
							<dd class="game-fact-number game-fact-copyable">
								{fact.value ?? (game ? "Unavailable" : "—")}
							</dd>
						</div>
					{/each}
				</dl>
			</section>

			{#if game}
				<section class="game-section" aria-labelledby="game-description">
					<h3 id="game-description">Description</h3>
					{#if game.description.trim()}
						<pre class="text-block">{game.description.trim()}</pre>
					{:else}
						<p class="game-description-empty">No description provided.</p>
					{/if}
				</section>
			{/if}
		</div>
	</section>
{/if}

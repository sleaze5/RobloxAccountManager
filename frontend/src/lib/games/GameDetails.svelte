<script lang="ts">
	import arrowsClockwiseIcon from "@phosphor-icons/core/regular/arrows-clockwise.svg?raw"
	import calendarDotsIcon from "@phosphor-icons/core/regular/calendar-dots.svg?raw"
	import circleNotchIcon from "@phosphor-icons/core/regular/circle-notch.svg?raw"
	import clockCounterClockwiseIcon from "@phosphor-icons/core/regular/clock-counter-clockwise.svg?raw"
	import copyIcon from "@phosphor-icons/core/regular/copy.svg?raw"
	import dotsThreeIcon from "@phosphor-icons/core/regular/dots-three.svg?raw"
	import eyeIcon from "@phosphor-icons/core/regular/eye.svg?raw"
	import footprintsIcon from "@phosphor-icons/core/regular/footprints.svg?raw"
	import gitCommitIcon from "@phosphor-icons/core/regular/git-commit.svg?raw"
	import hardDrivesIcon from "@phosphor-icons/core/regular/hard-drives.svg?raw"
	import magnifyingGlassIcon from "@phosphor-icons/core/regular/magnifying-glass.svg?raw"
	import microphoneIcon from "@phosphor-icons/core/regular/microphone.svg?raw"
	import pencilIcon from "@phosphor-icons/core/regular/pencil.svg?raw"
	import personArmsSpreadIcon from "@phosphor-icons/core/regular/person-arms-spread.svg?raw"
	import pulseIcon from "@phosphor-icons/core/regular/pulse.svg?raw"
	import sealCheckIcon from "@phosphor-icons/core/regular/seal-check.svg?raw"
	import shapesIcon from "@phosphor-icons/core/regular/shapes.svg?raw"
	import shieldCheckIcon from "@phosphor-icons/core/regular/shield-check.svg?raw"
	import starFillIcon from "@phosphor-icons/core/fill/star-fill.svg?raw"
	import starIcon from "@phosphor-icons/core/regular/star.svg?raw"
	import tShirtIcon from "@phosphor-icons/core/regular/t-shirt.svg?raw"
	import thumbsUpIcon from "@phosphor-icons/core/regular/thumbs-up.svg?raw"
	import ticketIcon from "@phosphor-icons/core/regular/ticket.svg?raw"
	import treeStructureIcon from "@phosphor-icons/core/regular/tree-structure.svg?raw"
	import uploadSimpleIcon from "@phosphor-icons/core/regular/upload-simple.svg?raw"
	import usersThreeIcon from "@phosphor-icons/core/regular/users-three.svg?raw"
	import videoCameraIcon from "@phosphor-icons/core/regular/video-camera.svg?raw"
	import warningIcon from "@phosphor-icons/core/regular/warning.svg?raw"
	import Icon from "../shared/Icon.svelte"
	import { untrack } from "svelte"
	import type { WorkspaceState } from "../layout/workspace-state.svelte"
	import Timestamp from "../shared/Timestamp.svelte"
	import { relativeTime } from "../shared/timestamp"
	import { creatorByline, type GamesStore } from "./games-store.svelte"
	import GameIcon from "./GameIcon.svelte"

	let {
		store,
		workspace,
		onSearchUniverse,
	}: {
		store: GamesStore
		workspace: WorkspaceState
		onSearchUniverse: (universeId: number) => void
	} = $props()
	const numbers = new Intl.NumberFormat(),
		percent = new Intl.NumberFormat(undefined, {
			style: "percent",
			maximumFractionDigits: 1,
		}),
		graphemes = new Intl.Segmenter(undefined, { granularity: "grapheme" })

	type Fact = {
		label: string
		icon: string
		value: string | undefined
		detail?: string
	}

	const exact = (value: number | undefined) =>
			value === undefined ? undefined : numbers.format(value),
		count = (value: number, one: string, many: string) =>
			`${numbers.format(value)} ${value === 1 ? one : many}`
	const place = $derived(store.selected),
		game = $derived(store.game),
		favorite = $derived(place ? store.isFavorite(place.placeId) : false),
		nickname = $derived(place ? store.nicknameFor(place.placeId) : ""),
		// Keep the final character and star together, even when a long word wraps.
		nameSplit = $derived(
			Array.from(graphemes.segment(place?.name.trimEnd() ?? "")).at(-1)?.index ??
				0,
		),
		nameStart = $derived(place?.name.slice(0, nameSplit) ?? ""),
		nameEnd = $derived(place?.name.slice(nameSplit).trimEnd() ?? ""),
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
		details = $derived<Fact[]>([
			{ label: "Genre", icon: shapesIcon, value: genre || undefined },
			{
				label: "Content rating",
				icon: shieldCheckIcon,
				value: contentRating || undefined,
			},
			...(rootPlace
				? []
				: [
						{
							label: "Rating",
							icon: thumbsUpIcon,
							value: game
								? votes
									? `${percent.format(likeShare)} liked`
									: "No votes"
								: undefined,
							detail:
								game && votes
									? `${count(game.upVotes, "like", "likes")} - ${count(game.downVotes, "dislike", "dislikes")}`
									: "",
						},
						{ label: "Visits", icon: eyeIcon, value: exact(game?.visits) },
						{
							label: "Favorites",
							icon: starIcon,
							value: exact(game?.favorites),
						},
						{
							label: "Server capacity",
							icon: usersThreeIcon,
							value:
								game?.maxPlayers === undefined
									? undefined
									: count(game.maxPlayers, "player", "players"),
						},
					]),
		]),
		dates = $derived([
			{
				label: "Updated",
				icon: clockCounterClockwiseIcon,
				value: game?.updatedAtMs,
			},
			{ label: "Created", icon: calendarDotsIcon, value: game?.createdAtMs },
		]),
		supported = (value: boolean | undefined) =>
			value === undefined ? undefined : value ? "Supported" : "Not supported",
		versions = $derived<Fact[]>([
			{
				label: "Saved version",
				icon: gitCommitIcon,
				value: game?.versions?.saved?.toString(),
			},
			{
				label: "Published version",
				icon: uploadSimpleIcon,
				value: game?.versions?.published?.toString(),
			},
		]),
		features = $derived<Fact[]>(
			game
				? [
						{
							label: "Voice chat",
							icon: microphoneIcon,
							value: supported(game.communication?.voiceChat),
						},
						{
							label: "Camera",
							icon: videoCameraIcon,
							value: supported(game.communication?.camera),
						},
						{
							label: "Access",
							icon: ticketIcon,
							value:
								game.price > 0
									? `Paid access - R$ ${numbers.format(game.price)}`
									: "Free",
						},
						{
							label: "Avatar type",
							icon: personArmsSpreadIcon,
							value: game.avatarType || "Unknown",
						},
						{
							label: "Animations",
							icon: footprintsIcon,
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
										icon: tShirtIcon,
										value: `${numbers.format(game.avatarRules.itemOverrides)} replaced by the game`,
									},
								]
							: []),
						{
							label: "Private servers",
							icon: hardDrivesIcon,
							value: game.privateServersAllowed
								? "Allowed"
								: "Not allowed",
						},
						...(rootPlace
							? []
							: [
									{
										label: "Copying",
										icon: copyIcon,
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
	const metaClass =
			"flex min-w-0 flex-wrap items-center gap-1 font-mono text-2xs wrap-anywhere text-fg-muted",
		sectionHeadingClass = "text-xs font-semibold text-fg-muted"
</script>

{#snippet row(fact: Fact, mono = false)}
	<div>
		<dt><Icon svg={fact.icon} size={16} aria-hidden="true" />{fact.label}</dt>
		<dd>
			{#if fact.value}
				<span class={[mono && "font-mono select-text"]}>{fact.value}</span>
				{#if fact.detail}<small>{fact.detail}</small>{/if}
			{:else}
				<span class="text-fg-faint">{game ? "Unavailable" : "-"}</span>
			{/if}
		</dd>
	</div>
{/snippet}

{#if place}
	<section
		class="min-h-0 min-w-0 animate-enter overflow-y-auto"
		aria-label={store.displayName(place)}
		aria-busy={store.loadingDetails}>
		<div class="flex flex-col gap-4 p-3 pb-4">
			<div class="flex items-start gap-3 @max-xl:flex-col">
				<div class="flex min-w-0 flex-1 items-start gap-4">
					<button
						type="button"
						class="shrink-0 cursor-default rounded-md"
						aria-label="Game icon"
						aria-haspopup="menu"
						aria-expanded={workspace.imageContextMenu?.imageUrl ===
							place.iconUrl}
						disabled={!place.iconUrl}
						onkeydown={(event) => {
							if (
								event.key === "ContextMenu" ||
								(event.shiftKey && event.key === "F10")
							) {
								workspace.openImageContextMenu(event, place.iconUrl)
							}
						}}
						oncontextmenu={(event) =>
							workspace.openImageContextMenu(event, place.iconUrl)}>
						<GameIcon url={place.iconUrl} size="xl" />
					</button>
					<div class="flex min-w-0 flex-col gap-1">
						<div class="min-w-0 leading-6">
							<h2
								class="mr-1 inline text-lg font-semibold tracking-tight wrap-anywhere text-fg">
								{nameStart}<span class="whitespace-nowrap"
									>{nameEnd}{#if favorite}<Icon
											svg={starFillIcon}
											class="relative top-px ml-1 inline-block animate-pop text-accent-strong"
											size={16}
											aria-label="Favorite" />{/if}</span>
							</h2>
							{#if favorite}
								<span
									class="inline-flex max-w-full min-w-0 items-center gap-0.5 align-top text-sm text-fg-muted">
									{#if nickname || editing}<span aria-hidden="true"
											>(</span
										>{/if}
									{#if editing}
										<input
											class="h-6 w-44 min-w-0 rounded-md border border-line-strong bg-transparent px-1.5 text-sm text-fg placeholder:text-fg-faint"
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
										{#if nickname}<span class="truncate"
												>{nickname}</span
											>{/if}
										<button
											class="btn-ghost-sm"
											type="button"
											aria-label={nickname
												? "Edit nickname"
												: "Add nickname"}
											data-tooltip={nickname
												? "Edit nickname"
												: "Add nickname"}
											disabled={store.savingNickname}
											onclick={startEditing}
											><Icon
												svg={pencilIcon}
												size={14}
												aria-hidden="true" /></button>
									{/if}
									{#if nickname || editing}<span aria-hidden="true"
											>)</span
										>{/if}
								</span>
							{/if}
						</div>
						<div class={metaClass}>
							<span>ID: {place.placeId}</span>
							<span class="text-fg-faint" aria-hidden="true">-</span>
							<span>Universe ID: {place.universeId}</span>
							<button
								class="-my-1 -ms-1 btn-ghost size-5"
								type="button"
								aria-label="Search places in this universe"
								data-tooltip="Search places in this universe"
								onclick={() => onSearchUniverse(place.universeId)}
								><Icon
									svg={magnifyingGlassIcon}
									size={14}
									aria-hidden="true" /></button>
						</div>
						<div class={metaClass}>
							<span class="inline-flex items-center gap-1 text-fg-muted"
								>{creatorByline(place)}{#if place.creatorVerified}<Icon
										svg={sealCheckIcon}
										size={14}
										aria-label="Verified creator" />{/if}</span>
							{#if place.creatorId}
								<span class="text-fg-faint" aria-hidden="true">-</span>
								<span>{place.creatorType} ID: {place.creatorId}</span>
							{/if}
						</div>
						{#if game && !rootPlace}
							<p
								class={[
									"inline-flex items-center gap-1 text-xs font-medium tabular-nums",
									game.playing > 0
										? "text-presence-game"
										: "text-fg-muted",
								]}>
								<Icon svg={pulseIcon} size={14} aria-hidden="true" />
								{numbers.format(game.playing)} playing
							</p>
						{/if}
					</div>
				</div>
				<div class="relative flex items-center gap-1 @max-xl:self-end">
					<button
						class="btn-icon"
						type="button"
						aria-label={store.loadingDetails
							? "Refreshing game details"
							: "Refresh game details"}
						aria-busy={store.loadingDetails}
						data-tooltip="Refresh game details"
						disabled={store.loadingDetails}
						onclick={() => void store.refreshDetails()}>
						{#if store.loadingDetails}<Icon
								svg={circleNotchIcon}
								class="spinner"
								size={16}
								aria-hidden="true" />{:else}<Icon
								svg={arrowsClockwiseIcon}
								data-turn
								size={16}
								aria-hidden="true" />{/if}
					</button>
					<button
						class="btn-icon"
						type="button"
						aria-label="Servers"
						data-tooltip="Servers"
						data-game-servers
						onclick={() => {
							store.closeMenu()
							store.detailsPage = "servers"
						}}>
						<Icon svg={hardDrivesIcon} size={16} aria-hidden="true" />
					</button>
					<button
						class="btn-icon"
						type="button"
						aria-label="More game actions"
						aria-haspopup="menu"
						aria-expanded={menuOpen}
						data-open={menuOpen || undefined}
						data-tooltip="Show more game actions"
						data-tooltip-side="bottom-end"
						data-game-menu
						onclick={(event) => {
							if (menuOpen) store.closeMenu()
							else store.openMenu(event, place)
						}}>
						<Icon svg={dotsThreeIcon} size={16} aria-hidden="true" />
					</button>
				</div>
			</div>

			{#if favorite}
				<div class="flex flex-wrap items-center gap-1">
					<span class="tag-pill-favorite">
						<Icon svg={starFillIcon} size={12} aria-hidden="true" />
						<span>Favorite</span>
					</span>
				</div>
			{/if}

			{#if rootPlace}
				<p
					class="inline-flex max-w-full items-center gap-1 self-start rounded-md border border-line bg-raised px-2 py-0.5 text-xs text-fg"
					role="note">
					<Icon
						svg={treeStructureIcon}
						class="shrink-0 text-fg-muted"
						size={14}
						aria-hidden="true" />
					<span
						>Subplace of <button
							class="rounded-sm underline underline-offset-2 hover:text-fg focus-visible:outline-offset-2"
							type="button"
							onclick={() => store.select(rootPlace)}
							>{store.displayName(rootPlace)}</button
						></span>
					<span class="truncate text-fg-muted"
						>{" - Main game stats are hidden"}</span>
				</p>
			{/if}

			{#if store.detailsError}
				<p
					class="flex items-start gap-2 text-xs text-danger-muted"
					role="alert">
					<Icon
						svg={warningIcon}
						class="mt-0.5 shrink-0"
						size={16}
						aria-hidden="true" />
					<span>{store.detailsError}</span>
				</p>
			{/if}

			<section
				class="flex flex-col gap-3 pt-3"
				aria-labelledby="game-information">
				<h3 id="game-information" class={sectionHeadingClass}>Details</h3>
				<dl class="facts">
					{#each details as fact (fact.label)}{@render row(fact)}{/each}
					{#if !rootPlace}
						{#each dates as fact (fact.label)}
							<div>
								<dt>
									<Icon
										svg={fact.icon}
										size={16}
										aria-hidden="true" />{fact.label}
								</dt>
								<dd>
									{#if fact.value}
										<Timestamp value={fact.value} />
										<small>{relativeTime(fact.value)}</small>
									{:else}
										<span class="text-fg-faint"
											>{game ? "Unavailable" : "-"}</span>
									{/if}
								</dd>
							</div>
						{/each}
					{/if}
					{#each versions as fact (fact.label)}{@render row(
							fact,
							true,
						)}{/each}
				</dl>
			</section>

			{#if game}
				<section
					class="flex flex-col gap-3 pt-3"
					aria-labelledby="game-features">
					<h3 id="game-features" class={sectionHeadingClass}>
						Access &amp; features
					</h3>
					<dl class="facts">
						{#each features as fact (fact.label)}{@render row(fact)}{/each}
					</dl>
				</section>
			{/if}

			{#if game}
				<section
					class="flex flex-col gap-3 pt-3"
					aria-labelledby="game-description">
					<h3 id="game-description" class={sectionHeadingClass}>
						Description
					</h3>
					{#if game.description.trim()}
						<p class="text-block" data-variant="prose">
							{game.description.trim()}
						</p>
					{:else}
						<p class="text-xs text-fg-muted">No description provided.</p>
					{/if}
				</section>
			{/if}
		</div>
	</section>
{/if}

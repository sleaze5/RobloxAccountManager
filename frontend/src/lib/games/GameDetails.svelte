<script lang="ts">
	import { notifyErrors } from "../notifications/error-notifications.svelte"
	import arrowSquareOutIcon from "@phosphor-icons/core/regular/arrow-square-out.svg?raw"
	import calendarDotsIcon from "@phosphor-icons/core/regular/calendar-dots.svg?raw"
	import clockCounterClockwiseIcon from "@phosphor-icons/core/regular/clock-counter-clockwise.svg?raw"
	import copyIcon from "@phosphor-icons/core/regular/copy.svg?raw"
	import dotsThreeIcon from "@phosphor-icons/core/regular/dots-three.svg?raw"
	import footprintsIcon from "@phosphor-icons/core/regular/footprints.svg?raw"
	import gitCommitIcon from "@phosphor-icons/core/regular/git-commit.svg?raw"
	import hardDrivesIcon from "@phosphor-icons/core/regular/hard-drives.svg?raw"
	import linkIcon from "@phosphor-icons/core/regular/link.svg?raw"
	import magnifyingGlassIcon from "@phosphor-icons/core/regular/magnifying-glass.svg?raw"
	import microphoneIcon from "@phosphor-icons/core/regular/microphone.svg?raw"
	import pencilIcon from "@phosphor-icons/core/regular/pencil.svg?raw"
	import personArmsSpreadIcon from "@phosphor-icons/core/regular/person-arms-spread.svg?raw"
	import prohibitIcon from "@phosphor-icons/core/regular/prohibit.svg?raw"
	import shapesIcon from "@phosphor-icons/core/regular/shapes.svg?raw"
	import shieldCheckIcon from "@phosphor-icons/core/regular/shield-check.svg?raw"
	import starFillIcon from "@phosphor-icons/core/fill/star-fill.svg?raw"
	import tShirtIcon from "@phosphor-icons/core/regular/t-shirt.svg?raw"
	import translateIcon from "@phosphor-icons/core/regular/translate.svg?raw"
	import ticketIcon from "@phosphor-icons/core/regular/ticket.svg?raw"
	import treeStructureIcon from "@phosphor-icons/core/regular/tree-structure.svg?raw"
	import uploadSimpleIcon from "@phosphor-icons/core/regular/upload-simple.svg?raw"
	import usersThreeIcon from "@phosphor-icons/core/regular/users-three.svg?raw"
	import videoCameraIcon from "@phosphor-icons/core/regular/video-camera.svg?raw"
	import { accountBackend, GameSection } from "../backend/bridge"
	import type { GameSocialLinks, Playability } from "../backend/bridge"
	import { AccountScope } from "../accounts/account-scope"
	import type { AccountStore } from "../accounts/account-store.svelte"
	import Icon from "../shared/Icon.svelte"
	import Skeleton from "../shared/Skeleton.svelte"
	import RefreshIcon from "../shared/RefreshIcon.svelte"
	import { untrack } from "svelte"
	import ImageMenuTarget from "../layout/ImageMenuTarget.svelte"
	import Timestamp from "../shared/Timestamp.svelte"
	import { relativeTime } from "../shared/timestamp"
	import CreatorByline from "../shared/CreatorByline.svelte"
	import ProfileStats from "../shared/ProfileStats.svelte"
	import ProfileTabs from "../shared/ProfileTabs.svelte"
	import GameMedia from "./GameMedia.svelte"
	import { creatorByline, type GamesStore } from "./games-store.svelte"
	import GameIcon from "./GameIcon.svelte"
	import { gameHeadlineStats } from "./game-stats"

	let {
		store,
		accounts,
		onSearchUniverse,
	}: {
		store: GamesStore
		accounts: AccountStore
		onSearchUniverse: (universeId: number) => void
	} = $props()
	const numbers = new Intl.NumberFormat(),
		graphemes = new Intl.Segmenter(undefined, { granularity: "grapheme" })

	let playability = $state.raw<Playability | null>(null),
		socialLinks = $state.raw<GameSocialLinks | null>(null),
		socialLinksFailed = $state(false),
		detailsRefreshCount = $state(0)

	// The store replaces the selected place and the account list replaces its
	// accounts when they refresh, so the effect tracks the IDs alone and runs
	// again only when another universe or account is selected.
	const selectedUniverseId = $derived(store.selected?.universeId),
		selectedAccountId = $derived(accounts.selectedAccount.id)

	// Roblox answers these for the selected account, so they load beside the
	// details instead of with them.
	$effect(() => {
		const scope = new AccountScope(selectedAccountId, accounts),
			universeId = selectedUniverseId
		playability = socialLinks = null
		socialLinksFailed = false
		if (!universeId || !scope.isCurrent || scope.accountId <= 0) return
		// A details refresh also reads them again for the selected account.
		void detailsRefreshCount
		void loadPlayability(scope, universeId)
		void loadSocialLinks(scope, universeId)
		return () => scope.dispose()
	})

	async function loadPlayability(
		scope: AccountScope,
		universeId: number,
	): Promise<void> {
		try {
			const status = await scope.track(
				accountBackend.GetGamePlayability(scope.accountId, universeId),
			)
			if (scope.isCurrent) playability = status
		} catch {
			// The backend logs the failure. A failed check cannot say the game is unplayable.
		}
	}

	async function loadSocialLinks(
		scope: AccountScope,
		universeId: number,
	): Promise<void> {
		try {
			const links = await scope.track(
				accountBackend.GetGameSocialLinks(scope.accountId, universeId),
			)
			if (scope.isCurrent) socialLinks = links
		} catch {
			// The backend logs the failure, and the row shows that it is unavailable.
			if (scope.isCurrent) socialLinksFailed = true
		}
	}

	function refreshGameDetails(): void {
		detailsRefreshCount++
		void store.refreshDetails()
	}

	type Fact = {
		label: string
		icon: string
		value: string | undefined
		detail?: string
		isLoading?: boolean
	}

	const exact = (value: number | undefined) =>
			value === undefined ? undefined : numbers.format(value),
		count = (value: number, one: string, many: string) =>
			`${numbers.format(value)} ${value === 1 ? one : many}`
	const place = $derived(store.selected),
		game = $derived(store.game),
		// Until the details arrive, every row holds its place with a placeholder,
		// so that the pane does not move when they fill in.
		isWaitingForGame = $derived(!game && !store.detailsError),
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
		// The develop API lists an unlisted game without its counts and genre.
		listedGame = $derived(game?.isUnlisted ? undefined : game),
		playabilityWarning = $derived(
			playability && !playability.isPlayable
				? { label: playability.label, message: playability.message }
				: game?.isUnlisted
					? {
							label: "Unavailable",
							message:
								"Roblox no longer lists this game, so its player counts are unavailable.",
						}
					: null,
		),
		genre = $derived(
			listedGame &&
				([listedGame.genre, listedGame.subgenre].filter(Boolean).join("; ") ||
					"None"),
		),
		languages = $derived(game?.languages ?? null),
		// A short list fits in the row. A longer one shows its count, then the names.
		maxInlineLanguages = 3,
		contentRating = $derived(
			game &&
				!store.isGameSectionPending(GameSection.GameSectionMaturity) &&
				[game.maturity || "Unknown", ...(game.maturityDescriptors ?? [])].join(
					"; ",
				),
		),
		// An unlisted game has no counts, which its warning explains.
		headlineStats = $derived(
			gameHeadlineStats(game, isWaitingForGame, {
				isLoaded: store.loadedGameSections.includes(
					GameSection.GameSectionVotes,
				),
				isPending: store.isGameSectionPending(GameSection.GameSectionVotes),
			}),
		),
		// A subplace shows only its own description and versions. Its other
		// details belong to the universe, which its root place shows.
		universeDetails = $derived<Fact[]>([
			{ label: "Genre", icon: shapesIcon, value: genre || undefined },
			{
				label: "Content rating",
				icon: shieldCheckIcon,
				value: contentRating || undefined,
				isLoading: store.isGameSectionPending(GameSection.GameSectionMaturity),
			},
			{
				label: "Languages",
				icon: translateIcon,
				value: !languages
					? undefined
					: languages.length === 0
						? "None"
						: languages.length <= maxInlineLanguages
							? languages.join(", ")
							: count(languages.length, "language", "languages"),
				detail:
					languages && languages.length > maxInlineLanguages
						? languages.join(", ")
						: "",
				isLoading: store.isGameSectionPending(GameSection.GameSectionLanguages),
			},
			{
				label: "Server capacity",
				icon: usersThreeIcon,
				value:
					listedGame?.maxPlayers === undefined
						? undefined
						: count(listedGame.maxPlayers, "player", "players"),
			},
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
				isLoading: store.isGameSectionPending(GameSection.GameSectionVersions),
			},
			{
				label: "Published version",
				icon: uploadSimpleIcon,
				value: game?.versions?.published?.toString(),
				isLoading: store.isGameSectionPending(GameSection.GameSectionVersions),
			},
		]),
		features = $derived<Fact[]>([
			{
				label: "Voice chat",
				icon: microphoneIcon,
				value: supported(game?.communication?.voiceChat),
				isLoading: store.isGameSectionPending(
					GameSection.GameSectionCommunication,
				),
			},
			{
				label: "Camera",
				icon: videoCameraIcon,
				value: supported(game?.communication?.camera),
				isLoading: store.isGameSectionPending(
					GameSection.GameSectionCommunication,
				),
			},
			{
				label: "Access",
				icon: ticketIcon,
				value: game
					? game.price > 0
						? `Paid access: R$ ${numbers.format(game.price)}`
						: "Free"
					: undefined,
			},
			{
				label: "Avatar type",
				icon: personArmsSpreadIcon,
				value: game ? game.avatarType || "Unknown" : undefined,
			},
			{
				label: "Animations",
				icon: footprintsIcon,
				value: game?.avatarRules
					? game.avatarRules.customAnimationsAllowed
						? "Custom allowed"
						: "Roblox defaults only"
					: !game ||
						  store.isGameSectionPending(GameSection.GameSectionAvatarRules)
						? undefined
						: "Unknown",
				isLoading: store.isGameSectionPending(
					GameSection.GameSectionAvatarRules,
				),
			},
			...(game?.avatarRules?.itemOverrides
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
				value: game
					? game.privateServersAllowed
						? "Allowed"
						: "Not allowed"
					: undefined,
			},
			{
				label: "Copying",
				icon: copyIcon,
				value: game
					? game.copyingAllowed
						? "Allowed"
						: "Not allowed"
					: undefined,
			},
		])
	let draft = $state(""),
		editing = $state(false),
		descriptionElement = $state<HTMLParagraphElement>(),
		isDescriptionExpanded = $state(false),
		isDescriptionClamped = $state(false)

	// A long description shows its first lines, with a button that shows the
	// rest. Another game starts collapsed again.
	$effect(() => {
		void selectedUniverseId
		isDescriptionExpanded = false
	})

	$effect(() => {
		const element = descriptionElement
		if (!element) return
		const measure = (): void => {
			if (!isDescriptionExpanded)
				isDescriptionClamped = element.scrollHeight > element.clientHeight
		}
		const resizeObserver = new ResizeObserver(measure)
		resizeObserver.observe(element)
		measure()
		return () => resizeObserver.disconnect()
	})

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

	notifyErrors("Game details not loaded", () => store.detailsError)
</script>

{#snippet row(fact: Fact, mono = false)}
	<div>
		<dt><Icon svg={fact.icon} size={16} aria-hidden="true" />{fact.label}</dt>
		<dd>
			{#if fact.value}
				<span class={[mono && "font-mono select-text"]}>{fact.value}</span>
				{#if fact.detail}<small>{fact.detail}</small>{/if}
			{:else if fact.isLoading || isWaitingForGame}
				<Skeleton className="w-28" />
			{:else}
				<span class="text-fg-faint">Unavailable</span>
			{/if}
		</dd>
	</div>
{/snippet}

{#snippet dateRow(fact: { label: string; icon: string; value: number | undefined })}
	<div>
		<dt>
			<Icon svg={fact.icon} size={16} aria-hidden="true" />{fact.label}
		</dt>
		<dd>
			{#if fact.value}
				<Timestamp value={fact.value} />
				<small>{relativeTime(fact.value)}</small>
			{:else if isWaitingForGame}
				<Skeleton className="w-28" />
			{:else}
				<span class="text-fg-faint">Unavailable</span>
			{/if}
		</dd>
	</div>
{/snippet}

{#if place}
	<section
		class="@container min-h-0 min-w-0 animate-enter overflow-y-auto"
		aria-label={store.displayName(place)}
		aria-busy={store.loadingDetails}>
		<div class="flex flex-col gap-5 p-4 pb-5">
			<div class="flex items-start gap-4">
				<ImageMenuTarget
					imageUrl={place.iconUrl}
					label={`${store.displayName(place)} icon`}>
					<GameIcon url={place.iconUrl} size="xl" />
				</ImageMenuTarget>
				<div class="flex min-w-0 flex-1 flex-col gap-1">
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
											if (event.key === "Enter") saveNickname()
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
										class="button-ghost-sm"
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
					{#if place.creatorId && (place.creatorType === "User" || place.creatorType === "Group")}
						<CreatorByline
							kind={place.creatorType}
							creatorId={place.creatorId}
							name={place.creatorName}
							hasVerifiedBadge={place.creatorVerified} />
					{:else}
						<p class="text-xs text-fg-muted">{creatorByline(place)}</p>
					{/if}
					<div class={metaClass}>
						<span class="select-text">Place ID: {place.placeId}</span>
						<span class="text-fg-faint" aria-hidden="true">·</span>
						<span class="select-text">Universe ID: {place.universeId}</span>
						<button
							class="-my-1 -ms-1 button-ghost size-5"
							type="button"
							aria-label="Search places in this universe"
							data-tooltip="Search places in this universe"
							onclick={() => onSearchUniverse(place.universeId)}
							><Icon
								svg={magnifyingGlassIcon}
								size={14}
								aria-hidden="true" /></button>
						{#if rootPlace}
							<span class="text-fg-faint" aria-hidden="true">·</span>
							<span class="select-text"
								>Root place ID: {rootPlace.placeId}</span>
						{/if}
					</div>
					{#if game && playabilityWarning}
						<p
							class="flex animate-enter flex-wrap items-center gap-x-2 gap-y-1 text-xs text-fg-muted">
							<span class="tag-pill-warning">
								<Icon svg={prohibitIcon} size={12} aria-hidden="true" />
								<span>{playabilityWarning.label}</span>
							</span>
							<span>{playabilityWarning.message}</span>
						</p>
					{/if}
					{#if !rootPlace}
						<ProfileStats label="Game numbers" stats={headlineStats} />
					{/if}
				</div>
				<div class="relative flex shrink-0 items-center gap-1">
					<button
						class="button-icon"
						type="button"
						aria-label={store.loadingDetails
							? "Refreshing game details…"
							: "Refresh game details"}
						aria-busy={store.loadingDetails}
						data-tooltip="Refresh game details"
						disabled={store.loadingDetails}
						onclick={refreshGameDetails}>
						<RefreshIcon isRefreshing={store.loadingDetails} />
					</button>
					<button
						class="button-icon"
						type="button"
						aria-label="More actions"
						aria-haspopup="menu"
						aria-expanded={menuOpen}
						data-open={menuOpen || undefined}
						data-tooltip="More actions"
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

			<div class="-mx-4 h-8 border-b border-line-subtle">
				<ProfileTabs />
			</div>

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
				</p>
			{/if}

			{#if !rootPlace}
				<GameMedia
					universeId={place.universeId}
					gameName={store.displayName(place)}
					accountId={accounts.selectedAccount.id}
					refreshCount={detailsRefreshCount}
					onOpenLink={(url) => void store.openLink(url)} />
			{/if}

			<div
				class={[
					"grid min-w-0 items-start gap-5 @4xl:gap-6",
					!rootPlace && "@4xl:grid-cols-3",
				]}>
				<div
					class={[
						"flex min-w-0 flex-col gap-5",
						!rootPlace && "@4xl:col-span-2",
					]}>
					{#if game && game.description.trim()}
						<section
							class="flex flex-col gap-2"
							aria-labelledby="game-description">
							<h3 id="game-description" class={sectionHeadingClass}>
								Description
							</h3>
							<p
								bind:this={descriptionElement}
								class={[
									"text-block",
									!isDescriptionExpanded && "line-clamp-6",
								]}
								data-variant="prose">
								{game.description.trim()}
							</p>
							{#if isDescriptionClamped || isDescriptionExpanded}
								<button
									class="button-text self-start"
									type="button"
									aria-expanded={isDescriptionExpanded}
									onclick={() =>
										(isDescriptionExpanded =
											!isDescriptionExpanded)}>
									{isDescriptionExpanded
										? "Show less"
										: "Show full description"}
								</button>
							{/if}
						</section>
					{:else if isWaitingForGame}
						<section class="flex flex-col gap-2" aria-busy="true">
							<h3 class={sectionHeadingClass}>Description</h3>
							<p class="text-block" data-variant="prose">
								<Skeleton
									className="w-72"
									label="Loading description…" />
							</p>
						</section>
					{/if}

					<section
						class="flex flex-col gap-2"
						aria-labelledby="game-information">
						<h3 id="game-information" class={sectionHeadingClass}>
							Details
						</h3>
						<dl class="facts">
							{#if !rootPlace}
								{#each universeDetails as fact (fact.label)}{@render row(
										fact,
									)}{/each}
							{/if}
							{#if !rootPlace && accounts.selectedAccount.id > 0}
								<div>
									<dt>
										<Icon
											svg={linkIcon}
											size={16}
											aria-hidden="true" />Social links
									</dt>
									<dd>
										{#if socialLinks?.links?.length}
											{#each socialLinks.links as link (link.url)}
												<button
													class="button"
													type="button"
													aria-label={`Open ${link.title || link.type} (${link.url})`}
													data-tooltip={link.url}
													onclick={() =>
														void store.openLink(link.url)}>
													<Icon
														svg={arrowSquareOutIcon}
														size={16}
														aria-hidden="true" />
													{link.title || link.type}
												</button>
											{/each}
										{:else if socialLinks?.isHiddenByAge}
											<span class="text-fg-muted"
												>Not shared by Roblox</span>
											<small
												>Roblox shows them only to accounts that
												it knows are over 12</small>
										{:else if socialLinks}
											<span class="text-fg-muted">None</span>
										{:else if socialLinksFailed}
											<span class="text-fg-faint"
												>Unavailable</span>
										{:else}
											<Skeleton
												className="w-28"
												label="Loading social links…" />
										{/if}
									</dd>
								</div>
							{/if}
							{#if !rootPlace}
								{#each dates as fact (fact.label)}{@render dateRow(
										fact,
									)}{/each}
							{/if}
							{#each versions as fact (fact.label)}{@render row(
									fact,
									true,
								)}{/each}
						</dl>
					</section>
				</div>

				{#if !rootPlace && (game || isWaitingForGame)}
					<section
						class="flex min-w-0 flex-col gap-2"
						aria-labelledby="game-features">
						<h3 id="game-features" class={sectionHeadingClass}>
							Access and features
						</h3>
						<dl class="facts">
							{#each features as fact (fact.label)}{@render row(
									fact,
								)}{/each}
						</dl>
					</section>
				{/if}
			</div>
		</div>
	</section>
{/if}

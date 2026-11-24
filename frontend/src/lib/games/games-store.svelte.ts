import { Browser, Events } from "@wailsio/runtime"
import { copyFailureMessage, copyText } from "../shared/clipboard"
import { createContext } from "svelte"
import { accountBackend, GameSection } from "../backend/bridge"
import type {
	Game,
	GamePlace,
	GameServerPage,
	GameServerRecordPage,
	ListedPrivateServerPage,
	PrivateServerPage,
} from "../backend/bridge"
import type { AccountStore } from "../accounts/account-store.svelte"
import type { NotificationCenter } from "../notifications/notification-center.svelte"
import { detailsKeepMs, isFresh } from "../shared/freshness"
import {
	PlaceServers,
	ServerPageCache,
	type ServerSource,
} from "./game-servers-state.svelte"
import { errorMessage } from "../shared/errors"
import { matchesSearch, searchTerms } from "../shared/search"
import { UniverseBadges } from "./game-badges-state.svelte"
import { UniverseDeveloperProducts } from "./developer-products-state.svelte"
import { UniverseGamePasses } from "./game-passes-state.svelte"
import { menuPointFor } from "../shared/presence"

// Roblox serves these parts of the game details from their own endpoints, so
// they load beside the main details and fill in as they arrive.
const gameSectionFields: { section: GameSection; fields: (keyof Game)[] }[] = [
	{ section: GameSection.GameSectionVotes, fields: ["upVotes", "downVotes"] },
	{
		section: GameSection.GameSectionMaturity,
		fields: ["maturity", "maturityDescriptors"],
	},
	{ section: GameSection.GameSectionCommunication, fields: ["communication"] },
	{ section: GameSection.GameSectionAvatarRules, fields: ["avatarRules"] },
	{ section: GameSection.GameSectionVersions, fields: ["versions"] },
	{ section: GameSection.GameSectionLanguages, fields: ["languages"] },
]

const allGameSections = gameSectionFields.map(({ section }) => section)

// A subplace shows only its own versions. Its other sections belong to its
// universe, which its root place shows.
const subplaceGameSections = [GameSection.GameSectionVersions]

function gameSectionsFor(game: Game | null): GameSection[] {
	return game?.rootPlace ? subplaceGameSections : allGameSections
}

function pickSectionFields(game: Game, sections: GameSection[]): Partial<Game> {
	return Object.fromEntries(
		gameSectionFields
			.filter(({ section }) => sections.includes(section))
			.flatMap(({ fields }) => fields.map((field) => [field, game[field]])),
	)
}

export interface GameMenu {
	place: GamePlace
	x: number
	y: number
}

function creatorLabel(place: GamePlace): string {
	if (!place.creatorName) return ""
	return place.creatorType === "User" ? `@${place.creatorName}` : place.creatorName
}

export function creatorByline(place: GamePlace): string {
	const creator = creatorLabel(place)
	return creator ? `by ${creator}` : "Unknown creator"
}

export const [getGamesStore, setGamesStore] = createContext<GamesStore>()

// StoreView names the list that the Game passes page shows.
export type StoreView = "gamePasses" | "developerProducts"

export class GamesStore {
	sidebarCollapsed = $state(false)
	sidebarWidth = $state(220)
	mode = $state<"search" | "favorites">("favorites")
	query = $state("")
	favoritesQuery = $state("")
	searchedText = $state("")
	results = $state<GamePlace[]>([])
	favorites = $state<GamePlace[]>([])
	selected = $state<GamePlace | null>(null)
	game = $state<Game | null>(null)
	detailsPage = $state<"details" | "servers" | "badges" | "gamePasses">("details")
	// The game pass opened from another page.
	highlightedGamePassId = $state<number | null>(null)
	storeView = $state<StoreView>("gamePasses")
	serverSource = $state<ServerSource>("roblox")
	menu = $state<GameMenu | null>(null)
	searching = $state(false)
	loadMoreFailed = $state(false)
	loadingDetails = $state(false)
	loadingFavorites = $state(false)
	favoritesLoaded = $state(false)
	savingFavorite = $state(false)
	savingNickname = $state(false)
	nicknameRequest = $state<number | null>(null)
	nextPageToken = $state("")
	error = $state("")
	searchError = $state("")
	detailsError = $state("")
	#sessionId = ""
	#epoch = 0
	#pendingMetadata = new Map<number, GamePlace>()
	#removeMetadataListener: (() => void) | null = null
	#searchRequest: ReturnType<typeof accountBackend.SearchGames> | null = null
	loadedGameSections = $state<GameSection[]>([])
	loadingGameSections = $state<GameSection[]>([])
	#detailsRequest: ReturnType<typeof accountBackend.GetGame> | null = null
	#sectionLoadCount = 0
	#pendingSectionFields: Partial<Game> = {}
	readonly #sectionRequests = new Set<
		ReturnType<typeof accountBackend.GetGameSection>
	>()
	readonly #details = new Map<
		number,
		{ game: Game; loadedSections: GameSection[]; fetchedAt: number }
	>()
	readonly serverPages = new ServerPageCache<GameServerPage>()
	readonly recordPages = new ServerPageCache<GameServerRecordPage>()
	readonly privatePages = new ServerPageCache<PrivateServerPage>()
	readonly listedPages = new ServerPageCache<ListedPrivateServerPage>()
	#placeServers: PlaceServers | null = null
	#universeBadges: UniverseBadges | null = null
	#universeGamePasses: UniverseGamePasses | null = null
	#universeDeveloperProducts: UniverseDeveloperProducts | null = null
	#favoritesRequest: ReturnType<typeof accountBackend.ListFavoritePlaces> | null =
		null
	#saveRequest:
		| ReturnType<typeof accountBackend.AddFavoritePlace>
		| ReturnType<typeof accountBackend.RemoveFavoritePlace>
		| ReturnType<typeof accountBackend.SetFavoritePlaceNickname>
		| null = null
	#nicknames = $derived(
		new Map(
			this.favorites
				.filter((place) => place.nickname)
				.map((place) => [place.placeId, place.nickname]),
		),
	)

	constructor(private readonly notifications: NotificationCenter) {}

	get places(): GamePlace[] {
		if (this.mode !== "favorites") return this.results
		const terms = searchTerms(this.favoritesQuery)
		if (terms.length === 0) return this.favorites
		return this.favorites.filter((place) =>
			matchesSearch(
				terms,
				`${place.nickname} ${place.name} ${place.creatorName}`,
			),
		)
	}

	nicknameFor(placeId: number): string {
		return this.#nicknames.get(placeId) ?? ""
	}

	displayName(place: GamePlace): string {
		return this.nicknameFor(place.placeId) || place.name
	}

	get canReorder(): boolean {
		return (
			this.mode === "favorites" &&
			this.favoritesLoaded &&
			this.favorites.length > 1 &&
			!this.favoritesQuery.trim() &&
			!this.savingFavorite &&
			!this.savingNickname
		)
	}

	isFavorite(placeId: number): boolean {
		return this.favorites.some((place) => place.placeId === placeId)
	}

	initialize(): void {
		if (!this.#removeMetadataListener)
			this.#removeMetadataListener = Events.On("game:metadata-changed", (event) =>
				this.updateMetadata(event.data as GamePlace),
			)
		if (!this.favoritesLoaded) void this.loadFavorites()
	}

	show(mode: "search" | "favorites"): void {
		if (this.mode === mode) return
		this.mode = mode
		this.error = ""
	}

	openMenu(event: MouseEvent | KeyboardEvent, place: GamePlace): void {
		event.preventDefault()
		event.stopPropagation()
		this.menu = { place, ...menuPointFor(event, 2) }
	}

	closeMenu(): void {
		this.menu = null
	}

	async copy(text: string | number, label: string): Promise<void> {
		this.closeMenu()
		this.error = ""
		if (
			!(await copyText(
				this.notifications,
				String(text),
				`${label} copied to clipboard.`,
			))
		) {
			this.error = copyFailureMessage(label)
		}
	}

	async openLink(url: string): Promise<void> {
		this.closeMenu()
		this.error = ""
		try {
			await Browser.OpenURL(url)
		} catch (error) {
			this.error = errorMessage(error, "Could not open the link. Try again.")
		}
	}

	clearSearch(): void {
		void this.#searchRequest?.cancel()
		this.#searchRequest = null
		this.searching = this.loadMoreFailed = false
		this.results = []
		this.query = this.searchedText = this.nextPageToken = this.#sessionId = ""
		this.error = this.searchError = ""
		this.closeMenu()
		this.select(null)
	}

	async search(more = false): Promise<void> {
		const text = more ? this.searchedText : this.query.trim()
		if (!text || (more && (this.searching || !this.nextPageToken))) return
		void this.#searchRequest?.cancel()
		this.mode = "search"
		this.loadMoreFailed = false
		if (!more) {
			this.results = []
			this.nextPageToken = ""
			this.#sessionId = ""
			this.searchedText = text
			this.select(null)
		}
		this.searching = true
		this.error = ""
		this.searchError = ""
		const request = accountBackend.SearchGames({
			text,
			sessionId: this.#sessionId,
			pageToken: this.nextPageToken,
		})
		this.#searchRequest = request
		try {
			const page = await request
			if (this.#searchRequest !== request) return
			const seen = new Set(this.results.map((place) => place.placeId))
			this.results = [
				...this.results,
				...(page.places ?? []).filter((place) => !seen.has(place.placeId)),
			]
			this.nextPageToken = page.nextPageToken
			this.#sessionId = page.sessionId
			this.updateMetadataList(page.places)
		} catch (error) {
			if (this.#searchRequest === request) {
				const reason = errorMessage(
					error,
					"Could not load the games. Try again.",
				)
				if (more) {
					this.error = reason
					this.loadMoreFailed = true
				} else this.searchError = reason
			}
		} finally {
			if (this.#searchRequest === request) {
				this.#searchRequest = null
				this.searching = false
			}
		}
	}

	searchFor(text: string): Promise<void> {
		this.query = text
		this.sidebarCollapsed = false
		return this.search()
	}

	async loadFavorites(): Promise<void> {
		if (this.loadingFavorites || this.savingFavorite || this.savingNickname) return
		this.loadingFavorites = true
		this.error = ""
		const request = accountBackend.ListFavoritePlaces()
		this.#favoritesRequest = request
		try {
			const places = await request
			if (this.#favoritesRequest !== request) return
			this.favorites = places ?? []
			for (const place of this.#pendingMetadata.values())
				this.updateMetadata(place)
			this.#pendingMetadata.clear()
			this.favoritesLoaded = true
		} catch (error) {
			if (this.#favoritesRequest === request) {
				this.error = errorMessage(
					error,
					"Could not load the favorites. Try again.",
				)
			}
		} finally {
			if (this.#favoritesRequest === request) {
				this.#favoritesRequest = null
				this.loadingFavorites = false
			}
		}
	}

	// serversFor returns the server lists of the place, and keeps them until
	// another place is selected.
	serversFor(place: GamePlace, accountId: number): PlaceServers {
		if (this.#placeServers?.placeId !== place.placeId) {
			this.#placeServers?.dispose()
			this.#placeServers = new PlaceServers(
				place.placeId,
				place.universeId,
				accountId,
				this,
			)
		}
		return this.#placeServers
	}

	#forgetPlaceServers(): void {
		this.#placeServers?.dispose()
		this.#placeServers = null
	}

	// badgesFor returns the badges of the universe, and keeps them until another
	// place is selected.
	badgesFor(universeId: number): UniverseBadges {
		if (this.#universeBadges?.universeId !== universeId) {
			this.#universeBadges?.dispose()
			this.#universeBadges = new UniverseBadges(universeId)
		}
		return this.#universeBadges
	}

	#forgetUniverseBadges(): void {
		this.#universeBadges?.dispose()
		this.#universeBadges = null
	}

	// gamePassesFor returns the universe's game passes, kept until another place
	// is selected.
	gamePassesFor(universeId: number, accounts: AccountStore): UniverseGamePasses {
		if (this.#universeGamePasses?.universeId !== universeId) {
			this.#universeGamePasses?.dispose()
			this.#universeGamePasses = new UniverseGamePasses(universeId, accounts)
		}
		return this.#universeGamePasses
	}

	#forgetUniverseGamePasses(): void {
		this.#universeGamePasses?.dispose()
		this.#universeGamePasses = null
	}

	// developerProductsFor returns the universe's developer products, kept until
	// another place is selected.
	developerProductsFor(universeId: number): UniverseDeveloperProducts {
		if (this.#universeDeveloperProducts?.universeId !== universeId) {
			this.#universeDeveloperProducts?.dispose()
			this.#universeDeveloperProducts = new UniverseDeveloperProducts(universeId)
		}
		return this.#universeDeveloperProducts
	}

	#forgetUniverseDeveloperProducts(): void {
		this.#universeDeveloperProducts?.dispose()
		this.#universeDeveloperProducts = null
	}

	select(place: GamePlace | null): void {
		if (place?.placeId !== this.selected?.placeId) {
			this.#forgetPlaceServers()
			this.#forgetUniverseBadges()
			this.#forgetUniverseGamePasses()
			this.#forgetUniverseDeveloperProducts()
		}
		void this.#detailsRequest?.cancel()
		this.#detailsRequest = null
		this.#cancelGameSections()
		this.loadedGameSections = []
		this.selected = place
		this.detailsPage = "details"
		this.highlightedGamePassId = null
		this.game = null
		this.detailsError = ""
		this.loadingDetails = false
		if (!place) return
		const cached = this.#details.get(place.placeId)
		if (cached && Date.now() - cached.fetchedAt < detailsKeepMs) {
			this.game = cached.game
			this.loadedGameSections = [...cached.loadedSections]
			this.selected = cached.game.place
		}
		this.revalidate()
	}

	// showGamePass opens the place's Game passes page with the pass highlighted.
	showGamePass(place: GamePlace, gamePassId: number): void {
		this.closeMenu()
		this.select(place)
		this.detailsPage = "gamePasses"
		this.storeView = "gamePasses"
		this.highlightedGamePassId = gamePassId
	}

	revalidate(): void {
		const place = this.selected,
			cached = place && this.#details.get(place.placeId)
		if (place && !(cached && isFresh(cached.fetchedAt))) void this.refreshDetails()
	}

	isGameSectionPending(section: GameSection): boolean {
		return (
			!this.loadedGameSections.includes(section) &&
			this.loadingGameSections.includes(section)
		)
	}

	async refreshDetails(): Promise<void> {
		const place = this.selected
		if (!place || this.loadingDetails) return
		this.loadingDetails = true
		this.detailsError = ""
		const request = accountBackend.GetGame(place.universeId, place.placeId)
		this.#detailsRequest = request
		// The sections load beside the details, so a place that was opened
		// before uses what it was then, and a new subplace learns it later.
		const knownGame = this.game?.place.placeId === place.placeId ? this.game : null,
			sectionLoad = place.universeId
				? this.#loadGameSections(
						place.universeId,
						place.placeId,
						gameSectionsFor(knownGame),
					)
				: null
		try {
			const game = await request
			if (this.#detailsRequest !== request) return
			const previousGame =
					this.game?.place.placeId === game.place.placeId ? this.game : null,
				merged = {
					...game,
					...(previousGame
						? pickSectionFields(previousGame, this.loadedGameSections)
						: {}),
					...this.#pendingSectionFields,
				}
			this.#pendingSectionFields = {}
			this.game = merged
			this.#remember(merged)
			this.selected = {
				...game.place,
				nickname: this.nicknameFor(game.place.placeId),
			}
			this.updateMetadata(game.place)
			if (!sectionLoad) {
				void this.#loadGameSections(
					game.place.universeId,
					game.place.placeId,
					gameSectionsFor(game),
				)
			}
		} catch (error) {
			if (this.#detailsRequest === request) {
				this.detailsError = errorMessage(
					error,
					"Could not load the game details. Try again.",
				)
				this.#cancelGameSections()
			}
		} finally {
			if (this.#detailsRequest === request) {
				this.#detailsRequest = null
				this.loadingDetails = false
			}
		}
	}

	// A failed section keeps its last values. The details show it as unavailable
	// when it never loaded, and the backend logs why.
	async #loadGameSections(
		universeId: number,
		placeId: number,
		sections: GameSection[],
	): Promise<void> {
		this.#cancelGameSections()
		const loadCount = this.#sectionLoadCount,
			isCurrentLoad = () =>
				loadCount === this.#sectionLoadCount &&
				this.selected?.placeId === placeId
		this.loadingGameSections = [...sections]
		await Promise.allSettled(
			sections.map(async (section) => {
				const request = accountBackend.GetGameSection(
					universeId,
					placeId,
					section,
				)
				this.#sectionRequests.add(request)
				try {
					const sectionGame = await request
					if (!isCurrentLoad()) return
					const sectionFields = pickSectionFields(sectionGame, [section])
					if (!this.loadedGameSections.includes(section)) {
						this.loadedGameSections.push(section)
					}
					if (this.game?.place.placeId === placeId) {
						this.game = { ...this.game, ...sectionFields }
						this.#remember(this.game)
					} else {
						this.#pendingSectionFields = {
							...this.#pendingSectionFields,
							...sectionFields,
						}
					}
				} catch {
					// The backend logs the failure, and the details show the part as unavailable.
				} finally {
					this.#sectionRequests.delete(request)
					if (isCurrentLoad()) {
						this.loadingGameSections = this.loadingGameSections.filter(
							(loadingSection) => loadingSection !== section,
						)
					}
				}
			}),
		)
	}

	#cancelGameSections(): void {
		this.#sectionLoadCount++
		for (const request of this.#sectionRequests) void request.cancel()
		this.#sectionRequests.clear()
		this.#pendingSectionFields = {}
		this.loadingGameSections = []
	}

	updateMetadataList(places: GamePlace[] | null): void {
		for (const place of places ?? []) this.updateMetadata(place)
	}

	updateMetadata(place: GamePlace): void {
		if (this.loadingFavorites) this.#pendingMetadata.set(place.placeId, place)
		const replace = (item: GamePlace): GamePlace =>
			item.placeId === place.placeId
				? { ...place, nickname: item.nickname }
				: item
		const cached = this.#details.get(place.placeId)
		if (cached) cached.game = { ...cached.game, place: replace(cached.game.place) }
		this.results = this.results.map(replace)
		this.favorites = this.favorites.map(replace)
		if (this.selected) this.selected = replace(this.selected)
		if (this.game?.place.placeId === place.placeId)
			this.game = { ...this.game, place: replace(this.game.place) }
		if (this.menu) this.menu = { ...this.menu, place: replace(this.menu.place) }
	}

	#remember(game: Game): void {
		const now = Date.now()
		for (const [placeId, entry] of this.#details) {
			if (now - entry.fetchedAt >= detailsKeepMs) this.#details.delete(placeId)
		}
		this.#details.set(game.place.placeId, {
			game,
			loadedSections: [...this.loadedGameSections],
			fetchedAt: now,
		})
	}

	async toggleFavorite(place: GamePlace): Promise<void> {
		const epoch = this.#epoch
		this.closeMenu()
		if (this.savingFavorite || this.savingNickname || !this.favoritesLoaded) return
		const removing = this.isFavorite(place.placeId)
		this.savingFavorite = true
		this.error = ""
		try {
			if (removing) {
				this.#saveRequest = accountBackend.RemoveFavoritePlace(place.placeId)
				await this.#saveRequest
				if (this.#epoch !== epoch) return
				this.favorites = this.favorites.filter(
					(favorite) => favorite.placeId !== place.placeId,
				)
			} else {
				const request = accountBackend.AddFavoritePlace(place.placeId)
				this.#saveRequest = request
				const saved = await request
				if (this.#epoch !== epoch) return
				this.favorites = [
					saved,
					...this.favorites.filter(
						(favorite) => favorite.placeId !== saved.placeId,
					),
				]
			}
		} catch (error) {
			if (this.#epoch === epoch) {
				this.error = errorMessage(
					error,
					"Could not update the favorites. Try again.",
				)
			}
		} finally {
			if (this.#epoch === epoch) {
				this.#saveRequest = null
				this.savingFavorite = false
			}
		}
	}

	async moveFavorite(
		placeId: number,
		targetId: number,
		after: boolean,
	): Promise<void> {
		const epoch = this.#epoch,
			previous = this.favorites,
			moved = previous.find((place) => place.placeId === placeId),
			next = previous.filter((place) => place.placeId !== placeId),
			targetIndex = next.findIndex((place) => place.placeId === targetId)
		if (!this.canReorder || !moved || targetIndex < 0) return
		next.splice(targetIndex + (after ? 1 : 0), 0, moved)
		if (next.every((place, index) => place.placeId === previous[index]?.placeId)) {
			return
		}
		this.closeMenu()
		this.favorites = next
		this.savingFavorite = true
		this.error = ""
		try {
			const request = accountBackend.ReorderFavoritePlaces(
				next.map((place) => place.placeId),
			)
			this.#saveRequest = request
			await request
		} catch (error) {
			if (this.#epoch === epoch) {
				this.favorites = previous
				this.error = errorMessage(
					error,
					"Could not reorder the favorites. Try again.",
				)
			}
		} finally {
			if (this.#epoch === epoch) {
				this.#saveRequest = null
				this.savingFavorite = false
			}
		}
	}

	editNickname(place: GamePlace): void {
		this.closeMenu()
		if (this.selected?.placeId !== place.placeId) this.select(place)
		this.nicknameRequest = place.placeId
	}

	async saveNickname(place: GamePlace, nickname: string): Promise<boolean> {
		const epoch = this.#epoch,
			current = this.favorites.find(
				(favorite) => favorite.placeId === place.placeId,
			)
		if (!current || this.savingFavorite || this.savingNickname) return false
		if (current.nickname === nickname.trim()) return true
		this.savingNickname = true
		this.error = ""
		try {
			const request = accountBackend.SetFavoritePlaceNickname(
				current.placeId,
				nickname,
			)
			this.#saveRequest = request
			const saved = await request
			if (this.#epoch !== epoch) return false
			this.favorites = this.favorites.map((favorite) =>
				favorite.placeId === current.placeId
					? { ...favorite, nickname: saved }
					: favorite,
			)
			return true
		} catch (error) {
			if (this.#epoch === epoch) {
				this.error = errorMessage(
					error,
					"Could not save the nickname. Try again.",
				)
			}
			return false
		} finally {
			if (this.#epoch === epoch) {
				this.#saveRequest = null
				this.savingNickname = false
			}
		}
	}

	reset(): void {
		this.#removeMetadataListener?.()
		this.#removeMetadataListener = null
		this.#epoch++
		this.#pendingMetadata.clear()
		void this.#searchRequest?.cancel()
		void this.#detailsRequest?.cancel()
		this.#cancelGameSections()
		this.loadedGameSections = []
		void this.#favoritesRequest?.cancel()
		void this.#saveRequest?.cancel()
		this.#searchRequest = null
		this.#detailsRequest = null
		this.#details.clear()
		this.#forgetPlaceServers()
		this.#forgetUniverseBadges()
		this.#forgetUniverseGamePasses()
		this.#forgetUniverseDeveloperProducts()
		this.serverPages.clear()
		this.recordPages.clear()
		this.privatePages.clear()
		this.listedPages.clear()
		this.#favoritesRequest = null
		this.#saveRequest = null
		this.searching =
			this.loadMoreFailed =
			this.loadingDetails =
			this.loadingFavorites =
			this.savingFavorite =
			this.savingNickname =
				false
		this.favoritesLoaded = false
		this.results = []
		this.favorites = []
		this.selected = this.game = this.menu = this.nicknameRequest = null
		this.query = this.searchedText = this.nextPageToken = this.#sessionId = ""
		this.favoritesQuery = ""
		this.mode = "favorites"
		this.detailsPage = "details"
		this.highlightedGamePassId = null
		this.storeView = "gamePasses"
		this.serverSource = "roblox"
		this.error = this.searchError = this.detailsError = ""
	}
}

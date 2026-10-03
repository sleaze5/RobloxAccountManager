import { createContext } from "svelte"
import { accountBackend } from "../backend/bridge"
import type {
	Game,
	GamePlace,
	GameServerPage,
	GameServerRecordPage,
} from "../backend/bridge"
import type { NotificationCenter } from "../notifications/notification-center.svelte"
import { detailsKeepMs, isFresh } from "../shared/freshness"
import { ServerPageCache, type ServerSource } from "./game-servers-state.svelte"

export interface GameMenu {
	place: GamePlace
	x: number
	y: number
}

// Pages outside Games read favorite nicknames through this context.
/** Returns the creator name with an @ for users, matching how Roblox shows usernames. */
export function creatorLabel(place: GamePlace): string {
	if (!place.creatorName) return ""
	return place.creatorType === "User" ? `@${place.creatorName}` : place.creatorName
}

/** Returns the "by creator" line shown under game names. */
export function creatorByline(place: GamePlace): string {
	const creator = creatorLabel(place)
	return creator ? `by ${creator}` : "Unknown creator"
}

export const [getGamesStore, setGamesStore] = createContext<GamesStore>()

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
	// The view shown for the selected place.
	detailsPage = $state<"details" | "servers">("details")
	// The server list shown by the server browser, kept while switching places.
	serverSource = $state<ServerSource>("roblox")
	menu = $state<GameMenu | null>(null)
	searching = $state(false)
	loadMoreFailed = $state(false)
	loadingDetails = $state(false)
	loadingFavorites = $state(false)
	favoritesLoaded = $state(false)
	savingFavorite = $state(false)
	savingNickname = $state(false)
	// The favorite place whose nickname field should take focus once shown.
	nicknameRequest = $state<number | null>(null)
	nextPageToken = $state("")
	error = $state("")
	searchError = $state("")
	detailsError = $state("")
	#sessionId = ""
	#epoch = 0
	#searchRequest: ReturnType<typeof accountBackend.SearchGames> | null = null
	#detailsRequest: ReturnType<typeof accountBackend.GetGame> | null = null
	// Loaded details by place ID, kept so revisited games show at once.
	readonly #details = new Map<number, { game: Game; fetchedAt: number }>()
	readonly serverPages = new ServerPageCache<GameServerPage>()
	readonly recordPages = new ServerPageCache<GameServerRecordPage>()
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
		const text = this.favoritesQuery.trim().toLowerCase()
		if (!text) return this.favorites
		return this.favorites.filter(
			(place) =>
				place.nickname.toLowerCase().includes(text) ||
				place.name.toLowerCase().includes(text) ||
				place.creatorName.toLowerCase().includes(text),
		)
	}

	// Favorites are saved by root place, so a universe has at most one nickname.
	nicknameFor(placeId: number): string {
		return this.#nicknames.get(placeId) ?? ""
	}

	displayName(place: GamePlace): string {
		return this.nicknameFor(place.placeId) || place.name
	}

	// Favorites can be reordered only while every favorite is listed.
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
		if (!this.favoritesLoaded) void this.loadFavorites()
	}

	clearError(): void {
		this.error = ""
	}

	show(mode: "search" | "favorites"): void {
		if (this.mode === mode) return
		this.mode = mode
		this.error = ""
	}

	openMenu(event: MouseEvent | KeyboardEvent, place: GamePlace): void {
		event.preventDefault()
		event.stopPropagation()
		const bounds = (event.currentTarget as HTMLElement).getBoundingClientRect(),
			atPointer = event instanceof MouseEvent && event.type === "contextmenu"
		this.menu = {
			place,
			x: atPointer ? event.clientX : bounds.left,
			y: atPointer ? event.clientY : bounds.bottom + 2,
		}
	}

	closeMenu(): void {
		this.menu = null
	}

	async copy(value: string | number, label: string): Promise<void> {
		this.closeMenu()
		try {
			await navigator.clipboard.writeText(String(value))
			this.notifications.show({
				id: "clipboard-copy",
				title: "Copied",
				message: `${label} copied to clipboard.`,
			})
		} catch {
			this.error = `${label} could not be copied. Try again.`
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
		} catch (error) {
			if (this.#searchRequest === request) {
				const reason = message(error, "Games could not be loaded. Try again.")
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
			this.favoritesLoaded = true
		} catch (error) {
			if (this.#favoritesRequest === request) {
				this.error = message(error, "Favorites could not be loaded. Try again.")
			}
		} finally {
			if (this.#favoritesRequest === request) {
				this.#favoritesRequest = null
				this.loadingFavorites = false
			}
		}
	}

	select(place: GamePlace | null): void {
		void this.#detailsRequest?.cancel()
		this.#detailsRequest = null
		this.selected = place
		this.detailsPage = "details"
		this.game = null
		this.detailsError = ""
		this.loadingDetails = false
		if (!place) return
		const cached = this.#details.get(place.placeId)
		if (cached && Date.now() - cached.fetchedAt < detailsKeepMs) {
			this.game = cached.game
		}
		this.revalidate()
	}

	// revalidate loads the selected game again once its details are no longer fresh.
	revalidate(): void {
		const place = this.selected,
			cached = place && this.#details.get(place.placeId)
		if (place && !(cached && isFresh(cached.fetchedAt))) void this.refreshDetails()
	}

	async refreshDetails(): Promise<void> {
		const place = this.selected
		if (!place || this.loadingDetails) return
		this.loadingDetails = true
		this.detailsError = ""
		const request = accountBackend.GetGame(place.universeId, place.placeId)
		this.#detailsRequest = request
		try {
			const game = await request
			if (this.#detailsRequest !== request) return
			this.#remember(game)
			this.game = game
			this.selected = game.place
		} catch (error) {
			if (this.#detailsRequest === request) {
				this.detailsError = message(
					error,
					"Game details are unavailable. Try again.",
				)
			}
		} finally {
			if (this.#detailsRequest === request) {
				this.#detailsRequest = null
				this.loadingDetails = false
			}
		}
	}

	#remember(game: Game): void {
		const now = Date.now()
		for (const [placeId, entry] of this.#details) {
			if (now - entry.fetchedAt >= detailsKeepMs) this.#details.delete(placeId)
		}
		this.#details.set(game.place.placeId, { game, fetchedAt: now })
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
					(item) => item.placeId !== place.placeId,
				)
			} else {
				const request = accountBackend.AddFavoritePlace(place.placeId)
				this.#saveRequest = request
				const saved = await request
				if (this.#epoch !== epoch) return
				this.favorites = [
					saved,
					...this.favorites.filter((item) => item.placeId !== saved.placeId),
				]
			}
		} catch (error) {
			if (this.#epoch === epoch) {
				this.error = message(
					error,
					"Favorites could not be updated. Try again.",
				)
			}
		} finally {
			if (this.#epoch === epoch) {
				this.#saveRequest = null
				this.savingFavorite = false
			}
		}
	}

	// moveFavorite places a favorite next to another one and saves the new order.
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
				this.error = message(
					error,
					"Favorites could not be reordered. Try again.",
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

	// saveNickname resolves false when the nickname could not be saved.
	async saveNickname(place: GamePlace, nickname: string): Promise<boolean> {
		const epoch = this.#epoch,
			current = this.favorites.find((item) => item.placeId === place.placeId)
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
			this.favorites = this.favorites.map((item) =>
				item.placeId === current.placeId ? { ...item, nickname: saved } : item,
			)
			return true
		} catch (error) {
			if (this.#epoch === epoch) {
				this.error = message(
					error,
					"The nickname could not be saved. Try again.",
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
		this.#epoch++
		void this.#searchRequest?.cancel()
		void this.#detailsRequest?.cancel()
		void this.#favoritesRequest?.cancel()
		void this.#saveRequest?.cancel()
		this.#searchRequest = null
		this.#detailsRequest = null
		this.#details.clear()
		this.serverPages.clear()
		this.recordPages.clear()
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
		this.serverSource = "roblox"
		this.error = this.searchError = this.detailsError = ""
	}
}

function message(error: unknown, fallback: string): string {
	return error instanceof Error && error.message.trim() ? error.message : fallback
}

import { Browser } from "@wailsio/runtime"
import { accountBackend, SocialList } from "../backend/bridge"
import type {
	SocialListPage,
	SocialProfileSnapshot,
	SocialRelationship,
	SocialUser,
} from "../backend/bridge"
import type { NotificationCenter } from "../notifications/notification-center.svelte"
import { errorMessage } from "../shared/errors"
import { detailsKeepMs, isFresh } from "../shared/freshness"

export interface SocialSubject {
	userId: number
	username: string
	displayName: string
	nickname: string
	imageUrl: string
}

export interface SocialMenu {
	person: SocialUser
	x: number
	y: number
}

export const playersView = "players" as const

export type SocialListKind = Exclude<SocialList, SocialList.$zero>

export type SocialView = SocialListKind | typeof playersView

export function personName(person: SocialSubject): string {
	return (
		person.nickname ||
		person.displayName ||
		person.username ||
		`User ${person.userId}`
	)
}

export class SocialStore {
	sidebarCollapsed = $state(false)
	sidebarWidth = $state(220)
	list = $state<SocialView>(SocialList.SocialFriends)
	query = $state("")
	playerQuery = $state("")
	accountId = $state(0)
	selfUserId = $state(0)
	trail = $state<SocialSubject[]>([])
	page = $state<SocialListPage | null>(null)
	loading = $state(false)
	loadingMore = $state(false)
	loadError = $state("")
	requestCount = $state<number | null>(null)
	selectedPerson = $state<SocialUser | null>(null)
	profile = $state<SocialProfileSnapshot | null>(null)
	loadingProfile = $state(false)
	profileError = $state("")
	responding = $state<{ userId: number; accept: boolean } | null>(null)
	unfriending = $state(false)
	unfriendError = $state("")
	updating = $state<"follow" | "nickname" | null>(null)
	nicknameError = $state("")
	menu = $state<SocialMenu | null>(null)
	error = $state("")
	#epoch = 0
	#listRequest: ReturnType<typeof accountBackend.GetSocialPage> | null = null
	#profileRequest: ReturnType<typeof accountBackend.GetSocialProfile> | null = null
	readonly #pages = new Map<string, { page: SocialListPage; fetchedAt: number }>()
	readonly #profiles = new Map<
		number,
		{ profile: SocialProfileSnapshot; fetchedAt: number }
	>()

	constructor(private readonly notifications: NotificationCenter) {}

	get subject(): SocialSubject | null {
		return this.trail.at(-1) ?? null
	}

	get subjectUserId(): number {
		return this.subject?.userId ?? this.selfUserId
	}

	get ownFriends(): boolean {
		return !this.subject && this.list === SocialList.SocialFriends
	}

	get people(): SocialUser[] {
		const people = this.page?.people ?? [],
			text = this.query.trim().toLowerCase().replace(/^@/, "")
		if (!text || this.list === playersView) return people
		return people.filter(
			(person) =>
				person.displayName.toLowerCase().includes(text) ||
				person.username.toLowerCase().includes(text),
		)
	}

	get selectedUserId(): number | null {
		return this.selectedPerson?.userId ?? null
	}

	// The current list has the freshest copy. The stored copy keeps the profile
	// open when the list changes, such as after an account switch.
	get selected(): SocialUser | null {
		return (
			(this.page?.people ?? []).find(
				(person) => person.userId === this.selectedUserId,
			) ?? this.selectedPerson
		)
	}

	clearError(): void {
		this.error = ""
	}

	show(accountId: number, selfUserId: number): void {
		if (accountId !== this.accountId) {
			const person = this.selectedPerson
			this.#switchAccount(accountId, selfUserId)
			this.#open()
			if (person) this.select({ ...person, nickname: "", request: null })
			return
		}
		const cached = this.#pages.get(this.#key())
		if (!this.#listRequest && (!cached || !isFresh(cached.fetchedAt))) {
			void this.load()
		}
	}

	home(accountId: number, selfUserId: number, list: SocialListKind): void {
		if (accountId !== this.accountId) this.#switchAccount(accountId, selfUserId)
		this.trail = []
		this.list = list
		this.#open()
	}

	showList(view: SocialView): void {
		if (this.list === view) return
		if (view === SocialList.SocialRequests || view === playersView) this.trail = []
		this.list = view
		this.#open()
	}

	searchPlayers(text: string): void {
		this.trail = []
		this.list = playersView
		this.query = this.playerQuery = text.trim()
		this.#open()
	}

	explore(person: SocialSubject, list: SocialListKind): void {
		const index = this.trail.findIndex((item) => item.userId === person.userId)
		if (person.userId === this.selfUserId) this.trail = []
		else if (index >= 0) this.trail = this.trail.slice(0, index + 1)
		else {
			const { userId, username, displayName, nickname, imageUrl } = person
			this.trail = [
				...this.trail,
				{ userId, username, displayName, nickname, imageUrl },
			]
		}
		this.list = list
		this.query = ""
		this.#open()
	}

	back(depth = this.trail.length - 1): void {
		this.trail = this.trail.slice(0, Math.max(0, depth))
		this.query = ""
		this.#open()
	}

	clearPlayers(): void {
		this.query = this.playerQuery = ""
		this.#open()
	}

	async refresh(): Promise<void> {
		this.#pages.delete(this.#key())
		await Promise.all([
			this.load(),
			this.selectedUserId === null
				? Promise.resolve()
				: this.loadProfile(this.selectedUserId, true),
		])
	}

	async load(more = false): Promise<void> {
		const key = this.#key(),
			cursor = more ? (this.page?.nextCursor ?? "") : ""
		if (!this.#ready() || (more && !cursor)) return
		void this.#listRequest?.cancel()
		const request = this.#request(cursor)
		this.#listRequest = request
		this.loading = !more
		this.loadingMore = more
		this.loadError = ""
		try {
			const received = await request
			if (request !== this.#listRequest) return
			this.#receive(key, more ? this.#append(received) : received)
		} catch (error) {
			if (request !== this.#listRequest) return
			this.loadError = errorMessage(error, "Could not load this list. Try again.")
		} finally {
			if (request === this.#listRequest) {
				this.#listRequest = null
				this.loading = this.loadingMore = false
			}
		}
	}

	#ready(): boolean {
		return (
			this.accountId > 0 &&
			(this.list === playersView ? !!this.playerQuery : this.subjectUserId > 0)
		)
	}

	#request(cursor: string): ReturnType<typeof accountBackend.GetSocialPage> {
		return this.list === playersView
			? accountBackend.SearchPlayers(this.accountId, this.playerQuery, cursor)
			: accountBackend.GetSocialPage(
					this.accountId,
					this.subjectUserId,
					this.list,
					cursor,
				)
	}

	#append(received: SocialListPage): SocialListPage {
		return {
			...received,
			people: [...(this.page?.people ?? []), ...(received.people ?? [])],
		}
	}

	#receive(key: string, page: SocialListPage): void {
		this.page = page
		this.#pages.set(key, { page, fetchedAt: Date.now() })
		if (page.requestCount !== null) this.requestCount = page.requestCount
	}

	select(person: SocialUser | null): void {
		if (person?.userId === this.selectedUserId) return
		this.selectedPerson = person
		void this.#profileRequest?.cancel()
		this.#profileRequest = null
		this.profile = null
		this.profileError = ""
		this.loadingProfile = false
		if (person) void this.loadProfile(person.userId)
	}

	async loadProfile(userId: number, force = false): Promise<void> {
		const cached = this.#profiles.get(userId)
		if (cached && Date.now() - cached.fetchedAt < detailsKeepMs) {
			this.profile = cached.profile
			if (!force && isFresh(cached.fetchedAt)) return
		}
		void this.#profileRequest?.cancel()
		const request = accountBackend.GetSocialProfile(this.accountId, userId)
		this.#profileRequest = request
		this.loadingProfile = true
		this.profileError = ""
		try {
			const profile = await request
			if (request !== this.#profileRequest) return
			this.#profiles.set(userId, { profile, fetchedAt: Date.now() })
			this.profile = profile
		} catch (error) {
			if (request !== this.#profileRequest) return
			this.profileError = errorMessage(
				error,
				"Could not load this profile. Refresh to retry.",
			)
		} finally {
			if (request === this.#profileRequest) {
				this.#profileRequest = null
				this.loadingProfile = false
			}
		}
	}

	async respond(person: SocialUser, accept: boolean): Promise<void> {
		if (this.responding !== null) return
		const epoch = this.#epoch,
			name = personName(person)
		this.closeMenu()
		this.responding = { userId: person.userId, accept }
		try {
			if (accept)
				await accountBackend.AcceptFriendRequest(this.accountId, person.userId)
			else
				await accountBackend.DeclineFriendRequest(this.accountId, person.userId)
			if (epoch !== this.#epoch) return
			this.#remove(person.userId, SocialList.SocialRequests)
			if (this.requestCount) this.requestCount--
			if (accept)
				this.#pages.delete(`${this.selfUserId}:${SocialList.SocialFriends}`)
			this.notifications.show({
				id: `friend-request-${person.userId}`,
				title: accept ? "Friend request accepted" : "Friend request declined",
				tone: "success",
				message: accept
					? `${name} is now a friend.`
					: `The request from ${name} was declined.`,
			})
		} catch (error) {
			if (epoch !== this.#epoch) return
			this.error = errorMessage(
				error,
				accept
					? "Could not accept the friend request. Try again."
					: "Could not decline the friend request. Try again.",
			)
		} finally {
			if (epoch === this.#epoch) this.responding = null
		}
	}

	async unfriend(person: SocialUser): Promise<boolean> {
		if (this.unfriending) return false
		const epoch = this.#epoch
		this.unfriending = true
		this.unfriendError = ""
		try {
			await accountBackend.Unfriend(this.accountId, person.userId)
			if (epoch !== this.#epoch) return true
			this.#remove(person.userId, SocialList.SocialFriends)
			this.notifications.show({
				id: `unfriend-${person.userId}`,
				title: "Unfriended",
				tone: "success",
				message: `${personName(person)} is no longer a friend.`,
			})
			return true
		} catch (error) {
			if (epoch === this.#epoch) {
				this.unfriendError = errorMessage(
					error,
					"Could not unfriend this user. Try again.",
				)
			}
			return false
		} finally {
			if (epoch === this.#epoch) this.unfriending = false
		}
	}

	async setFollowing(person: SocialSubject, following: boolean): Promise<void> {
		if (this.updating !== null) return
		const epoch = this.#epoch,
			name = personName(person)
		this.updating = "follow"
		try {
			if (following) await accountBackend.Follow(this.accountId, person.userId)
			else await accountBackend.Unfollow(this.accountId, person.userId)
			if (epoch !== this.#epoch) return
			this.#patchRelationship(person.userId, { following })
			this.notifications.show({
				id: `follow-${person.userId}`,
				title: following ? "Followed" : "Unfollowed",
				tone: "success",
				message: following
					? `${name} is now followed.`
					: `${name} is no longer followed.`,
			})
		} catch (error) {
			if (epoch !== this.#epoch) return
			this.error = errorMessage(
				error,
				following
					? "Could not follow this user. Try again."
					: "Could not unfollow this user. Try again.",
			)
		} finally {
			if (epoch === this.#epoch) this.updating = null
		}
	}

	async setNickname(person: SocialSubject, nickname: string): Promise<boolean> {
		if (this.updating !== null) return false
		const epoch = this.#epoch
		this.updating = "nickname"
		this.nicknameError = ""
		try {
			const saved = await accountBackend.SetNickname(
				this.accountId,
				person.userId,
				nickname,
			)
			if (epoch !== this.#epoch) return true
			this.#patchRelationship(person.userId, { nickname: saved })
			this.#patchNickname(person.userId, saved)
			this.notifications.show({
				id: `nickname-${person.userId}`,
				title: saved ? "Nickname saved" : "Nickname removed",
				tone: "success",
				message: saved
					? `${person.displayName || person.username} now shows as ${saved} on Roblox.`
					: `${person.displayName || person.username} shows their display name on Roblox again.`,
			})
			return true
		} catch (error) {
			if (epoch === this.#epoch) {
				this.nicknameError = errorMessage(
					error,
					"Could not save the nickname. Try again.",
				)
			}
			return false
		} finally {
			if (epoch === this.#epoch) this.updating = null
		}
	}

	#patchRelationship(userId: number, change: Partial<SocialRelationship>): void {
		const cached = this.#profiles.get(userId),
			patch = (profile: SocialProfileSnapshot): SocialProfileSnapshot => ({
				...profile,
				relationship: { ...profile.relationship, ...change },
			})
		if (cached) cached.profile = patch(cached.profile)
		if (this.profile?.userId === userId) this.profile = patch(this.profile)
	}

	#patchNickname(userId: number, nickname: string): void {
		const pages = [...this.#pages.values()].map((entry) => entry.page)
		if (this.page) pages.push(this.page)
		for (const person of [
			...pages.flatMap((page) => page.people ?? []),
			...this.trail,
		]) {
			if (person.userId === userId) person.nickname = nickname
		}
	}

	openMenu(event: MouseEvent | KeyboardEvent, person: SocialUser): void {
		event.preventDefault()
		event.stopPropagation()
		const bounds = (event.currentTarget as HTMLElement).getBoundingClientRect(),
			atPointer = event instanceof MouseEvent && event.type === "contextmenu"
		this.menu = {
			person,
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
				tone: "success",
				message: `${label} copied to clipboard.`,
			})
		} catch {
			this.error = `${label} could not be copied. Try again.`
		}
	}

	async openProfile(person: SocialSubject): Promise<void> {
		await this.openLink(`https://www.roblox.com/users/${person.userId}/profile`)
	}

	async openLink(url: string): Promise<void> {
		this.closeMenu()
		try {
			await Browser.OpenURL(url)
		} catch (error) {
			this.error = errorMessage(error, "Could not open the link. Try again.")
		}
	}

	reset(): void {
		this.#switchAccount(0, 0)
		this.trail = []
		this.query = this.playerQuery = ""
		this.list = SocialList.SocialFriends
	}

	#key(): string {
		return this.list === playersView
			? `${playersView}:${this.playerQuery.toLowerCase()}`
			: `${this.subjectUserId}:${this.list}`
	}

	#open(): void {
		void this.#listRequest?.cancel()
		this.#listRequest = null
		this.loading = this.loadingMore = false
		this.loadError = ""
		this.closeMenu()
		this.select(null)
		const cached = this.#pages.get(this.#key())
		this.page = cached?.page ?? null
		if (!cached || !isFresh(cached.fetchedAt)) void this.load()
	}

	#remove(userId: number, list: SocialList): void {
		const key = `${this.selfUserId}:${list}`,
			cached = this.#pages.get(key)
		if (!cached) return
		const page = {
			...cached.page,
			people: (cached.page.people ?? []).filter(
				(person) => person.userId !== userId,
			),
		}
		this.#pages.set(key, { page, fetchedAt: cached.fetchedAt })
		if (key !== this.#key()) return
		this.page = page
		if (this.selectedUserId === userId) this.select(null)
	}

	#switchAccount(accountId: number, selfUserId: number): void {
		this.#epoch++
		void this.#listRequest?.cancel()
		void this.#profileRequest?.cancel()
		this.#listRequest = this.#profileRequest = null
		this.#pages.clear()
		this.#profiles.clear()
		this.accountId = accountId
		this.selfUserId = selfUserId
		this.page = this.profile = this.menu = this.selectedPerson = null
		this.responding = this.requestCount = null
		this.loading = this.loadingMore = this.loadingProfile = false
		this.unfriending = false
		this.updating = null
		this.loadError = this.profileError = this.unfriendError = this.error = ""
		this.nicknameError = ""
	}
}

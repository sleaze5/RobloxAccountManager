import { Browser } from "@wailsio/runtime"
import { copyText } from "../shared/clipboard"
import {
	accountBackend,
	FriendStatus,
	SocialList,
	SocialProfileSection,
	TrustedStatus,
} from "../backend/bridge"
import type {
	SocialListPage,
	SocialProfileSnapshot,
	SocialRelationship,
	SocialUser,
	UserPresence,
	SocialRelationshipSnapshot,
} from "../backend/bridge"
import type { NotificationCenter } from "../notifications/notification-center.svelte"
import { errorMessage } from "../shared/errors"
import { detailsKeepMs, isFresh } from "../shared/freshness"
import { menuPointFor } from "../shared/presence"
import { SectionLoad } from "../shared/section-load.svelte"
import type { SectionFields } from "../shared/section-load.svelte"
import { relationshipActionsById } from "./relationship"
import type { RelationshipActionId } from "./relationship"

// Sections load in this order, so that the header and the actions fill first.
const socialProfileSections: SectionFields<
	SocialProfileSnapshot,
	SocialProfileSection
>[] = [
	{
		section: SocialProfileSection.SocialSectionRelationship,
		fields: ["relationship", "unavailable"],
	},
	{
		section: SocialProfileSection.SocialSectionAbout,
		fields: ["description", "verified", "banned", "createdAtMs"],
	},
	{ section: SocialProfileSection.SocialSectionFriends, fields: ["friendCount"] },
	{ section: SocialProfileSection.SocialSectionFollowers, fields: ["followerCount"] },
	{
		section: SocialProfileSection.SocialSectionFollowing,
		fields: ["followingCount"],
	},
	{ section: SocialProfileSection.SocialSectionPremium, fields: ["premium"] },
	{ section: SocialProfileSection.SocialSectionSocialLinks, fields: ["socialLinks"] },
	{
		section: SocialProfileSection.SocialSectionPrimaryGroup,
		fields: ["primaryGroup"],
	},
	{
		section: SocialProfileSection.SocialSectionRobloxBadges,
		fields: ["robloxBadges"],
	},
	{
		section: SocialProfileSection.SocialSectionInventory,
		fields: ["inventoryVisible"],
	},
	{
		section: SocialProfileSection.SocialSectionConnections,
		fields: ["savedAccountId", "connections"],
	},
	{
		section: SocialProfileSection.SocialSectionCommunities,
		fields: ["communityCount"],
	},
	{
		section: SocialProfileSection.SocialSectionPreviousNames,
		fields: ["previousNames"],
	},
	{
		section: SocialProfileSection.SocialSectionCollectibles,
		fields: ["collectibleCount", "collectibleRecentAveragePriceTotal"],
	},
]

function emptySocialProfile(userId: number): SocialProfileSnapshot {
	return {
		userId,
		section: SocialProfileSection.$zero,
		isComplete: false,
		description: "",
		verified: false,
		banned: false,
		premium: null,
		inventoryVisible: null,
		createdAtMs: 0,
		friendCount: null,
		followerCount: null,
		followingCount: null,
		previousNames: null,
		primaryGroup: null,
		robloxBadges: null,
		socialLinks: null,
		savedAccountId: 0,
		connections: null,
		relationship: {
			self: false,
			friend: FriendStatus.$zero,
			trusted: TrustedStatus.$zero,
			following: false,
			followedBy: false,
			nickname: "",
		},
		communityCount: null,
		collectibleCount: null,
		collectibleRecentAveragePriceTotal: null,
		unavailable: null,
	}
}

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

const presenceRefreshError = "Presence could not be refreshed."

export type BulkRemovalList =
	| SocialList.SocialFriends
	| SocialList.SocialFollowing
	| SocialList.SocialRequests

export function isBulkRemovalList(list: SocialView): list is BulkRemovalList {
	return (
		list === SocialList.SocialFriends ||
		list === SocialList.SocialFollowing ||
		list === SocialList.SocialRequests
	)
}

function counted(count: number, singular: string, plural: string): string {
	return `${count} ${count === 1 ? singular : plural}`
}

export const bulkRemovalCopy: Record<
	BulkRemovalList,
	{
		action: string
		progress: string
		done: string
		consequence: string
		result: (count: number) => string
	}
> = {
	[SocialList.SocialFriends]: {
		action: "Unfriend",
		progress: "Unfriending…",
		done: "Unfriended",
		consequence:
			"This ends each Roblox friendship. To be friends again, one of you must send a new request.",
		result: (count) =>
			`${counted(count, "friend was", "friends were")} unfriended.`,
	},
	[SocialList.SocialFollowing]: {
		action: "Unfollow",
		progress: "Unfollowing…",
		done: "Unfollowed",
		consequence: "You can follow them again at any time.",
		result: (count) => `${counted(count, "user was", "users were")} unfollowed.`,
	},
	[SocialList.SocialRequests]: {
		action: "Decline",
		progress: "Declining…",
		done: "Friend requests declined",
		consequence: "They can send a new request later.",
		result: (count) =>
			`${counted(count, "request was", "requests were")} declined.`,
	},
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
	profileLoad = $state<SectionLoad<
		SocialProfileSnapshot,
		SocialProfileSection
	> | null>(null)
	savingNickname = $state(false)
	relationshipError = $state("")
	menuRelationship = $state<SocialRelationshipSnapshot | null>(null)
	menuRelationshipFailed = $state(false)
	nicknameError = $state("")
	menu = $state<SocialMenu | null>(null)
	error = $state("")
	trustedOnly = $state(false)
	selecting = $state(false)
	selectedUserIds = $state<number[]>([])
	removingSelected = $state(false)
	removeSelectedError = $state("")
	pendingAction = $state<{ userId: number; action: RelationshipActionId } | null>(
		null,
	)
	#epoch = 0
	#listRequest: ReturnType<typeof accountBackend.GetSocialPage> | null = null
	#relationshipRequest: ReturnType<
		typeof accountBackend.GetSocialRelationship
	> | null = null
	readonly #relationships = new Map<number, SocialRelationshipSnapshot>()
	#presenceRequest: ReturnType<typeof accountBackend.GetSocialPresences> | null = null
	readonly #pages = new Map<string, { page: SocialListPage; fetchedAt: number }>()
	readonly #profiles = new Map<
		number,
		{
			load: SectionLoad<SocialProfileSnapshot, SocialProfileSection>
			fetchedAt: number
		}
	>()

	constructor(private readonly notifications: NotificationCenter) {}

	get subject(): SocialSubject | null {
		return this.trail.at(-1) ?? null
	}

	get subjectUserId(): number {
		return this.subject?.userId ?? this.selfUserId
	}

	get people(): SocialUser[] {
		const people = this.page?.people ?? [],
			text = this.query.trim().toLowerCase().replace(/^@/, ""),
			trusted = this.showsTrustedFilter && this.trustedOnly
		if ((!text && !trusted) || this.list === playersView) return people
		return people.filter(
			(person) =>
				(!trusted || person.trusted) &&
				(person.displayName.toLowerCase().includes(text) ||
					person.username.toLowerCase().includes(text)),
		)
	}

	// Roblox only lists the trusted friends of the account itself.
	get showsTrustedFilter(): boolean {
		return this.list === SocialList.SocialFriends && !this.subject
	}

	get trustedCount(): number {
		return (this.page?.people ?? []).filter((person) => person.trusted).length
	}

	get canRemoveInBulk(): boolean {
		return !this.subject && isBulkRemovalList(this.list)
	}

	setTrustedOnly(trustedOnly: boolean): void {
		this.trustedOnly = trustedOnly
		this.selectedUserIds = []
	}

	setSelecting(selecting: boolean): void {
		this.selecting = selecting && this.canRemoveInBulk
		this.selectedUserIds = []
		this.removeSelectedError = ""
		if (this.selecting) this.closeMenu()
	}

	toggleSelected(userId: number): void {
		this.selectedUserIds = this.selectedUserIds.includes(userId)
			? this.selectedUserIds.filter((selectedId) => selectedId !== userId)
			: [...this.selectedUserIds, userId]
	}

	toggleAllVisible(): void {
		const visible = this.people.map((person) => person.userId),
			allSelected =
				visible.length > 0 &&
				visible.every((userId) => this.selectedUserIds.includes(userId))
		this.selectedUserIds = allSelected ? [] : visible
	}

	async removeSelected(): Promise<boolean> {
		const list = this.list,
			userIds = [...this.selectedUserIds]
		if (
			this.removingSelected ||
			userIds.length === 0 ||
			this.subject ||
			!isBulkRemovalList(list)
		) {
			return false
		}
		const epoch = this.#epoch
		this.removingSelected = true
		this.removeSelectedError = ""
		try {
			const removal = await accountBackend.RemoveSocialConnections(
				this.accountId,
				list,
				userIds,
			)
			if (epoch !== this.#epoch) return true
			const removed = removal.removed ?? []
			for (const userId of removed) this.#remove(userId, list)
			if (list === SocialList.SocialRequests && this.requestCount !== null) {
				this.requestCount = Math.max(0, this.requestCount - removed.length)
			}
			this.selectedUserIds = this.selectedUserIds.filter(
				(userId) => !removed.includes(userId),
			)
			if (removed.length > 0) {
				this.notifications.show({
					id: `social-bulk-remove-${list}`,
					title: bulkRemovalCopy[list].done,
					tone: "success",
					message: bulkRemovalCopy[list].result(removed.length),
				})
			}
			if (removal.failedCount > 0) {
				this.removeSelectedError = `${counted(removal.failedCount, "person was", "people were")} not changed. ${removal.failureMessage}`
				return false
			}
			this.selecting = false
			return true
		} catch (error) {
			if (epoch === this.#epoch) {
				this.removeSelectedError = errorMessage(
					error,
					"Could not remove the selected people. Try again.",
				)
			}
			return false
		} finally {
			if (epoch === this.#epoch) this.removingSelected = false
		}
	}

	get profile(): SocialProfileSnapshot | null {
		return this.profileLoad?.hasAnySection ? this.profileLoad.snapshot : null
	}

	get loadingProfile(): boolean {
		return this.profileLoad?.isLoading ?? false
	}

	get profileError(): string {
		return this.profileLoad?.error ?? ""
	}

	// profileRelationship is null until the relationship section arrives, so
	// that the profile never offers actions for a relationship it has not read.
	get profileRelationship(): SocialRelationshipSnapshot | null {
		const load = this.profileLoad
		if (!load?.hasSection(SocialProfileSection.SocialSectionRelationship))
			return null
		return {
			relationship: load.snapshot.relationship,
			unavailable: load.snapshot.unavailable,
		}
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
		const index = this.trail.findIndex(
			(trailEntry) => trailEntry.userId === person.userId,
		)
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

	async refreshPresences(): Promise<void> {
		const page = this.page,
			userIds = this.#presenceUserIds()
		if (!page || userIds.length === 0 || this.#presenceRequest) return
		const epoch = this.#epoch,
			request = accountBackend.GetSocialPresences(this.accountId, userIds)
		this.#presenceRequest = request
		try {
			const presences = (await request) ?? []
			if (request !== this.#presenceRequest || epoch !== this.#epoch) return
			if (this.page === page) this.#installPresences(page, presences)
			if (this.error === presenceRefreshError) this.error = ""
		} catch {
			if (request !== this.#presenceRequest || epoch !== this.#epoch) return
			this.error = presenceRefreshError
		} finally {
			if (request === this.#presenceRequest) this.#presenceRequest = null
		}
	}

	#presenceUserIds(): number[] {
		if (this.loading || this.loadingMore) return []
		return (this.page?.people ?? [])
			.filter((person) => person.presence)
			.map((person) => person.userId)
	}

	#installPresences(page: SocialListPage, presences: UserPresence[]): void {
		const byUser = new Map(
				presences.map((presence) => [presence.userId, presence]),
			),
			people: SocialUser[] = []
		for (const person of page.people ?? []) {
			const presence = person.presence && byUser.get(person.userId)
			people.push(presence ? { ...person, presence } : person)
		}
		const next = { ...page, people },
			key = this.#key(),
			cached = this.#pages.get(key)
		this.page = next
		if (cached?.page === page) {
			this.#pages.set(key, { page: next, fetchedAt: cached.fetchedAt })
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
		this.profileLoad?.cancel()
		this.profileLoad = null
		if (person) void this.loadProfile(person.userId)
	}

	async loadProfile(userId: number, force = false): Promise<void> {
		const now = Date.now()
		for (const [cachedUserId, entry] of this.#profiles) {
			if (now - entry.fetchedAt >= detailsKeepMs && !entry.load.isLoading) {
				entry.load.reset()
				this.#profiles.delete(cachedUserId)
			}
		}
		let entry = this.#profiles.get(userId)
		if (!entry) {
			entry = {
				load: new SectionLoad(
					socialProfileSections,
					() => emptySocialProfile(userId),
					"Could not load this profile. Refresh to retry.",
				),
				fetchedAt: 0,
			}
			this.#profiles.set(userId, entry)
		}
		const { load } = entry,
			epoch = this.#epoch,
			accountId = this.accountId
		this.profileLoad = load
		if (load.isLoading || (!force && isFresh(entry.fetchedAt))) return
		const isCurrentLoad = await load.load(
			(section) => accountBackend.GetSocialProfile(accountId, userId, section),
			() => epoch === this.#epoch,
		)
		if (isCurrentLoad && !load.error) entry.fetchedAt = Date.now()
	}

	// runRelationshipAction runs a confirmed relationship action and returns
	// whether it succeeded. A failure appears in relationshipError.
	async runRelationshipAction(
		person: SocialUser,
		actionId: RelationshipActionId,
	): Promise<boolean> {
		if (this.pendingAction) return false
		const epoch = this.#epoch,
			action = relationshipActionsById[actionId],
			operation = this.#relationshipOperation(person.userId, actionId)
		this.closeMenu()
		this.pendingAction = { userId: person.userId, action: actionId }
		this.relationshipError = ""
		try {
			await operation.send()
			if (epoch !== this.#epoch) return true
			this.#patchRelationship(person.userId, operation.change)
			operation.after?.()
			this.notifications.show({
				id: `relationship-${person.userId}`,
				title: action.done,
				tone: "success",
				message: action.result(personName(person)),
			})
			return true
		} catch (error) {
			if (epoch === this.#epoch) {
				this.relationshipError = errorMessage(error, action.failure)
			}
			return false
		} finally {
			if (epoch === this.#epoch) this.pendingAction = null
		}
	}

	// Roblox cancels a sent friend request with the unfriend request, and a sent
	// trusted friend request with the remove trusted friend request.
	#relationshipOperation(
		userId: number,
		actionId: RelationshipActionId,
	): {
		send: () => Promise<void>
		change: Partial<SocialRelationship>
		after?: () => void
	} {
		const accountId = this.accountId,
			answerRequest = (accept: boolean) => () => {
				this.#remove(userId, SocialList.SocialRequests)
				if (this.requestCount) this.requestCount--
				if (accept)
					this.#pages.delete(`${this.selfUserId}:${SocialList.SocialFriends}`)
			}
		switch (actionId) {
			case "add-friend":
				return {
					send: () => accountBackend.SendFriendRequest(accountId, userId),
					change: { friend: FriendStatus.FriendStatusRequestSent },
				}
			case "cancel-friend":
				return {
					send: () => accountBackend.Unfriend(accountId, userId),
					change: { friend: FriendStatus.FriendStatusNone },
				}
			case "accept-friend":
				return {
					send: () => accountBackend.AcceptFriendRequest(accountId, userId),
					change: { friend: FriendStatus.FriendStatusFriends },
					after: answerRequest(true),
				}
			case "decline-friend":
				return {
					send: () => accountBackend.DeclineFriendRequest(accountId, userId),
					change: { friend: FriendStatus.FriendStatusNone },
					after: answerRequest(false),
				}
			case "unfriend":
				return {
					send: () => accountBackend.Unfriend(accountId, userId),
					change: {
						friend: FriendStatus.FriendStatusNone,
						trusted: TrustedStatus.TrustedStatusNone,
					},
					after: () => this.#remove(userId, SocialList.SocialFriends),
				}
			case "add-trusted":
				return {
					send: () =>
						accountBackend.SendTrustedFriendRequest(accountId, userId),
					change: { trusted: TrustedStatus.TrustedStatusRequestSent },
				}
			case "accept-trusted":
				return {
					send: () =>
						accountBackend.AcceptTrustedFriendRequest(accountId, userId),
					change: { trusted: TrustedStatus.TrustedStatusTrusted },
				}
			case "decline-trusted":
				return {
					send: () =>
						accountBackend.DeclineTrustedFriendRequest(accountId, userId),
					change: { trusted: TrustedStatus.TrustedStatusNone },
				}
			case "cancel-trusted":
			case "remove-trusted":
				return {
					send: () => accountBackend.RemoveTrustedFriend(accountId, userId),
					change: { trusted: TrustedStatus.TrustedStatusNone },
				}
			case "follow":
			case "follow-back":
				return {
					send: () => accountBackend.Follow(accountId, userId),
					change: { following: true },
				}
			// The remaining action is unfollow.
			default:
				return {
					send: () => accountBackend.Unfollow(accountId, userId),
					change: { following: false },
					after: () => this.#remove(userId, SocialList.SocialFollowing),
				}
		}
	}

	isActionPending(userId: number, action: RelationshipActionId): boolean {
		return (
			this.pendingAction?.userId === userId &&
			this.pendingAction.action === action
		)
	}

	async setNickname(person: SocialSubject, nickname: string): Promise<boolean> {
		if (this.savingNickname) return false
		const epoch = this.#epoch
		this.savingNickname = true
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
			if (epoch === this.#epoch) this.savingNickname = false
		}
	}

	#patchRelationship(userId: number, change: Partial<SocialRelationship>): void {
		const load = this.#profiles.get(userId)?.load
		if (load) {
			load.snapshot.relationship = { ...load.snapshot.relationship, ...change }
		}
		const relationship = this.#relationships.get(userId)
		if (relationship) {
			this.#relationships.set(userId, {
				...relationship,
				relationship: { ...relationship.relationship, ...change },
			})
		}
		if (change.trusted !== undefined) {
			const trusted = change.trusted === TrustedStatus.TrustedStatusTrusted
			for (const person of this.#listedPeople()) {
				if (person.userId === userId) person.trusted = trusted
			}
		}
	}

	#patchNickname(userId: number, nickname: string): void {
		for (const person of [...this.#listedPeople(), ...this.trail]) {
			if (person.userId === userId) person.nickname = nickname
		}
	}

	#listedPeople(): SocialUser[] {
		const pages = [...this.#pages.values()].map((entry) => entry.page)
		if (this.page) pages.push(this.page)
		return pages.flatMap((page) => page.people ?? [])
	}

	openMenu(event: MouseEvent | KeyboardEvent, person: SocialUser): void {
		event.preventDefault()
		event.stopPropagation()
		this.menu = { person, ...menuPointFor(event, 2) }
		void this.#loadMenuRelationship(person.userId)
	}

	closeMenu(): void {
		this.menu = null
		void this.#relationshipRequest?.cancel()
		this.#relationshipRequest = null
	}

	async #loadMenuRelationship(userId: number): Promise<void> {
		void this.#relationshipRequest?.cancel()
		const load = this.#profiles.get(userId)?.load
		this.menuRelationship =
			this.#relationships.get(userId) ??
			(load?.hasSection(SocialProfileSection.SocialSectionRelationship)
				? {
						relationship: load.snapshot.relationship,
						unavailable: load.snapshot.unavailable,
					}
				: null)
		this.menuRelationshipFailed = false
		const request = accountBackend.GetSocialRelationship(this.accountId, userId)
		this.#relationshipRequest = request
		try {
			const snapshot = await request
			if (request !== this.#relationshipRequest) return
			this.#relationships.set(userId, snapshot)
			this.menuRelationship = snapshot
		} catch {
			if (request === this.#relationshipRequest) {
				this.menuRelationshipFailed = this.menuRelationship === null
			}
		}
	}

	async copy(value: string | number, label: string): Promise<void> {
		this.closeMenu()
		if (
			!(await copyText(
				this.notifications,
				String(value),
				`${label} copied to clipboard.`,
			))
		) {
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
		void this.#presenceRequest?.cancel()
		this.#presenceRequest = null
		this.loading = this.loadingMore = false
		this.loadError = ""
		this.closeMenu()
		this.select(null)
		this.selecting = false
		this.selectedUserIds = []
		this.removeSelectedError = ""
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
		void this.#presenceRequest?.cancel()
		void this.#relationshipRequest?.cancel()
		this.#listRequest = this.#presenceRequest = null
		this.#relationshipRequest = null
		this.#pages.clear()
		for (const entry of this.#profiles.values()) entry.load.reset()
		this.#profiles.clear()
		this.#relationships.clear()
		this.accountId = accountId
		this.selfUserId = selfUserId
		this.page = this.profileLoad = this.menu = this.selectedPerson = null
		this.requestCount = this.menuRelationship = this.pendingAction = null
		this.loading = this.loadingMore = false
		this.savingNickname = this.menuRelationshipFailed = false
		this.loadError = this.error = this.relationshipError = ""
		this.nicknameError = ""
		this.trustedOnly = this.selecting = this.removingSelected = false
		this.selectedUserIds = []
		this.removeSelectedError = ""
	}
}

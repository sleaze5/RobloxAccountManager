import { accountBackend } from "../backend/bridge"
import type {
	GroupProfile,
	GroupSummary,
	UserGroup,
	UserGroupList,
} from "../backend/bridge"
import { AccountScope } from "../accounts/account-scope"
import type { AccountStore } from "../accounts/account-store.svelte"
import type { NotificationCenter } from "../notifications/notification-center.svelte"
import type { CancellableRequest } from "../shared/cancellable-request"
import { copyLines, copyValue, notifyCopyFailure } from "../shared/clipboard"
import { errorMessage } from "../shared/errors"
import { ListSelection } from "../shared/list-selection.svelte"
import { menuPointFor } from "../shared/transitions"
import { robloxGroupUrl } from "../shared/roblox-pages"
import type { PageLocationKeys } from "../layout/page-history"

type GroupsMode = "mine" | "search"

// GroupSubpage is a page of the selected group that replaces its profile.
export type GroupSubpage = "members" | "games" | "items" | "joinRequests"

type GroupAction =
	| "join"
	| "leave"
	| "primary"
	| "removePrimary"
	| "shout"
	| "description"

export const minGroupSearchLength = 2

export interface GroupMenu {
	group: GroupSummary
	x: number
	y: number
}

// GroupCopyField names a detail of a group that its menu and the selection
// copy.
export type GroupCopyField = "name" | "groupId" | "link"

const groupCopyLabels: Record<GroupCopyField, string> = {
	name: "Group name",
	groupId: "Group ID",
	link: "Group link",
}

function groupCopyValue(group: GroupSummary, field: GroupCopyField): string {
	if (field === "name") return group.name
	if (field === "groupId") return String(group.groupId)
	return robloxGroupUrl(group.groupId)
}

// GroupsLocation is what Groups shows: the list, the search, and the group
// and page that it opened. Its history goes back to it.
interface GroupsLocation {
	mode: GroupsMode
	searchedText: string
	groupId: number
	subpage: GroupSubpage | null
}

export function groupsLocationKeys(location: GroupsLocation): PageLocationKeys {
	return {
		context: `${location.mode}:${location.searchedText}`,
		selection: location.groupId
			? `${location.groupId}:${location.subpage ?? ""}`
			: "",
	}
}

// GroupsStore lists the groups of the selected account, searches Roblox
// groups, and shows one group at a time. Every request belongs to the
// account that was selected when it started.
export class GroupsStore {
	sidebarCollapsed = $state(false)
	sidebarWidth = $state(248)
	mode = $state<GroupsMode>("mine")
	query = $state("")
	searchedText = $state("")
	myGroups = $state.raw<UserGroupList | null>(null)
	isLoadingMyGroups = $state(false)
	myGroupsError = $state("")
	searchResults = $state.raw<GroupSummary[]>([])
	searchCursor = $state("")
	isSearching = $state(false)
	searchError = $state("")
	selectedGroupId = $state(0)
	profile = $state.raw<GroupProfile | null>(null)
	isLoadingProfile = $state(false)
	profileError = $state("")
	subpage = $state<GroupSubpage | null>(null)
	pendingAction = $state<GroupAction | null>(null)
	actionError = $state("")
	menu = $state<GroupMenu | null>(null)
	readonly selection = new ListSelection<number>()
	isLeavingGroups = $state(false)
	readonly #accounts: AccountStore
	readonly #notifications: NotificationCenter
	#scope: AccountScope | null = null
	#profileRequest: CancellableRequest<GroupProfile> | null = null
	#searchRequest: ReturnType<typeof accountBackend.SearchGroups> | null = null

	constructor(accounts: AccountStore, notifications: NotificationCenter) {
		this.#accounts = accounts
		this.#notifications = notifications
	}

	get scope(): AccountScope | null {
		return this.#scope
	}

	get accountId(): number {
		return this.#scope?.accountId ?? 0
	}

	// membershipOf returns the role of the account in the group, for a group
	// in its list.
	membershipOf(groupId: number): UserGroup | undefined {
		return this.myGroups?.groups?.find(
			(userGroup) => userGroup.group.groupId === groupId,
		)
	}

	// bindAccount starts the page for the selected account. The groups of the
	// previous account and its open group close.
	bindAccount(accountId: number): void {
		if (this.#scope?.accountId === accountId && this.#scope.isCurrent) return
		this.#scope?.dispose()
		this.#scope = accountId ? new AccountScope(accountId, this.#accounts) : null
		this.myGroups = null
		this.myGroupsError = ""
		this.searchResults = []
		this.searchCursor = this.searchError = ""
		this.searchedText = ""
		this.selection.setSelecting(false)
		this.closeMenu()
		this.closeGroup()
		void this.refreshMyGroups()
	}

	show(mode: GroupsMode): void {
		if (mode !== this.mode) this.selection.setSelecting(false)
		this.mode = mode
		if (
			mode === "search" &&
			this.query.trim() &&
			this.query.trim() !== this.searchedText
		) {
			void this.search()
		}
	}

	async refreshMyGroups(): Promise<void> {
		const scope = this.#scope,
			userId = Number(this.#accounts.selectedAccount.userId) || 0
		if (!scope?.isCurrent || !userId) return
		this.isLoadingMyGroups = true
		this.myGroupsError = ""
		try {
			const list = await scope.track(
				accountBackend.GetUserGroups(scope.accountId, userId),
			)
			if (scope === this.#scope && scope.isCurrent) this.myGroups = list
		} catch (error) {
			if (scope === this.#scope && scope.isCurrent) {
				this.myGroupsError = errorMessage(
					error,
					"Could not load the groups. Try again.",
				)
			}
		} finally {
			if (scope === this.#scope) this.isLoadingMyGroups = false
		}
	}

	async search(): Promise<void> {
		const text = this.query.trim()
		if (text.length < minGroupSearchLength) return
		this.mode = "search"
		this.searchedText = text
		this.searchResults = []
		this.searchCursor = ""
		await this.#requestSearchPage(text, "")
	}

	async loadMoreSearchResults(): Promise<void> {
		if (!this.searchCursor || this.isSearching) return
		await this.#requestSearchPage(this.searchedText, this.searchCursor)
	}

	async #requestSearchPage(text: string, cursor: string): Promise<void> {
		const scope = this.#scope
		if (!scope?.isCurrent) return
		void this.#searchRequest?.cancel()
		const request = accountBackend.SearchGroups(scope.accountId, text, cursor)
		this.#searchRequest = request
		this.isSearching = true
		this.searchError = ""
		try {
			const page = await scope.track(request)
			if (request !== this.#searchRequest || !scope.isCurrent) return
			this.searchResults = [...this.searchResults, ...(page.groups ?? [])]
			this.searchCursor = page.nextCursor
		} catch (error) {
			if (request === this.#searchRequest && scope.isCurrent) {
				this.searchError = errorMessage(
					error,
					"Could not search groups. Try again.",
				)
			}
		} finally {
			if (request === this.#searchRequest) {
				this.#searchRequest = null
				this.isSearching = false
			}
		}
	}

	clearSearch(): void {
		void this.#searchRequest?.cancel()
		this.query = this.searchedText = this.searchCursor = this.searchError = ""
		this.searchResults = []
		this.isSearching = false
	}

	get location(): GroupsLocation {
		return {
			mode: this.mode,
			searchedText: this.mode === "search" ? this.searchedText : "",
			groupId: this.selectedGroupId,
			subpage: this.subpage,
		}
	}

	async restoreLocation(location: GroupsLocation): Promise<void> {
		if (location.mode === "search" && location.searchedText !== this.searchedText) {
			this.query = location.searchedText
			await this.search()
		} else {
			if (location.mode === "search") this.query = location.searchedText
			this.show(location.mode)
		}
		if (location.groupId) {
			this.showGroup(location.groupId)
			this.subpage = location.subpage
		} else this.closeGroup()
	}

	showGroup(groupId: number): void {
		this.subpage = null
		if (groupId === this.selectedGroupId && this.profile) return
		this.selectedGroupId = groupId
		this.profile = null
		void this.refreshProfile()
	}

	// showGroupAsIDSearch opens a group that another page names, such as the
	// creator of a game, as a search by group ID. The account binds first,
	// because Groups may not have opened yet.
	showGroupAsIDSearch(accountId: number, groupId: number): void {
		this.bindAccount(accountId)
		this.sidebarCollapsed = false
		this.query = `id:${groupId}`
		void this.search()
		this.showGroup(groupId)
	}

	closeGroup(): void {
		void this.#profileRequest?.cancel()
		this.#profileRequest = null
		this.selectedGroupId = 0
		this.profile = null
		this.profileError = this.actionError = ""
		this.isLoadingProfile = false
		this.subpage = null
	}

	async refreshProfile(): Promise<void> {
		const scope = this.#scope,
			groupId = this.selectedGroupId
		if (!scope?.isCurrent || !groupId) return
		void this.#profileRequest?.cancel()
		const request = accountBackend.GetGroupProfile(scope.accountId, groupId)
		this.#profileRequest = request
		this.isLoadingProfile = true
		this.profileError = ""
		try {
			const profile = await scope.track(request)
			if (request === this.#profileRequest && scope.isCurrent)
				this.profile = profile
		} catch (error) {
			if (request === this.#profileRequest && scope.isCurrent) {
				this.profileError = errorMessage(
					error,
					"Could not load the group. Try again.",
				)
			}
		} finally {
			if (request === this.#profileRequest) {
				this.#profileRequest = null
				this.isLoadingProfile = false
			}
		}
	}

	async join(): Promise<boolean> {
		return await this.#runAction(
			"join",
			this.selectedGroupId,
			(accountId, groupId) => accountBackend.JoinGroup(accountId, groupId),
			"Could not join the group. Try again.",
		)
	}

	async leave(): Promise<boolean> {
		return await this.#runAction(
			"leave",
			this.selectedGroupId,
			(accountId, groupId) => accountBackend.LeaveGroup(accountId, groupId),
			"Could not leave the group. Try again.",
		)
	}

	async makePrimary(groupId: number): Promise<boolean> {
		return await this.#runAction(
			"primary",
			groupId,
			(accountId) => accountBackend.SetPrimaryGroup(accountId, groupId),
			"Could not make the group primary. Try again.",
		)
	}

	async removePrimary(groupId: number): Promise<boolean> {
		return await this.#runAction(
			"removePrimary",
			groupId,
			(accountId) => accountBackend.RemovePrimaryGroup(accountId),
			"Could not remove the primary group. Try again.",
		)
	}

	async postShout(message: string): Promise<boolean> {
		return await this.#runAction(
			"shout",
			this.selectedGroupId,
			(accountId, groupId) =>
				accountBackend.PostGroupShout(accountId, groupId, message),
			"Could not post the shout. Try again.",
		)
	}

	async saveDescription(description: string): Promise<boolean> {
		return await this.#runAction(
			"description",
			this.selectedGroupId,
			(accountId, groupId) =>
				accountBackend.SetGroupDescription(accountId, groupId, description),
			"Could not save the description. Try again.",
		)
	}

	openMenu(event: MouseEvent | KeyboardEvent, group: GroupSummary): void {
		event.preventDefault()
		event.stopPropagation()
		this.menu = { group, ...menuPointFor(event, 2) }
	}

	closeMenu(): void {
		this.menu = null
	}

	async copyField(group: GroupSummary, field: GroupCopyField): Promise<void> {
		this.closeMenu()
		await copyValue(
			this.#notifications,
			groupCopyValue(group, field),
			groupCopyLabels[field],
		)
	}

	async copySelectedField(
		groups: GroupSummary[],
		field: GroupCopyField,
	): Promise<void> {
		const selected = groups.filter((group) => this.selection.has(group.groupId)),
			label = groupCopyLabels[field]
		if (selected.length === 0) return
		if (
			!(await copyLines(
				this.#notifications,
				selected.map((group) => groupCopyValue(group, field)),
				label,
				"groups",
			))
		) {
			notifyCopyFailure(this.#notifications, label)
		}
	}

	// canLeave tells whether the account is a member that can leave the group.
	// Roblox does not let the owner leave, and ranks the owner 255.
	canLeave(groupId: number): boolean {
		const membership = this.membershipOf(groupId)
		return !!membership && membership.role.rank < 255
	}

	// leaveGroups leaves the groups, then reads the group list again, and
	// reports whether every group was left.
	async leaveGroups(groupIds: number[]): Promise<boolean> {
		const scope = this.#scope
		if (!scope?.isCurrent || this.isLeavingGroups || this.pendingAction)
			return false
		this.isLeavingGroups = true
		try {
			const departures = await Promise.allSettled(
				groupIds.map((groupId) =>
					scope.track(accountBackend.LeaveGroup(scope.accountId, groupId)),
				),
			)
			if (!scope.isCurrent) return false
			const failure = departures.find(
				(departure) => departure.status === "rejected",
			)
			if (failure) {
				this.#notifications.show({
					title: "Groups not left",
					tone: "error",
					message: errorMessage(
						failure.reason,
						"Could not leave the groups. Try again.",
					),
				})
				return false
			}
			this.selection.setSelecting(false)
			return true
		} finally {
			if (scope.isCurrent) {
				await this.refreshMyGroups()
				if (this.selectedGroupId && groupIds.includes(this.selectedGroupId))
					await this.refreshProfile()
			}
			this.isLeavingGroups = false
		}
	}

	clearActionError(): void {
		this.actionError = ""
	}

	reset(): void {
		this.#scope?.dispose()
		this.#scope = null
		this.myGroups = null
		this.myGroupsError = ""
		this.clearSearch()
		this.closeGroup()
		this.closeMenu()
		this.selection.setSelecting(false)
		this.mode = "mine"
		this.pendingAction = null
	}

	// #runAction changes the open group for the account, then reads the
	// group and the group list again, because the action changes both.
	async #runAction(
		action: GroupAction,
		groupId: number,
		run: (accountId: number, groupId: number) => CancellableRequest<unknown>,
		fallbackError: string,
	): Promise<boolean> {
		const scope = this.#scope
		if (!scope?.isCurrent || !groupId || this.pendingAction) return false
		this.pendingAction = action
		this.actionError = ""
		try {
			await scope.track(run(scope.accountId, groupId))
			if (!scope.isCurrent) return false
			await Promise.all([this.refreshProfile(), this.refreshMyGroups()])
			return true
		} catch (error) {
			if (scope.isCurrent) {
				const message = errorMessage(error, fallbackError)
				if (groupId === this.selectedGroupId) this.actionError = message
				else
					this.#notifications.show({
						title: "Group not changed",
						tone: "error",
						message,
					})
			}
			return false
		} finally {
			this.pendingAction = null
		}
	}
}

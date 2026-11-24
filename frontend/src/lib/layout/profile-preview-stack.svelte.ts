import type { AccountStore } from "../accounts/account-store.svelte"
import type { SocialUser } from "../backend/bridge"
import type { GamesStore } from "../games/games-store.svelte"
import type { GroupsStore } from "../groups/groups-store.svelte"
import type { NotificationCenter } from "../notifications/notification-center.svelte"
import { focusAfterRender } from "../shared/focus-after-render"
import {
	isSameProfilePreviewSubject,
	type ProfilePreviewSubject,
} from "../shared/profile-preview"
import type { RelationshipActionId } from "../social/relationship"
import type { SocialStore } from "../social/social-store.svelte"
import type { AppNavigation } from "./app-navigation.svelte"

interface FullProfilePages {
	accounts: AccountStore
	social: SocialStore
	groups: GroupsStore
	games: GamesStore
	navigation: AppNavigation
	notifications: NotificationCenter
}

// ProfilePreviewStack holds the previews that links opened above the page, so
// that the page stays. A preview that opens from another one stacks on it.
// The full profile opens on its own page, as a search by its ID with the
// profile opened.
export class ProfilePreviewStack {
	subjects = $state<ProfilePreviewSubject[]>([])
	readonly #pages: FullProfilePages

	constructor(pages: FullProfilePages) {
		this.#pages = pages
	}

	show(subject: ProfilePreviewSubject): void {
		const shownSubject = this.subjects.at(-1)
		if (shownSubject && isSameProfilePreviewSubject(shownSubject, subject)) return
		this.subjects = [...this.subjects, subject]
	}

	back(): void {
		this.subjects = this.subjects.slice(0, -1)
	}

	close(): void {
		this.subjects = []
	}

	openFullProfile(subject: ProfilePreviewSubject): void {
		this.close()
		const { accounts, social, groups, games, navigation } = this.#pages,
			account = accounts.selectedAccount
		if (subject.kind === "User")
			navigation.showPage("social", () =>
				social.showUser(account.id, Number(account.userId) || 0, subject),
			)
		else if (subject.kind === "Group")
			navigation.showPage("groups", () =>
				groups.showGroupAsIDSearch(account.id, subject.groupId),
			)
		else
			navigation.showPage("games", () => {
				void games.showPlaceAsIDSearch(subject)
				void focusAfterRender("[data-games-search]")
			})
	}

	// runRelationshipAction runs the action through Social, so that Social shows
	// the new relationship too. Social works for the selected account.
	async runRelationshipAction(
		person: SocialUser,
		actionId: RelationshipActionId,
	): Promise<boolean> {
		const { accounts, social, notifications } = this.#pages,
			account = accounts.selectedAccount
		if (social.accountId !== account.id)
			social.show(account.id, Number(account.userId) || 0)
		const isDone = await social.runRelationshipAction(person, actionId)
		if (!isDone && social.relationshipError)
			notifications.show({
				id: `error:${social.relationshipError}`,
				title: "Relationship not changed",
				tone: "error",
				message: social.relationshipError,
			})
		return isDone
	}

	// refreshChangedGroup shows a membership that a preview changed in Groups
	// too.
	refreshChangedGroup(groupId: number): void {
		const { groups } = this.#pages
		void groups.refreshMyGroups()
		if (groups.selectedGroupId === groupId) void groups.refreshProfile()
	}
}

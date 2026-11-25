import { untrack } from "svelte"
import type { AccountStore } from "../accounts/account-store.svelte"
import type { AccountPage } from "../accounts/account-pages"
import { gamesLocationKeys, type GamesStore } from "../games/games-store.svelte"
import { groupsLocationKeys, type GroupsStore } from "../groups/groups-store.svelte"
import type { LogsExplorerStore } from "../logs-explorer/logs-explorer-store.svelte"
import { socialLocationKeys, type SocialStore } from "../social/social-store.svelte"
import { PageHistory } from "./page-history"
import type { WorkspaceState } from "./workspace-state.svelte"

// HistoryPage names a page that remembers where it went.
type HistoryPage = "accounts" | "social" | "groups" | "games" | "logs-explorer"

interface PageHistoryNavigation {
	back: () => Promise<void>
	forward: () => Promise<void>
}

// trackPageHistories gives each page a history of the places that it showed,
// which it records as they change. The history of Social and Groups belongs
// to the selected account, so it starts again when another account is
// selected.
export function trackPageHistories(pages: {
	accounts: AccountStore
	workspace: WorkspaceState
	social: SocialStore
	groups: GroupsStore
	games: GamesStore
	logsExplorer: LogsExplorerStore
}): Record<HistoryPage, PageHistoryNavigation> {
	const { accounts, workspace, social, groups, games, logsExplorer } = pages
	const accountsHistory = new PageHistory<{
			accountId: number
			accountPage: "profile" | AccountPage
		}>(
			(location) => ({
				context: "",
				selection: `${location.accountId}:${location.accountPage}`,
			}),
			(location) => {
				if (
					accounts.accounts.some(
						(account) => account.id === location.accountId,
					)
				)
					accounts.selectAccount(location.accountId)
				workspace.accountPage = location.accountPage
			},
		),
		socialHistory = new PageHistory(socialLocationKeys, (location) =>
			social.restoreLocation(location),
		),
		groupsHistory = new PageHistory(groupsLocationKeys, (location) =>
			groups.restoreLocation(location),
		),
		gamesHistory = new PageHistory(gamesLocationKeys, (location) =>
			games.restoreLocation(location),
		),
		logsHistory = new PageHistory<string>(
			(selectedFile) => ({ context: "", selection: selectedFile }),
			(selectedFile) => {
				// A filter that hides the log file no longer applies to it.
				if (
					!logsExplorer.filteredSessions.some(
						(session) => session.fileName === selectedFile,
					)
				)
					logsExplorer.query = ""
				logsExplorer.selectedFile = selectedFile
			},
		)

	$effect(() => {
		const location = {
			accountId: accounts.selectedAccount.id,
			accountPage: workspace.accountPage,
		}
		if (location.accountId > 0) untrack(() => accountsHistory.record(location))
	})
	$effect(() => {
		void accounts.selectedAccount.id
		untrack(() => {
			socialHistory.clear()
			groupsHistory.clear()
		})
	})
	$effect(() => {
		const location = social.location
		untrack(() => socialHistory.record(location))
	})
	$effect(() => {
		const location = groups.location
		untrack(() => groupsHistory.record(location))
	})
	$effect(() => {
		const location = games.location
		untrack(() => gamesHistory.record(location))
	})
	$effect(() => {
		const selectedFile = logsExplorer.selectedFile
		if (selectedFile) untrack(() => logsHistory.record(selectedFile))
	})

	return {
		"accounts": accountsHistory,
		"social": socialHistory,
		"groups": groupsHistory,
		"games": gamesHistory,
		"logs-explorer": logsHistory,
	}
}

<script lang="ts">
	import warningIcon from "@phosphor-icons/core/regular/warning.svg?raw"
	import { onMount, tick, untrack } from "svelte"
	import { SvelteSet } from "svelte/reactivity"
	import AccountPlaceholder from "./lib/accounts/AccountPlaceholder.svelte"
	import AccountProfile from "./lib/accounts/AccountProfile.svelte"
	import AccountImageMenu from "./lib/accounts/AccountImageMenu.svelte"
	import ExpandedImageDialog from "./lib/shared/ExpandedImageDialog.svelte"
	import AccountContextMenu from "./lib/accounts/AccountContextMenu.svelte"
	import AccountSidebar from "./lib/accounts/AccountSidebar.svelte"
	import { AccountStore } from "./lib/accounts/account-store.svelte"
	import type { Account } from "./lib/accounts/account-model"
	import type { AccountProfileState } from "./lib/accounts/account-profile-state.svelte"
	import AddAccountDialog from "./lib/dialogs/AddAccountDialog.svelte"
	import AppLocationDialog from "./lib/dialogs/AppLocationDialog.svelte"
	import BrowserImportDialog from "./lib/dialogs/BrowserImportDialog.svelte"
	import DevelopmentBuildDialog from "./lib/dialogs/DevelopmentBuildDialog.svelte"
	import ImportVaultDialog from "./lib/dialogs/ImportVaultDialog.svelte"
	import LegacyVaultDialog from "./lib/dialogs/LegacyVaultDialog.svelte"
	import { chooseExportFile } from "./lib/dialogs/vault-transfer"
	import BrowserRuntimeDialog from "./lib/dialogs/BrowserRuntimeDialog.svelte"
	import NicknameDialog from "./lib/dialogs/NicknameDialog.svelte"
	import RelationshipActionDialog from "./lib/dialogs/RelationshipActionDialog.svelte"
	import type { RelationshipActionId } from "./lib/social/relationship"
	import RemoveConnectionsDialog from "./lib/dialogs/RemoveConnectionsDialog.svelte"
	import DescriptionEditDialog from "./lib/dialogs/DescriptionEditDialog.svelte"
	import LockVaultDialog from "./lib/dialogs/LockVaultDialog.svelte"
	import ManageTagsDialog from "./lib/dialogs/ManageTagsDialog.svelte"
	import RemoveAccountDialog from "./lib/dialogs/RemoveAccountDialog.svelte"
	import RobloxLaunchDialog from "./lib/dialogs/RobloxLaunchDialog.svelte"
	import RenewCookieDialog from "./lib/dialogs/RenewCookieDialog.svelte"
	import TestMasterPasswordDialog from "./lib/dialogs/TestMasterPasswordDialog.svelte"
	import VaultDialog from "./lib/dialogs/VaultDialog.svelte"
	import {
		accountBackend,
		CatalogCreatorType,
		FileState,
		LaunchMethod,
		RuntimeStatus,
	} from "./lib/backend/bridge"
	import type {
		AppBuild,
		AppLocation,
		GamePlace,
		RobloxNotificationUser,
		ShutdownEffects,
		SocialUser,
	} from "./lib/backend/bridge"
	import { browserStore } from "./lib/browser/browser-store.svelte"
	import { SocialStore } from "./lib/social/social-store.svelte"
	import type { SocialListKind } from "./lib/social/social-store.svelte"
	import { GamesStore, setGamesStore } from "./lib/games/games-store.svelte"
	import { CatalogStore } from "./lib/catalog/catalog-store.svelte"
	import { GroupsStore } from "./lib/groups/groups-store.svelte"
	import { setCreatorNavigation } from "./lib/shared/creator-navigation"
	import { LogsExplorerStore } from "./lib/logs-explorer/logs-explorer-store.svelte"
	import LaunchDrawer from "./lib/layout/LaunchDrawer.svelte"
	import PageNavigation from "./lib/layout/PageNavigation.svelte"
	import RetainedPage from "./lib/layout/RetainedPage.svelte"
	import StatusBar from "./lib/layout/StatusBar.svelte"
	import WindowTitleBar from "./lib/layout/WindowTitleBar.svelte"
	import { setWorkspace, WorkspaceState } from "./lib/layout/workspace-state.svelte"
	import { unsavedChangesGuard } from "./lib/layout/unsaved-changes-guard"
	import Notifications from "./lib/notifications/Notifications.svelte"
	import {
		NotificationCenter,
		setNotificationCenter,
	} from "./lib/notifications/notification-center.svelte"
	import { appSettings } from "./lib/settings/settings-store.svelte"
	import { errorMessage } from "./lib/shared/errors"
	import Icon from "./lib/shared/Icon.svelte"
	import OperationError from "./lib/shared/OperationError.svelte"
	import Tooltip from "./lib/shared/Tooltip.svelte"
	import { updateStore } from "./lib/updates/update-store.svelte"

	type ActiveDialog =
		| "add-account"
		| "edit-description"
		| "browser-import"
		| "import-vault"
		| "browser-runtime"
		| "lock-vault"
		| "manage-tags"
		| "remove-account"
		| "renew-cookie"
		| "test-password"
		| "relationship"
		| "nickname"
		| "remove-connections"
		| null
	type ActivePage =
		| "accounts"
		| "social"
		| "groups"
		| "games"
		| "catalog"
		| "logs-explorer"
		| "settings"
	type TabPage = Exclude<ActivePage, "settings">

	// The pages other than Accounts load as separate files, so that the app
	// starts faster. They begin to load once the app shows, so that opening a
	// page rarely waits.
	const pageImports = {
		social: () => import("./lib/social/SocialPage.svelte"),
		groups: () => import("./lib/groups/GroupsPage.svelte"),
		games: () => import("./lib/games/GamesPage.svelte"),
		catalog: () => import("./lib/catalog/CatalogPage.svelte"),
		logsExplorer: () => import("./lib/logs-explorer/LogsExplorerPage.svelte"),
		settings: () => import("./lib/settings/SettingsPage.svelte"),
	}

	const notificationCenter = new NotificationCenter(),
		store = new AccountStore(notificationCenter),
		logsExplorer = new LogsExplorerStore(),
		games = new GamesStore(notificationCenter),
		catalog = new CatalogStore(store),
		groups = new GroupsStore(store, notificationCenter),
		social = new SocialStore(notificationCenter),
		workspace = new WorkspaceState(),
		// The tab pages that the user opened since the vault unlocked. They stay
		// mounted and hidden while another page shows, so that switching back
		// keeps their state.
		openedTabPages = new SvelteSet<TabPage>(),
		passwordReminderNotificationId = "password-test-reminder"
	let build = $state<AppBuild | null>(null),
		developmentWarning = $state<"loading" | "open" | "closed">("loading"),
		appLocation = $state<AppLocation | null>(null),
		activePage = $state<ActivePage>("accounts"),
		settingsReturnPage = $state<TabPage>("accounts"),
		activeDialog = $state<ActiveDialog>(null),
		passwordReminderNotified = $state(false),
		tagAccountId = $state<number | null>(null),
		removeAccountId = $state<number | null>(null),
		relationshipTarget = $state<{
			person: SocialUser
			actionId: RelationshipActionId
		} | null>(null),
		nicknameTarget = $state<{ person: SocialUser; nickname: string } | null>(null),
		renewAccountId = $state<number | null>(null),
		importPath = $state("")
	let descriptionEditor = $state<{
		account: Account
		state: AccountProfileState
		description: string
	} | null>(null)
	// The browser to open once the browser runtime is ready: a page in the
	// browser of an account, or the import browser when accountId is null.
	let pendingBrowserOpening = $state<
			{ accountId: number | null; pageUrl: string } | undefined
		>(undefined),
		lockEffects = $state<ShutdownEffects>({
			accountCandidates: 0,
			accountChecks: false,
			browserDownloads: false,
			browserSessions: 0,
		})
	setGamesStore(games)
	setWorkspace(workspace)
	setNotificationCenter(notificationCenter)
	const accountSubpage = $derived(
		workspace.accountPage !== "profile" &&
			store.selectedAccount.id !== 0 &&
			store.vault.unlocked,
	)
	const accountToRemove = $derived(
		store.accounts.find((account) => account.id === removeAccountId) ?? null,
	)
	const accountToRenew = $derived(
		store.accounts.find((account) => account.id === renewAccountId) ?? null,
	)

	// resetAccountPages forgets what the pages read for the accounts, such as
	// after the vault locks.
	function resetAccountPages(): void {
		games.reset()
		social.reset()
		catalog.reset()
		groups.reset()
	}

	$effect(() => {
		if (!store.vault.unlocked) {
			workspace.closeAccountMenus()
			workspace.viewedImage = null
			untrack(() => {
				resetAccountPages()
				openedTabPages.clear()
			})
		} else untrack(() => games.initialize())
	})

	$effect(() => {
		const page = activePage
		if (page !== "settings" && store.vault.unlocked) {
			untrack(() => openedTabPages.add(page))
		}
	})

	function isTabPageMounted(page: TabPage): boolean {
		return activePage === page || openedTabPages.has(page)
	}

	function showLoadError(id: string, title: string, error: unknown): void {
		notificationCenter.show({
			id,
			title,
			tone: "error",
			message: errorMessage(error, "Restart the app and try again."),
		})
	}

	async function loadBuild(): Promise<void> {
		try {
			build = await accountBackend.GetBuild()
		} catch (error) {
			showLoadError("build-load", "Build details unavailable", error)
		} finally {
			developmentWarning = build && !build.release ? "open" : "closed"
		}
	}

	async function loadAppLocation(): Promise<void> {
		let location: AppLocation
		try {
			location = await accountBackend.GetAppLocation()
		} catch (error) {
			showLoadError("location-load", "Data folder unavailable", error)
			return
		}
		appLocation = location
		if (location.moveError) {
			notificationCenter.show({
				title: location.moved ? "Old data not removed" : "Data not moved",
				tone: location.moved ? "warning" : "error",
				message: location.moveError,
				durationMs: 0,
			})
		} else if (location.moved) {
			notificationCenter.show({
				title: "Data moved",
				tone: "success",
				message: "The vault, backups, and settings are in the new folder.",
			})
		}
	}

	onMount(() => {
		// The page itself handles failure when opened.
		void Promise.allSettled(
			Object.values(pageImports).map((importPage) => importPage()),
		)
		void loadBuild()
		void loadAppLocation()
		void appSettings.initialize(notificationCenter)
		const unmountAccounts = store.mount(),
			unmountBrowser = browserStore.mount(notificationCenter),
			unmountUpdates = updateStore.mount(notificationCenter)
		return () => {
			resetAccountPages()
			unmountAccounts()
			unmountBrowser()
			unmountUpdates()
			notificationCenter.dispose()
		}
	})

	// The other pages work with one account, so a multiple selection ends when
	// the Accounts page closes.
	$effect(() => {
		if (activePage !== "accounts") {
			untrack(() => store.reduceSelectionToTopAccount())
		}
	})

	$effect(() => {
		const presence = appSettings.presence.accounts,
			intervalMs = presence.intervalSeconds * 1000
		if (
			activePage !== "accounts" ||
			!appSettings.initialized ||
			!store.vault.unlocked ||
			!presence.enabled
		) {
			return
		}
		untrack(() => {
			if (Date.now() - store.presencesRefreshedAt >= intervalMs) {
				void store.refreshPresences()
			}
		})
		const timer = window.setInterval(
			() => void store.refreshPresences(),
			intervalMs,
		)
		return () => window.clearInterval(timer)
	})

	function openAddAccount(): void {
		store.clearError()
		workspace.closeAccountMenus()
		activeDialog = "add-account"
	}

	async function openImportFile(): Promise<void> {
		store.clearError()
		workspace.closeAccountMenus()
		importPath = await chooseExportFile(notificationCenter)
		if (importPath) activeDialog = "import-vault"
	}

	function openManageTags(accountId: number | null = null): void {
		store.clearError()
		workspace.closeAccountMenus()
		tagAccountId = accountId
		activeDialog = "manage-tags"
	}

	function openRemoveAccount(accountId: number): void {
		store.clearError()
		workspace.closeAccountMenus()
		store.selectAccount(accountId)
		removeAccountId = accountId
		activeDialog = "remove-account"
	}

	// showPage changes the page once a page with unsaved changes lets it close,
	// and then runs onShown.
	function showPage(page: ActivePage, onShown?: () => void): void {
		if (page === activePage) {
			onShown?.()
			return
		}
		unsavedChangesGuard.leave(() => {
			activePage = page
			onShown?.()
		})
	}

	async function focusAfterRender(selector: string): Promise<void> {
		await tick()
		document.querySelector<HTMLInputElement>(selector)?.focus()
	}

	function openSettings(): void {
		store.clearError()
		workspace.closeAccountMenus()
		const returnPage = activePage
		showPage("settings", () => {
			if (returnPage !== "settings") settingsReturnPage = returnPage
		})
	}

	function showLaunchDrawer(focusSelector: string): void {
		showPage("accounts", () => {
			workspace.launchOpen = true
			void focusAfterRender(focusSelector)
		})
	}

	function fillLaunchOptions(server: { placeId: string; jobId: string }): void {
		store.launchInput.method = LaunchMethod.MethodPlace
		store.launchInput.placeId = server.placeId
		store.launchInput.jobId = server.jobId
		showLaunchDrawer("[data-place-input]")
	}

	function fillGameLaunchOptions(placeId: number, jobId = ""): void {
		store.launchInput.launchData = ""
		fillLaunchOptions({ placeId: String(placeId), jobId })
	}

	// Joining the user who sent an invite follows them into their server.
	function fillFollowUserLaunchOptions(user: RobloxNotificationUser): void {
		store.launchInput.method = LaunchMethod.MethodUser
		store.launchInput.user = user.username || `id:${user.userId}`
		store.launchInput.resolveCurrentGame = false
		showLaunchDrawer("[data-launch-user]")
	}

	function fillGameLaunchLink(link: string): void {
		store.launchInput.method = LaunchMethod.MethodLink
		store.launchInput.link = link
		store.launchInput.launchData = ""
		showLaunchDrawer("[data-launch-link]")
	}

	function onShowGamePass(place: GamePlace, gamePassId: number): void {
		showPage("games", () => games.showGamePass(place, gamePassId))
	}

	function searchGames(text: string): void {
		showPage("games", () => {
			void games.searchFor(text)
			void focusAfterRender("[data-games-search]")
		})
	}

	function searchPlace(placeId: number): void {
		searchGames(`id:${placeId}`)
	}

	function searchUniverse(universeId: number): void {
		searchGames(`universe:${universeId}`)
	}

	// The creations of a user or a group open in the Marketplace with the items
	// that the creator hides.
	function showUserCreations(username: string): void {
		showPage("catalog", () =>
			catalog.showCreations(username, CatalogCreatorType.CatalogCreatorUser),
		)
	}

	function showGroupCreations(groupName: string): void {
		showPage("catalog", () =>
			catalog.showCreations(groupName, CatalogCreatorType.CatalogCreatorGroup),
		)
	}

	setCreatorNavigation({
		showGroup: (groupId) => showPage("groups", () => groups.showGroup(groupId)),
		showUser: (user) => {
			const account = store.selectedAccount
			showPage("social", () =>
				social.showUser(account.id, Number(account.userId) || 0, user),
			)
		},
	})

	function openRenewCookie(accountId: number): void {
		if (store.busy) return
		store.clearError()
		workspace.closeAccountMenus()
		store.selectAccount(accountId)
		renewAccountId = accountId
		activeDialog = "renew-cookie"
	}

	function openDescriptionEditor(description: string): void {
		if (store.busy || !store.vault.unlocked) return
		workspace.closeAccountMenus()
		descriptionEditor = {
			account: store.selectedAccount,
			state: store.getAccountProfile(store.selectedAccount.id),
			description,
		}
		activeDialog = "edit-description"
	}

	function closeDescriptionEditor(): void {
		activeDialog = null
		descriptionEditor = null
	}

	async function openManagedBrowser(
		accountId: number | null,
		pageUrl = "https://www.roblox.com/home",
	): Promise<void> {
		workspace.closeAccountMenus()
		await browserStore.refresh()
		if (!browserStore.runtimeReady) {
			pendingBrowserOpening = { accountId, pageUrl }
			activeDialog = "browser-runtime"
			return
		}
		if (accountId === null) activeDialog = "browser-import"
		else await browserStore.openAccount(accountId, pageUrl)
	}

	function browserRuntimeReady(): void {
		const opening = pendingBrowserOpening
		pendingBrowserOpening = undefined
		activeDialog = null
		if (opening) void openManagedBrowser(opening.accountId, opening.pageUrl)
	}

	function dismissBrowserRuntime(): void {
		if (!browserStore.runtimeDownloadActive) {
			pendingBrowserOpening = undefined
		}
		activeDialog = null
	}

	// requestLockVault asks before locking discards unsaved changes, because
	// locking closes every page.
	function requestLockVault(): void {
		unsavedChangesGuard.leave(() => void openLockVault())
	}

	async function openLockVault(): Promise<void> {
		try {
			lockEffects = await browserStore.effects()
		} catch (error) {
			showLoadError("lock-effects-load", "Vault not locked", error)
			return
		}
		activeDialog = "lock-vault"
	}

	$effect(() => {
		if (browserStore.runtimeReady && pendingBrowserOpening) {
			browserRuntimeReady()
		}
	})

	$effect(() => {
		const runtime = browserStore.snapshot.runtime
		if (
			activeDialog !== "browser-runtime" &&
			pendingBrowserOpening &&
			!browserStore.runtimeDownloadActive &&
			runtime.status === RuntimeStatus.RuntimeMissing &&
			!runtime.error
		) {
			pendingBrowserOpening = undefined
		}
	})

	$effect(() => {
		const reminderDue =
			store.vault.passwordReminderDue &&
			store.vault.unlocked &&
			store.vault.fileState === FileState.FileStateReady
		if (reminderDue && !passwordReminderNotified) {
			passwordReminderNotified = true
			notificationCenter.show({
				actions: [
					{ emphasis: true, label: "Test", onClick: openPasswordTest },
					{ label: "Ignore", onClick: ignorePasswordReminder },
				],
				dismissible: false,
				id: passwordReminderNotificationId,
				message: "Confirm that you still know your portable vault password.",
				title: "Master password check",
			})
		} else if (!reminderDue) {
			passwordReminderNotified = false
			notificationCenter.dismiss(passwordReminderNotificationId)
		}
	})

	$effect(() => {
		if (!store.vault.unlocked && activeDialog !== null) {
			activeDialog = null
			descriptionEditor = null
			pendingBrowserOpening = undefined
		}
	})

	function openPasswordTest(): void {
		store.clearError()
		notificationCenter.dismiss(passwordReminderNotificationId)
		activeDialog = "test-password"
	}

	async function ignorePasswordReminder(): Promise<void> {
		if (await store.dismissPasswordReminder()) {
			notificationCenter.dismiss(passwordReminderNotificationId)
		}
	}

	function closePasswordTest(): void {
		if (store.busy) {
			return
		}
		activeDialog = null
		passwordReminderNotified = false
	}

	function openRelationshipAction(
		person: SocialUser,
		actionId: RelationshipActionId,
	): void {
		social.relationshipError = ""
		relationshipTarget = { person, actionId }
		activeDialog = "relationship"
	}

	function closeRelationshipAction(): void {
		if (social.pendingAction) return
		social.relationshipError = ""
		activeDialog = null
		relationshipTarget = null
	}

	function showSocial(list: SocialListKind): void {
		const account = store.selectedAccount
		social.home(account.id, Number(account.userId) || 0, list)
		activePage = "social"
	}

	function openNickname(person: SocialUser, nickname: string): void {
		social.nicknameError = ""
		nicknameTarget = { person, nickname }
		activeDialog = "nickname"
	}

	function closeNickname(): void {
		if (social.savingNickname) return
		activeDialog = null
		nicknameTarget = null
	}

	function openRemoveConnections(): void {
		social.removeSelectedError = ""
		activeDialog = "remove-connections"
	}

	function closeRemoveConnections(): void {
		if (social.removingSelected) return
		social.removeSelectedError = ""
		activeDialog = null
	}

	function closeDialog(): void {
		if (activeDialog === "relationship") {
			closeRelationshipAction()
			return
		}
		if (activeDialog === "remove-connections") {
			closeRemoveConnections()
			return
		}
		if (activeDialog === "nickname") {
			closeNickname()
			return
		}
		if (store.busy) {
			return
		}

		activeDialog = null
		tagAccountId = null
		removeAccountId = null
		renewAccountId = null
	}

	function closePasteDialog(): void {
		const summary = store.consumeImportSummary()
		closeDialog()
		if (!summary || summary.created + summary.updated === 0) return
		let message = ""
		if (summary.created > 0 && summary.updated > 0) {
			message = `Added ${summary.created} new ${summary.created === 1 ? "account" : "accounts"}, and updated ${summary.updated} saved account ${summary.updated === 1 ? "cookie" : "cookies"}.`
		} else if (summary.created > 0) {
			message = `Added ${summary.created} new ${summary.created === 1 ? "account" : "accounts"}.`
		} else {
			message = `Updated ${summary.updated} saved account ${summary.updated === 1 ? "cookie" : "cookies"}.`
		}
		notificationCenter.show({ message, title: "Accounts saved", tone: "success" })
	}

	const pageOrder = [
		"accounts",
		"social",
		"groups",
		"games",
		"catalog",
		"logs-explorer",
	] as const
	const pageLabels = {
		"accounts": "Accounts",
		"social": "Social",
		"groups": "Groups",
		"games": "Games",
		"catalog": "Marketplace",
		"logs-explorer": "Logs Explorer",
	} satisfies Record<(typeof pageOrder)[number], string>

	function cyclePage(step: 1 | -1): void {
		if (
			activeDialog !== null ||
			workspace.viewedImage !== null ||
			activePage === "settings"
		) {
			return
		}
		const index = pageOrder.indexOf(activePage)
		workspace.closeAccountMenus()
		showPage(pageOrder[(index + step + pageOrder.length) % pageOrder.length])
	}

	function hasOpenAccountMenu(): boolean {
		return (
			workspace.activeAccountMenu !== null ||
			workspace.accountContextMenu !== null ||
			workspace.imageContextMenu !== null
		)
	}

	const dialogsWithOwnEscape = new Set<ActiveDialog>([
		"browser-import",
		"edit-description",
		"import-vault",
	])

	function handleWindowKeydown(event: KeyboardEvent): void {
		if (event.ctrlKey && !event.altKey && event.key === "Tab") {
			event.preventDefault()
			cyclePage(event.shiftKey ? -1 : 1)
			return
		}
		if (event.key !== "Escape") {
			return
		}

		if (activeDialog === "test-password") {
			closePasswordTest()
		} else if (dialogsWithOwnEscape.has(activeDialog)) {
			return
		} else if (activeDialog === "browser-runtime") {
			dismissBrowserRuntime()
		} else if (activeDialog === null) {
			if (hasOpenAccountMenu() || event.defaultPrevented) {
				workspace.closeAccountMenus()
			} else if (activePage === "accounts" && store.multipleSelected) {
				store.reduceSelectionToTopAccount()
			}
		} else {
			closeDialog()
		}
	}
</script>

{#snippet pageLoading()}
	<div class="min-h-0 flex-1" role="status">
		<span class="sr-only">Loading page…</span>
	</div>
{/snippet}

{#snippet pageUnavailable()}
	<div class="empty-state min-h-0 flex-1" role="alert">
		<Icon svg={warningIcon} size={28} aria-hidden="true" />
		<h2>Page unavailable</h2>
		<p>Could not load the page. Restart the app.</p>
	</div>
{/snippet}

<svelte:head>
	<title>Roblox Account Manager{build ? `: ${build.version}` : ""}</title>
</svelte:head>

<svelte:window
	onclick={(event) => workspace.handleWindowClick(event)}
	oncontextmenu={(event) => event.preventDefault()}
	onkeydown={handleWindowKeydown} />

<div class="flex h-screen flex-col bg-canvas">
	<WindowTitleBar
		version={build?.version ?? ""}
		release={build?.release ?? true}
		notifications={notificationCenter} />

	{#if activePage === "settings"}
		{#await pageImports.settings()}
			{@render pageLoading()}
		{:then { default: SettingsPage }}
			<SettingsPage
				{store}
				browser={browserStore}
				notifications={notificationCenter}
				{workspace}
				backLabel={`Back to ${pageLabels[settingsReturnPage]}`}
				onBack={() => (activePage = settingsReturnPage)}
				onLock={requestLockVault} />
		{:catch}
			{@render pageUnavailable()}
		{/await}
	{:else}
		<PageNavigation
			{store}
			{activePage}
			onPage={(page) => {
				workspace.closeAccountMenus()
				showPage(page)
			}}
			onSettings={openSettings}
			onLock={requestLockVault} />
	{/if}

	{#if isTabPageMounted("accounts")}
		<RetainedPage isShown={activePage === "accounts"}>
			{#if activePage === "accounts" && store.error && activeDialog === null && store.vault.unlocked}
				<OperationError {store} />
			{/if}
			<main
				class="flex min-h-0 flex-1"
				aria-label="Accounts"
				style={`--sidebar-width: ${workspace.sidebarCollapsed ? 44 : workspace.sidebarWidth}px; --launch-height: ${workspace.launchHeight}px;`}>
				<AccountSidebar
					{store}
					{workspace}
					onAddAccount={openAddAccount}
					onManageTags={openManageTags}
					onBrowser={(accountId) => void openManagedBrowser(accountId)}
					onImportFile={() => void openImportFile()} />

				<div class="flex min-w-0 flex-1 flex-col overflow-hidden bg-canvas">
					<div
						class={[
							"@container min-h-0 flex-1",
							accountSubpage
								? "overflow-hidden"
								: "overflow-x-hidden overflow-y-auto",
						]}
						data-workspace-scroll>
						{#if store.multipleSelected || store.selectedAccount.id === 0}
							<AccountPlaceholder
								{store}
								{workspace}
								onAddAccount={openAddAccount}
								onBrowser={() => void openManagedBrowser(null)}
								onImportFile={() => void openImportFile()} />
						{:else}
							{#key store.selectedAccountId}
								<div
									class={[
										"animate-enter",
										accountSubpage ? "h-full" : "min-h-full",
									]}>
									<AccountProfile
										{store}
										{workspace}
										notifications={notificationCenter}
										onManageTags={openManageTags}
										onJoinServer={(placeId, jobId) =>
											fillLaunchOptions({
												placeId: String(placeId),
												jobId,
											})}
										onFollowUser={(user) =>
											fillFollowUserLaunchOptions(user)}
										onSearchPlace={searchPlace}
										{onShowGamePass}
										onSearchUniverse={searchUniverse}
										onReplaceCookie={openAddAccount}
										onEditDescription={openDescriptionEditor}
										onShowSocial={showSocial}
										onShowCreations={showUserCreations}
										onOpenBrowser={(accountId) =>
											void openManagedBrowser(accountId)} />
								</div>
							{/key}
						{/if}
					</div>
					<LaunchDrawer
						{store}
						{workspace}
						notifications={notificationCenter} />
				</div>
			</main>
			<StatusBar {store} />
		</RetainedPage>
	{/if}
	{#if isTabPageMounted("social")}
		<RetainedPage isShown={activePage === "social"}>
			{#await pageImports.social()}
				{@render pageLoading()}
			{:then { default: SocialPage }}
				<SocialPage
					store={social}
					accounts={store}
					notifications={notificationCenter}
					onFillLaunch={(placeId, jobId) =>
						fillGameLaunchOptions(placeId, jobId)}
					onSearchPlace={searchPlace}
					{onShowGamePass}
					onSearchUniverse={searchUniverse}
					onRelationshipAction={openRelationshipAction}
					onEditNickname={openNickname}
					onRemoveSelected={openRemoveConnections}
					onOpenInBrowser={(accountId, pageUrl) =>
						void openManagedBrowser(accountId, pageUrl)}
					onShowCreations={showUserCreations} />
			{:catch}
				{@render pageUnavailable()}
			{/await}
		</RetainedPage>
	{/if}
	{#if isTabPageMounted("groups")}
		<RetainedPage isShown={activePage === "groups"}>
			{#await pageImports.groups()}
				{@render pageLoading()}
			{:then { default: GroupsPage }}
				<GroupsPage
					store={groups}
					accounts={store}
					notifications={notificationCenter}
					onSearchPlace={searchPlace}
					onOpenInBrowser={(accountId, pageUrl) =>
						void openManagedBrowser(accountId, pageUrl)}
					onShowCreations={showGroupCreations} />
			{:catch}
				{@render pageUnavailable()}
			{/await}
		</RetainedPage>
	{/if}
	{#if isTabPageMounted("games")}
		<RetainedPage isShown={activePage === "games"}>
			{#await pageImports.games()}
				{@render pageLoading()}
			{:then { default: GamesPage }}
				<GamesPage
					store={games}
					accounts={store}
					notifications={notificationCenter}
					onFillLaunch={(placeId, jobId) =>
						fillGameLaunchOptions(placeId, jobId)}
					onFillLaunchLink={(link) => fillGameLaunchLink(link)}
					onOpenInBrowser={(accountId, pageUrl) =>
						void openManagedBrowser(accountId, pageUrl)} />
			{:catch}
				{@render pageUnavailable()}
			{/await}
		</RetainedPage>
	{/if}
	{#if isTabPageMounted("catalog")}
		<RetainedPage isShown={activePage === "catalog"}>
			{#await pageImports.catalog()}
				{@render pageLoading()}
			{:then { default: CatalogPage }}
				<CatalogPage
					store={catalog}
					accounts={store}
					notifications={notificationCenter}
					onSearchPlace={searchPlace} />
			{:catch}
				{@render pageUnavailable()}
			{/await}
		</RetainedPage>
	{/if}
	{#if isTabPageMounted("logs-explorer")}
		<RetainedPage isShown={activePage === "logs-explorer"}>
			{#await pageImports.logsExplorer()}
				{@render pageLoading()}
			{:then { default: LogsExplorerPage }}
				<LogsExplorerPage
					store={logsExplorer}
					accounts={store}
					onFillLaunch={(visit) => fillLaunchOptions(visit)}
					onSearchPlace={searchPlace} />
			{:catch}
				{@render pageUnavailable()}
			{/await}
		</RetainedPage>
	{/if}
	<Tooltip />
</div>

<Notifications center={notificationCenter} />

{#if workspace.accountContextMenu && store.selectedAccountIds.includes(workspace.accountContextMenu.accountId) && store.vault.unlocked}
	<AccountContextMenu
		{store}
		{workspace}
		menu={workspace.accountContextMenu}
		onManageTags={openManageTags}
		onRemove={openRemoveAccount}
		onRenewCookie={openRenewCookie}
		onBrowser={(accountId) => void openManagedBrowser(accountId)} />
{/if}

{#if workspace.imageContextMenu && store.vault.unlocked}
	<AccountImageMenu {store} {workspace} menu={workspace.imageContextMenu} />
{/if}

{#if workspace.viewedImage && store.vault.unlocked}
	{@const viewedImage = workspace.viewedImage}
	<ExpandedImageDialog
		label={viewedImage.label}
		onClose={() => (workspace.viewedImage = null)}>
		{#snippet frame()}
			<img
				class="size-full animate-pop object-contain"
				src={viewedImage.imageUrl}
				alt={viewedImage.label}
				decoding="async"
				draggable="false"
				referrerpolicy="no-referrer" />
		{/snippet}
	</ExpandedImageDialog>
{/if}

{#if developmentWarning === "open" && build}
	<DevelopmentBuildDialog
		version={build.version}
		onContinue={() => (developmentWarning = "closed")} />
{:else if developmentWarning === "closed"}
	{#if appLocation && !appLocation.initialized}
		<AppLocationDialog
			location={appLocation}
			onConfirm={() => {
				if (appLocation) appLocation = { ...appLocation, initialized: true }
				void store.initialize()
			}} />
	{:else if store.vault.fileState === FileState.FileStateLegacy}
		<LegacyVaultDialog {store} />
	{:else if !store.vault.unlocked || store.vault.fileState === FileState.FileStateIncomplete}
		<VaultDialog {store} />
	{/if}
{/if}

{#if activeDialog === "add-account"}
	<AddAccountDialog {store} onClose={closePasteDialog} />
{/if}

{#if activeDialog === "import-vault" && importPath}
	<ImportVaultDialog
		{store}
		notifications={notificationCenter}
		path={importPath}
		onClose={closeDialog} />
{/if}

{#if activeDialog === "edit-description" && descriptionEditor}
	{@const editor = descriptionEditor}
	<DescriptionEditDialog
		account={editor.account}
		description={editor.description}
		notifications={notificationCenter}
		onSaved={(description) => editor.state.setDescription(description)}
		onClose={closeDescriptionEditor} />
{/if}

{#if activeDialog === "manage-tags"}
	<ManageTagsDialog {store} accountId={tagAccountId} onClose={closeDialog} />
{/if}

{#if activeDialog === "remove-account" && accountToRemove}
	<RemoveAccountDialog {store} account={accountToRemove} onClose={closeDialog} />
{/if}

{#if activeDialog === "renew-cookie" && accountToRenew}
	<RenewCookieDialog {store} account={accountToRenew} onClose={closeDialog} />
{/if}

{#if activeDialog === "nickname" && nicknameTarget}
	<NicknameDialog
		store={social}
		account={store.selectedAccount}
		person={nicknameTarget.person}
		nickname={nicknameTarget.nickname}
		onClose={closeNickname} />
{/if}

{#if activeDialog === "relationship" && relationshipTarget}
	<RelationshipActionDialog
		store={social}
		account={store.selectedAccount}
		person={relationshipTarget.person}
		actionId={relationshipTarget.actionId}
		onClose={closeRelationshipAction} />
{/if}

{#if activeDialog === "remove-connections"}
	<RemoveConnectionsDialog
		store={social}
		account={store.selectedAccount}
		onClose={closeRemoveConnections} />
{/if}

{#if activeDialog === "test-password"}
	<TestMasterPasswordDialog {store} onClose={closePasswordTest} />
{/if}

{#if activeDialog === "browser-import"}
	<BrowserImportDialog
		browser={browserStore}
		accounts={store}
		notifications={notificationCenter}
		onClose={() => (activeDialog = null)} />
{/if}

{#if activeDialog === "browser-runtime"}
	<BrowserRuntimeDialog browser={browserStore} onDismiss={dismissBrowserRuntime} />
{/if}

{#if activeDialog === "lock-vault"}
	<LockVaultDialog
		{store}
		effects={lockEffects}
		onClose={() => (activeDialog = null)} />
{/if}

<RobloxLaunchDialog />

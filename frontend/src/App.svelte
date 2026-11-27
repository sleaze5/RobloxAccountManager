<script lang="ts">
	import warningIcon from "@phosphor-icons/core/regular/warning.svg?raw"
	import { onMount, untrack } from "svelte"
	import AccountImageMenu from "./lib/accounts/AccountImageMenu.svelte"
	import AccountContextMenu from "./lib/accounts/AccountContextMenu.svelte"
	import AccountsPage from "./lib/accounts/AccountsPage.svelte"
	import { AccountStore } from "./lib/accounts/account-store.svelte"
	import { refreshPresencesWhileShown } from "./lib/accounts/presence-refresh.svelte"
	import { CatalogCreatorType } from "./lib/backend/bridge"
	import { browserStore } from "./lib/browser/browser-store.svelte"
	import { CatalogStore } from "./lib/catalog/catalog-store.svelte"
	import { AppDialogs } from "./lib/dialogs/app-dialogs.svelte"
	import AppDialogLayer from "./lib/dialogs/AppDialogLayer.svelte"
	import RobloxLaunchDialog from "./lib/dialogs/RobloxLaunchDialog.svelte"
	import { GamesStore, setGamesStore } from "./lib/games/games-store.svelte"
	import { GroupsStore } from "./lib/groups/groups-store.svelte"
	import { AppNavigation } from "./lib/layout/app-navigation.svelte"
	import { AppStartup } from "./lib/layout/app-startup.svelte"
	import { LaunchOptionsFiller } from "./lib/layout/launch-options-filler"
	import { trackPageHistories } from "./lib/layout/page-histories.svelte"
	import PageNavigation from "./lib/layout/PageNavigation.svelte"
	import { ProfilePreviewStack } from "./lib/layout/profile-preview-stack.svelte"
	import ProfilePreviewDialog from "./lib/layout/ProfilePreviewDialog.svelte"
	import RetainedPage from "./lib/layout/RetainedPage.svelte"
	import StartupDialogs from "./lib/layout/StartupDialogs.svelte"
	import { unsavedChangesGuard } from "./lib/layout/unsaved-changes-guard"
	import {
		isTextField,
		pageHistoryStepOfKey,
		pageHistoryStepOfMouseButton,
		restrictSelectAllToFocusedText,
	} from "./lib/layout/window-keys"
	import WindowTitleBar from "./lib/layout/WindowTitleBar.svelte"
	import { setWorkspace, WorkspaceState } from "./lib/layout/workspace-state.svelte"
	import { LogsExplorerStore } from "./lib/logs-explorer/logs-explorer-store.svelte"
	import { notifyErrors } from "./lib/notifications/error-notifications.svelte"
	import Notifications from "./lib/notifications/Notifications.svelte"
	import {
		NotificationCenter,
		setNotificationCenter,
	} from "./lib/notifications/notification-center.svelte"
	import { appSettings } from "./lib/settings/settings-store.svelte"
	import ExpandedImageDialog from "./lib/shared/ExpandedImageDialog.svelte"
	import Icon from "./lib/shared/Icon.svelte"
	import { setProfilePreviews } from "./lib/shared/profile-preview"
	import { setRobloxPageOpener } from "./lib/shared/roblox-pages"
	import Tooltip from "./lib/shared/Tooltip.svelte"
	import { SocialStore } from "./lib/social/social-store.svelte"
	import type { SocialListKind } from "./lib/social/social-store.svelte"
	import { updateStore } from "./lib/updates/update-store.svelte"

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
		startup = new AppStartup(notificationCenter),
		navigation = new AppNavigation(() => store.vault.unlocked),
		dialogs = new AppDialogs(store, social, workspace, notificationCenter),
		launchOptions = new LaunchOptionsFiller(store, workspace, navigation),
		profilePreviews = new ProfilePreviewStack({
			accounts: store,
			social,
			groups,
			games,
			navigation,
			notifications: notificationCenter,
		}),
		pageHistories = trackPageHistories({
			accounts: store,
			workspace,
			social,
			groups,
			games,
			logsExplorer,
		})
	setGamesStore(games)
	setWorkspace(workspace)
	setNotificationCenter(notificationCenter)
	setProfilePreviews(profilePreviews)
	setRobloxPageOpener({
		get canOpen() {
			return store.selectedAccount.id !== 0
		},
		open: (pageUrl) =>
			void dialogs.openManagedBrowser(store.selectedAccount.id, pageUrl),
	})
	refreshPresencesWhileShown(store, () => navigation.activePage === "accounts")
	// A dialog reports its own failure, so the page reports only the errors
	// of actions that ran while no dialog was open.
	notifyErrors("Action not completed", () => {
		const message = store.error
		return untrack(
			() =>
				navigation.activePage === "accounts" &&
				dialogs.open === null &&
				store.vault.unlocked,
		)
			? message
			: ""
	})

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
				navigation.forgetOpenedTabPages()
			})
		} else untrack(() => games.initialize())
	})

	onMount(() => {
		// The page itself handles failure when opened.
		void Promise.allSettled(
			Object.values(pageImports).map((importPage) => importPage()),
		)
		startup.load()
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
		if (navigation.activePage !== "accounts") {
			untrack(() => store.reduceSelectionToTopAccount())
		}
	})

	function openSettings(): void {
		store.clearError()
		workspace.closeAccountMenus()
		navigation.showSettings()
	}

	function showSocial(list: SocialListKind): void {
		const account = store.selectedAccount
		social.home(account.id, Number(account.userId) || 0, list)
		navigation.activePage = "social"
	}

	// The creations of a user or a group open in the Marketplace with the items
	// that the creator hides.
	function showCreations(creatorName: string, creatorType: CatalogCreatorType): void {
		navigation.showPage("catalog", () =>
			catalog.showCreations(creatorName, creatorType),
		)
	}

	// Alt+Left and Alt+Right, and the back and forward buttons of a mouse, go
	// back and forward through what the page showed, while no dialog is open.
	function goThroughPageHistory(step: -1 | 1): void {
		const page = navigation.activePage
		if (page === "catalog" || page === "settings") return
		if (document.querySelector('[aria-modal="true"]')) return
		const history = pageHistories[page]
		workspace.closeAccountMenus()
		unsavedChangesGuard.leave(() => {
			void (step < 0 ? history.back() : history.forward())
		})
	}

	function showNeighborTabPage(step: 1 | -1): void {
		if (dialogs.open !== null || workspace.viewedImage !== null) return
		workspace.closeAccountMenus()
		navigation.showNeighborTabPage(step)
	}

	function closeOnEscape(event: KeyboardEvent): void {
		if (dialogs.open !== null) {
			dialogs.closeOnEscape()
		} else if (workspace.hasOpenAccountMenu || event.defaultPrevented) {
			workspace.closeAccountMenus()
		} else if (navigation.activePage === "accounts" && store.multipleSelected) {
			store.reduceSelectionToTopAccount()
		}
	}

	function handleWindowKeydown(event: KeyboardEvent): void {
		const historyStep = pageHistoryStepOfKey(event)
		if (historyStep !== 0) {
			// In a text field, the keys move the caret by words on macOS.
			if (!isTextField(event.target)) {
				event.preventDefault()
				goThroughPageHistory(historyStep)
			}
			return
		}
		if (event.ctrlKey && !event.altKey && event.key === "Tab") {
			event.preventDefault()
			showNeighborTabPage(event.shiftKey ? -1 : 1)
			return
		}
		if (restrictSelectAllToFocusedText(event)) return
		if (event.key === "Escape") closeOnEscape(event)
	}

	function handleWindowMouseUp(event: MouseEvent): void {
		const historyStep = pageHistoryStepOfMouseButton(event)
		if (historyStep === 0) return
		event.preventDefault()
		goThroughPageHistory(historyStep)
	}
</script>

{#snippet pageLoading()}
	<div class="min-h-0 flex-1" role="status">
		<span class="sr-only">Loading page…</span>
	</div>
{/snippet}

{#snippet pageUnavailable()}
	<div class="empty-state min-h-0 flex-1">
		<Icon svg={warningIcon} size={28} aria-hidden="true" />
		<h2>Page unavailable</h2>
		<p>Restart the app.</p>
	</div>
{/snippet}

<svelte:head>
	<title
		>Roblox Account Manager{startup.build
			? `: ${startup.build.version}`
			: ""}</title>
</svelte:head>

<svelte:window
	onclick={(event) => workspace.handleWindowClick(event)}
	oncontextmenu={(event) => event.preventDefault()}
	onkeydown={handleWindowKeydown}
	onmouseup={handleWindowMouseUp} />

<div class="flex h-screen flex-col bg-canvas">
	<WindowTitleBar
		version={startup.build?.version ?? ""}
		release={startup.build?.release ?? true}
		notifications={notificationCenter} />

	{#if navigation.activePage === "settings"}
		{#await pageImports.settings()}
			{@render pageLoading()}
		{:then { default: SettingsPage }}
			<SettingsPage
				{store}
				browser={browserStore}
				notifications={notificationCenter}
				{workspace}
				backLabel={navigation.settingsBackLabel}
				onBack={() => navigation.leaveSettings()}
				onLock={() => dialogs.requestLockVault()} />
		{:catch}
			{@render pageUnavailable()}
		{/await}
	{:else}
		<PageNavigation
			{store}
			activePage={navigation.activePage}
			onPage={(page) => {
				workspace.closeAccountMenus()
				navigation.showPage(page)
			}}
			onSettings={openSettings}
			onLock={() => dialogs.requestLockVault()} />
	{/if}

	{#if navigation.isTabPageMounted("accounts")}
		<RetainedPage isShown={navigation.activePage === "accounts"}>
			<AccountsPage
				{store}
				{workspace}
				notifications={notificationCenter}
				{dialogs}
				{launchOptions}
				onShowSocial={showSocial}
				onShowCreations={(username) =>
					showCreations(username, CatalogCreatorType.CatalogCreatorUser)} />
		</RetainedPage>
	{/if}
	{#if navigation.isTabPageMounted("social")}
		<RetainedPage isShown={navigation.activePage === "social"}>
			{#await pageImports.social()}
				{@render pageLoading()}
			{:then { default: SocialPage }}
				<SocialPage
					store={social}
					accounts={store}
					notifications={notificationCenter}
					onFillLaunch={(placeId, serverId) =>
						launchOptions.fillGame(placeId, serverId)}
					onFillLaunchUser={(user) => launchOptions.fillFollowUser(user)}
					onRelationshipAction={(person, actionId) =>
						dialogs.openRelationshipAction(person, actionId)}
					onEditNickname={(person, nickname) =>
						dialogs.openNickname(person, nickname)}
					onRemoveSelected={() => dialogs.openRemoveConnections()}
					onShowCreations={(username) =>
						showCreations(
							username,
							CatalogCreatorType.CatalogCreatorUser,
						)} />
			{:catch}
				{@render pageUnavailable()}
			{/await}
		</RetainedPage>
	{/if}
	{#if navigation.isTabPageMounted("groups")}
		<RetainedPage isShown={navigation.activePage === "groups"}>
			{#await pageImports.groups()}
				{@render pageLoading()}
			{:then { default: GroupsPage }}
				<GroupsPage
					store={groups}
					accounts={store}
					notifications={notificationCenter}
					onShowCreations={(groupName) =>
						showCreations(
							groupName,
							CatalogCreatorType.CatalogCreatorGroup,
						)} />
			{:catch}
				{@render pageUnavailable()}
			{/await}
		</RetainedPage>
	{/if}
	{#if navigation.isTabPageMounted("games")}
		<RetainedPage isShown={navigation.activePage === "games"}>
			{#await pageImports.games()}
				{@render pageLoading()}
			{:then { default: GamesPage }}
				<GamesPage
					store={games}
					accounts={store}
					notifications={notificationCenter}
					onFillLaunch={(placeId, serverId) =>
						launchOptions.fillGame(placeId, serverId)}
					onFillLaunchLink={(link) => launchOptions.fillLink(link)} />
			{:catch}
				{@render pageUnavailable()}
			{/await}
		</RetainedPage>
	{/if}
	{#if navigation.isTabPageMounted("catalog")}
		<RetainedPage isShown={navigation.activePage === "catalog"}>
			{#await pageImports.catalog()}
				{@render pageLoading()}
			{:then { default: CatalogPage }}
				<CatalogPage
					store={catalog}
					accounts={store}
					notifications={notificationCenter} />
			{:catch}
				{@render pageUnavailable()}
			{/await}
		</RetainedPage>
	{/if}
	{#if navigation.isTabPageMounted("logs-explorer")}
		<RetainedPage isShown={navigation.activePage === "logs-explorer"}>
			{#await pageImports.logsExplorer()}
				{@render pageLoading()}
			{:then { default: LogsExplorerPage }}
				<LogsExplorerPage
					store={logsExplorer}
					accounts={store}
					onFillLaunch={(visit) => launchOptions.fillServer(visit)} />
			{:catch}
				{@render pageUnavailable()}
			{/await}
		</RetainedPage>
	{/if}
	<Tooltip />
</div>

<ProfilePreviewDialog
	subjects={profilePreviews.subjects}
	accountId={store.selectedAccount.id}
	onBack={() => profilePreviews.back()}
	onClose={() => profilePreviews.close()}
	onOpenFull={(subject) => profilePreviews.openFullProfile(subject)}
	onRelationshipAction={(person, actionId) =>
		profilePreviews.runRelationshipAction(person, actionId)}
	onMembershipChanged={(groupId) => profilePreviews.refreshChangedGroup(groupId)} />

{#if workspace.accountContextMenu && store.selectedAccountIds.includes(workspace.accountContextMenu.accountId) && store.vault.unlocked}
	<AccountContextMenu
		{store}
		{workspace}
		menu={workspace.accountContextMenu}
		onManageTags={(accountId) => dialogs.openManageTags(accountId)}
		onRemove={(accountId) => dialogs.openRemoveAccount(accountId)}
		onRenewCookie={(accountId) => dialogs.openRenewCookie(accountId)}
		onBrowser={(accountId) => void dialogs.openManagedBrowser(accountId)} />
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

<StartupDialogs {startup} {store} />

<AppDialogLayer {dialogs} {store} {social} notifications={notificationCenter} />

<RobloxLaunchDialog />

<!-- Notifications come last, so that they stack above the lock screen, which
shares their z-index. -->
<Notifications center={notificationCenter} />

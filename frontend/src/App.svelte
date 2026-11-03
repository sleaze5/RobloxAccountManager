<script lang="ts">
	import { onMount, tick, untrack } from "svelte"
	import AccountPlaceholder from "./lib/accounts/AccountPlaceholder.svelte"
	import AccountProfile from "./lib/accounts/AccountProfile.svelte"
	import AccountImageMenu from "./lib/accounts/AccountImageMenu.svelte"
	import AccountContextMenu from "./lib/accounts/AccountContextMenu.svelte"
	import AccountSidebar from "./lib/accounts/AccountSidebar.svelte"
	import { AccountStore } from "./lib/accounts/account-store.svelte"
	import type { Account } from "./lib/accounts/account-model"
	import type { AccountProfileState } from "./lib/accounts/account-profile-state.svelte"
	import AddAccountDialog from "./lib/dialogs/AddAccountDialog.svelte"
	import AppLocationDialog from "./lib/dialogs/AppLocationDialog.svelte"
	import BrowserImportDialog from "./lib/dialogs/BrowserImportDialog.svelte"
	import ImportVaultDialog from "./lib/dialogs/ImportVaultDialog.svelte"
	import { chooseExportFile } from "./lib/dialogs/vault-transfer"
	import BrowserRuntimeDialog from "./lib/dialogs/BrowserRuntimeDialog.svelte"
	import NicknameDialog from "./lib/dialogs/NicknameDialog.svelte"
	import UnfriendDialog from "./lib/dialogs/UnfriendDialog.svelte"
	import DescriptionEditDialog from "./lib/dialogs/DescriptionEditDialog.svelte"
	import LockVaultDialog from "./lib/dialogs/LockVaultDialog.svelte"
	import ManageTagsDialog from "./lib/dialogs/ManageTagsDialog.svelte"
	import RemoveAccountDialog from "./lib/dialogs/RemoveAccountDialog.svelte"
	import RobloxLaunchDialog from "./lib/dialogs/RobloxLaunchDialog.svelte"
	import RenewCookieDialog from "./lib/dialogs/RenewCookieDialog.svelte"
	import TestMasterPasswordDialog from "./lib/dialogs/TestMasterPasswordDialog.svelte"
	import UpdateExportDialog from "./lib/dialogs/UpdateExportDialog.svelte"
	import VaultDialog from "./lib/dialogs/VaultDialog.svelte"
	import {
		accountBackend,
		FileState,
		LaunchMethod,
		RuntimeStatus,
		UpdateStatus,
	} from "./lib/backend/bridge"
	import type { AppLocation, ShutdownEffects, SocialUser } from "./lib/backend/bridge"
	import { browserStore } from "./lib/browser/browser-store.svelte"
	import SocialPage from "./lib/social/SocialPage.svelte"
	import { SocialStore } from "./lib/social/social-store.svelte"
	import type { SocialListKind } from "./lib/social/social-store.svelte"
	import LogsExplorerPage from "./lib/logs-explorer/LogsExplorerPage.svelte"
	import GamesPage from "./lib/games/GamesPage.svelte"
	import { GamesStore, setGamesStore } from "./lib/games/games-store.svelte"
	import { LogsExplorerStore } from "./lib/logs-explorer/logs-explorer-store.svelte"
	import LaunchDrawer from "./lib/layout/LaunchDrawer.svelte"
	import PageNavigation from "./lib/layout/PageNavigation.svelte"
	import StatusBar from "./lib/layout/StatusBar.svelte"
	import WindowTitleBar from "./lib/layout/WindowTitleBar.svelte"
	import { WorkspaceState } from "./lib/layout/workspace-state.svelte"
	import Notifications from "./lib/notifications/Notifications.svelte"
	import { NotificationCenter } from "./lib/notifications/notification-center.svelte"
	import SettingsPage from "./lib/settings/SettingsPage.svelte"
	import { appSettings } from "./lib/settings/settings-store.svelte"
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
		| "unfriend"
		| "nickname"
		| null
	type ActivePage = "accounts" | "social" | "games" | "logs-explorer" | "settings"

	const notificationCenter = new NotificationCenter(),
		store = new AccountStore(notificationCenter),
		logsExplorer = new LogsExplorerStore(),
		games = new GamesStore(notificationCenter),
		social = new SocialStore(notificationCenter),
		workspace = new WorkspaceState(),
		passwordReminderNotificationId = "password-test-reminder"
	let appLocation = $state<AppLocation | null>(null),
		activePage = $state<ActivePage>("accounts"),
		settingsReturnPage = $state<Exclude<ActivePage, "settings">>("accounts"),
		activeDialog = $state<ActiveDialog>(null),
		passwordReminderNotified = $state(false),
		tagAccountId = $state<number | null>(null),
		removeAccountId = $state<number | null>(null),
		unfriendTarget = $state<SocialUser | null>(null),
		nicknameTarget = $state<{ person: SocialUser; nickname: string } | null>(null),
		renewAccountId = $state<number | null>(null),
		importPath = $state("")
	let descriptionEditor = $state<{
		account: Account
		state: AccountProfileState
		description: string
	} | null>(null)
	let pendingBrowserAccountId = $state<number | null | undefined>(undefined),
		lockEffects = $state<ShutdownEffects>({
			accountCandidates: 0,
			accountChecks: false,
			browserDownloads: false,
			browserSessions: 0,
		})
	setGamesStore(games)
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

	$effect(() => {
		if (!store.vault.unlocked) {
			workspace.closeAccountMenus()
			untrack(() => {
				games.reset()
				social.reset()
			})
		} else untrack(() => games.initialize())
	})

	async function loadAppLocation(): Promise<void> {
		const location = await accountBackend.GetAppLocation()
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
		void loadAppLocation()
		void appSettings.initialize(notificationCenter)
		const unmountAccounts = store.mount(),
			unmountBrowser = browserStore.mount(notificationCenter),
			unmountUpdates = updateStore.mount(notificationCenter)
		return () => {
			games.reset()
			social.reset()
			unmountAccounts()
			unmountBrowser()
			unmountUpdates()
			notificationCenter.dispose()
		}
	})

	$effect(() => {
		const presence = appSettings.presence.accounts
		if (!appSettings.initialized || !store.vault.unlocked || !presence.enabled) {
			return
		}
		untrack(() => void store.refreshPresences())
		const timer = window.setInterval(
			() => void store.refreshPresences(),
			presence.intervalSeconds * 1000,
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

	function openSettings(): void {
		store.clearError()
		workspace.closeAccountMenus()
		if (activePage !== "settings") settingsReturnPage = activePage
		activePage = "settings"
	}

	async function fillLaunchOptions(server: {
		placeId: string
		jobId: string
	}): Promise<void> {
		store.launchInput.method = LaunchMethod.MethodPlace
		store.launchInput.placeId = server.placeId
		store.launchInput.jobId = server.jobId
		workspace.launchOpen = true
		activePage = "accounts"
		await tick()
		document.querySelector<HTMLInputElement>("[data-place-input]")?.focus()
	}

	async function fillGameLaunchOptions(placeId: number, jobId = ""): Promise<void> {
		store.launchInput.method = LaunchMethod.MethodPlace
		store.launchInput.placeId = String(placeId)
		store.launchInput.jobId = jobId
		store.launchInput.launchData = ""
		workspace.launchOpen = true
		activePage = "accounts"
		await tick()
		document.querySelector<HTMLInputElement>("[data-place-input]")?.focus()
	}

	async function searchGames(text: string): Promise<void> {
		void games.searchFor(text)
		activePage = "games"
		await tick()
		document.querySelector<HTMLInputElement>("[data-games-search]")?.focus()
	}

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

	async function openManagedBrowser(accountId: number | null): Promise<void> {
		workspace.closeAccountMenus()
		await browserStore.refresh()
		if (!browserStore.runtimeReady) {
			pendingBrowserAccountId = accountId
			activeDialog = "browser-runtime"
			return
		}
		if (accountId === null) activeDialog = "browser-import"
		else await browserStore.openAccount(accountId)
	}

	function browserRuntimeReady(): void {
		const accountId = pendingBrowserAccountId
		pendingBrowserAccountId = undefined
		activeDialog = null
		if (accountId !== undefined) void openManagedBrowser(accountId)
	}

	function dismissBrowserRuntime(): void {
		if (!browserStore.runtimeDownloadActive) {
			pendingBrowserAccountId = undefined
		}
		activeDialog = null
	}

	async function openLockVault(): Promise<void> {
		lockEffects = await browserStore.effects()
		activeDialog = "lock-vault"
	}

	$effect(() => {
		if (browserStore.runtimeReady && pendingBrowserAccountId !== undefined) {
			browserRuntimeReady()
		}
	})

	$effect(() => {
		const runtime = browserStore.snapshot.runtime
		if (
			activeDialog !== "browser-runtime" &&
			pendingBrowserAccountId !== undefined &&
			!browserStore.runtimeDownloadActive &&
			runtime.status === RuntimeStatus.RuntimeMissing &&
			!runtime.error
		) {
			pendingBrowserAccountId = undefined
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
			pendingBrowserAccountId = undefined
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

	function openUnfriend(friend: SocialUser): void {
		social.unfriendError = ""
		unfriendTarget = friend
		activeDialog = "unfriend"
	}

	function showSocial(list: SocialListKind): void {
		const account = store.selectedAccount
		social.home(account.id, Number(account.userId) || 0, list)
		activePage = "social"
	}

	function closeUnfriend(): void {
		if (social.unfriending) return
		social.unfriendError = ""
		activeDialog = null
		unfriendTarget = null
	}

	function openNickname(person: SocialUser, nickname: string): void {
		social.nicknameError = ""
		nicknameTarget = { person, nickname }
		activeDialog = "nickname"
	}

	function closeNickname(): void {
		if (social.updating === "nickname") return
		activeDialog = null
		nicknameTarget = null
	}

	function closeDialog(): void {
		if (activeDialog === "unfriend") {
			closeUnfriend()
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

	const pageOrder = ["accounts", "social", "games", "logs-explorer"] as const
	const pageLabels = {
		"accounts": "Accounts",
		"social": "Social",
		"games": "Games",
		"logs-explorer": "Logs Explorer",
	} satisfies Record<(typeof pageOrder)[number], string>

	function cyclePage(step: 1 | -1): void {
		if (activeDialog !== null || activePage === "settings") return
		const index = pageOrder.indexOf(activePage)
		workspace.closeAccountMenus()
		activePage = pageOrder[(index + step + pageOrder.length) % pageOrder.length]
	}

	function hasOpenAccountMenu(): boolean {
		return (
			workspace.activeAccountMenu !== null ||
			workspace.accountContextMenu !== null ||
			workspace.imageContextMenu !== null
		)
	}

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
		} else if (
			activeDialog === "browser-import" ||
			activeDialog === "edit-description"
		) {
			return
		} else if (activeDialog === "browser-runtime") {
			dismissBrowserRuntime()
		} else if (activeDialog === null) {
			if (hasOpenAccountMenu() || event.defaultPrevented) {
				workspace.closeAccountMenus()
			} else if (activePage === "accounts" && store.multipleSelected) {
				store.clearSelection()
			}
		} else {
			closeDialog()
		}
	}
</script>

<svelte:head>
	<title>Roblox Account Manager: {APP_VERSION}</title>
</svelte:head>

<svelte:window
	onclick={(event) => workspace.handleWindowClick(event)}
	oncontextmenu={(event) => event.preventDefault()}
	onkeydown={handleWindowKeydown} />

<div class="flex h-screen flex-col bg-canvas">
	<WindowTitleBar
		version={APP_VERSION}
		pageTitle={activePage === "settings" ? "Settings" : undefined} />

	{#if activePage === "settings"}
		<SettingsPage
			{store}
			browser={browserStore}
			notifications={notificationCenter}
			{workspace}
			backLabel={`Back to ${pageLabels[settingsReturnPage]}`}
			onBack={() => (activePage = settingsReturnPage)}
			onLock={() => void openLockVault()} />
	{:else}
		<PageNavigation
			{store}
			{activePage}
			onPage={(page) => {
				workspace.closeAccountMenus()
				activePage = page
			}}
			onSettings={openSettings}
			onLock={() => void openLockVault()} />

		{#if activePage === "accounts" && store.error && activeDialog === null && store.vault.unlocked}
			<OperationError {store} />
		{/if}

		{#if activePage === "social"}
			<SocialPage
				store={social}
				accounts={store}
				onFillLaunch={(placeId, jobId) =>
					void fillGameLaunchOptions(placeId, jobId)}
				onSearchPlace={(placeId) => void searchGames(`id:${placeId}`)}
				onUnfriend={openUnfriend}
				onEditNickname={openNickname} />
		{:else if activePage === "games"}
			<GamesPage
				store={games}
				{workspace}
				onFillLaunch={(placeId, jobId) =>
					void fillGameLaunchOptions(placeId, jobId)} />
		{:else if activePage === "logs-explorer"}
			<LogsExplorerPage
				store={logsExplorer}
				onFillLaunch={(visit) => void fillLaunchOptions(visit)}
				onSearchPlace={(placeId) => void searchGames(`id:${placeId}`)} />
		{:else}
			<div
				class="flex min-h-0 flex-1"
				style={`--sidebar-width: ${workspace.sidebarCollapsed ? 44 : workspace.sidebarWidth}px; --launch-height: ${workspace.launchHeight}px;`}>
				<AccountSidebar
					{store}
					{workspace}
					onAddAccount={openAddAccount}
					onManageTags={openManageTags}
					onBrowser={(accountId) => void openManagedBrowser(accountId)}
					onImportFile={() => void openImportFile()} />

				<main class="flex min-w-0 flex-1 flex-col overflow-hidden bg-canvas">
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
											void fillLaunchOptions({
												placeId: String(placeId),
												jobId,
											})}
										onSearchPlace={(placeId) =>
											void searchGames(`id:${placeId}`)}
										onReplaceCookie={openAddAccount}
										onEditDescription={openDescriptionEditor}
										onShowSocial={showSocial} />
								</div>
							{/key}
						{/if}
					</div>
					<LaunchDrawer {store} {workspace} />
				</main>
			</div>
		{/if}

		{#if activePage === "accounts"}
			<StatusBar {store} />
		{/if}
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

{#if appLocation && !appLocation.initialized}
	<AppLocationDialog
		location={appLocation}
		onConfirm={() => {
			if (appLocation) appLocation = { ...appLocation, initialized: true }
			void store.initialize()
		}} />
{:else if !store.vault.unlocked || store.vault.fileState === FileState.FileStateIncomplete}
	<VaultDialog {store} />
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

{#if activeDialog === "unfriend" && unfriendTarget}
	<UnfriendDialog
		store={social}
		account={store.selectedAccount}
		friend={unfriendTarget}
		onClose={closeUnfriend} />
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

{#if updateStore.reviewing && updateStore.state.status === UpdateStatus.StatusAvailable}
	<UpdateExportDialog
		{store}
		notifications={notificationCenter}
		onClose={() => (updateStore.reviewing = false)} />
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

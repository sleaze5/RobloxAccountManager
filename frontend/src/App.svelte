<script lang="ts">
	import { onMount, tick, untrack } from "svelte"
	import AccountProfile from "./lib/accounts/AccountProfile.svelte"
	import AccountImageMenu from "./lib/accounts/AccountImageMenu.svelte"
	import AccountContextMenu from "./lib/accounts/AccountContextMenu.svelte"
	import AccountSidebar from "./lib/accounts/AccountSidebar.svelte"
	import { AccountStore } from "./lib/accounts/account-store.svelte"
	import AddAccountDialog from "./lib/dialogs/AddAccountDialog.svelte"
	import AppLocationDialog from "./lib/dialogs/AppLocationDialog.svelte"
	import BrowserImportDialog from "./lib/dialogs/BrowserImportDialog.svelte"
	import BrowserRuntimeDialog from "./lib/dialogs/BrowserRuntimeDialog.svelte"
	import LockVaultDialog from "./lib/dialogs/LockVaultDialog.svelte"
	import ManageTagsDialog from "./lib/dialogs/ManageTagsDialog.svelte"
	import RemoveAccountDialog from "./lib/dialogs/RemoveAccountDialog.svelte"
	import RobloxLaunchDialog from "./lib/dialogs/RobloxLaunchDialog.svelte"
	import RenewCookieDialog from "./lib/dialogs/RenewCookieDialog.svelte"
	import TestMasterPasswordDialog from "./lib/dialogs/TestMasterPasswordDialog.svelte"
	import VaultDialog from "./lib/dialogs/VaultDialog.svelte"
	import {
		accountBackend,
		FileState,
		LaunchMethod,
		RuntimeStatus,
	} from "./lib/backend/bridge"
	import type { AppLocation, ShutdownEffects } from "./lib/backend/bridge"
	import { browserStore } from "./lib/browser/browser-store.svelte"
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
		| "browser-import"
		| "browser-runtime"
		| "lock-vault"
		| "manage-tags"
		| "remove-account"
		| "renew-cookie"
		| "test-password"
		| null
	type ActivePage = "accounts" | "games" | "logs-explorer" | "settings"

	const notificationCenter = new NotificationCenter(),
		store = new AccountStore(notificationCenter),
		logsExplorer = new LogsExplorerStore(),
		games = new GamesStore(notificationCenter),
		workspace = new WorkspaceState(),
		passwordReminderNotificationId = "password-test-reminder"
	let appLocation = $state<AppLocation | null>(null),
		activePage = $state<ActivePage>("accounts"),
		settingsReturnPage = $state<Exclude<ActivePage, "settings">>("accounts"),
		activeDialog = $state<ActiveDialog>(null),
		passwordReminderNotified = $state(false),
		tagAccountId = $state<number | null>(null),
		removeAccountId = $state<number | null>(null),
		renewAccountId = $state<number | null>(null)
	let pendingBrowserAccountId = $state<number | null | undefined>(undefined),
		lockEffects = $state<ShutdownEffects>({
			accountCandidates: 0,
			accountChecks: false,
			browserDownloads: false,
			browserSessions: 0,
		})
	setGamesStore(games)
	const accountToRemove = $derived(
		store.accounts.find((account) => account.id === removeAccountId) ?? null,
	)
	const accountToRenew = $derived(
		store.accounts.find((account) => account.id === renewAccountId) ?? null,
	)

	$effect(() => {
		if (!store.vault.unlocked) {
			workspace.closeAccountMenus()
			untrack(() => games.reset())
		} else untrack(() => games.initialize())
	})

	onMount(() => {
		void accountBackend
			.GetAppLocation()
			.then((location) => (appLocation = location))
		void appSettings.initialize(notificationCenter)
		const unmountAccounts = store.mount(),
			unmountBrowser = browserStore.mount(notificationCenter),
			unmountUpdates = updateStore.mount(notificationCenter)
		return () => {
			games.reset()
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
		document.querySelector<HTMLInputElement>(".launch-place input")?.focus()
	}

	async function fillGameLaunchOptions(placeId: number, jobId = ""): Promise<void> {
		store.launchInput.method = LaunchMethod.MethodPlace
		store.launchInput.placeId = String(placeId)
		store.launchInput.jobId = jobId
		store.launchInput.launchData = ""
		workspace.launchOpen = true
		activePage = "accounts"
		await tick()
		document.querySelector<HTMLInputElement>(".launch-place input")?.focus()
	}

	function openRenewCookie(accountId: number): void {
		if (store.busy) return
		store.clearError()
		workspace.closeAccountMenus()
		store.selectAccount(accountId)
		renewAccountId = accountId
		activeDialog = "renew-cookie"
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

	function closeDialog(): void {
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
		notificationCenter.show({ message, title: "Accounts saved" })
	}

	const pageOrder = ["accounts", "games", "logs-explorer"] as const

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
		} else if (activeDialog === "browser-import") {
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

<div
	class:settings-open={activePage === "settings"}
	class:logs-explorer-open={activePage === "logs-explorer"}
	class:games-open={activePage === "games"}
	class="app-frame">
	<WindowTitleBar
		version={APP_VERSION}
		pageTitle={activePage === "settings" ? "Settings" : undefined} />

	{#if activePage === "settings"}
		<SettingsPage
			{store}
			browser={browserStore}
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

		{#if activePage === "games"}
			<GamesPage
				store={games}
				onFillLaunch={(placeId, jobId) =>
					void fillGameLaunchOptions(placeId, jobId)} />
		{:else if activePage === "logs-explorer"}
			<LogsExplorerPage
				store={logsExplorer}
				onFillLaunch={(visit) => void fillLaunchOptions(visit)} />
		{:else}
			<div
				class:sidebar-collapsed={workspace.sidebarCollapsed}
				class="workbench"
				style={`--sidebar-width: ${workspace.sidebarCollapsed ? 42 : workspace.sidebarWidth}px; --launch-height: ${workspace.launchHeight}px;`}>
				<AccountSidebar
					{store}
					{workspace}
					onAddAccount={openAddAccount}
					onManageTags={openManageTags}
					onBrowser={(accountId) => void openManagedBrowser(accountId)} />

				<main class="account-workspace">
					<div class="workspace-scroll">
						{#if !store.multipleSelected}
							{#key store.selectedAccountId}
								<div class="account-content">
									<AccountProfile
										{store}
										{workspace}
										notifications={notificationCenter}
										onManageTags={openManageTags}
										onJoinServer={(placeId, jobId) =>
											void fillLaunchOptions({
												placeId: String(placeId),
												jobId,
											})} />
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
		}} />
{:else if !store.vault.unlocked || store.vault.fileState === FileState.FileStateIncomplete}
	<VaultDialog {store} />
{/if}

{#if activeDialog === "add-account"}
	<AddAccountDialog {store} onClose={closePasteDialog} />
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

<script lang="ts">
	import type { AppDialogs } from "../dialogs/app-dialogs.svelte"
	import LaunchDrawer from "../layout/LaunchDrawer.svelte"
	import type { LaunchOptionsFiller } from "../layout/launch-options-filler"
	import StatusBar from "../layout/StatusBar.svelte"
	import type { WorkspaceState } from "../layout/workspace-state.svelte"
	import type { NotificationCenter } from "../notifications/notification-center.svelte"
	import type { SocialListKind } from "../social/social-store.svelte"
	import AccountPlaceholder from "./AccountPlaceholder.svelte"
	import AccountProfile from "./AccountProfile.svelte"
	import AccountSidebar from "./AccountSidebar.svelte"
	import type { AccountStore } from "./account-store.svelte"

	let {
		store,
		workspace,
		notifications,
		dialogs,
		launchOptions,
		onShowSocial,
		onShowCreations,
	}: {
		store: AccountStore
		workspace: WorkspaceState
		notifications: NotificationCenter
		dialogs: AppDialogs
		launchOptions: LaunchOptionsFiller
		onShowSocial: (list: SocialListKind) => void
		onShowCreations: (username: string) => void
	} = $props()

	const accountSubpage = $derived(
		workspace.accountPage !== "profile" &&
			store.selectedAccount.id !== 0 &&
			store.vault.unlocked,
	)
</script>

<main
	class="flex min-h-0 flex-1"
	aria-label="Accounts"
	style={`--sidebar-width: ${workspace.sidebarCollapsed ? 44 : workspace.sidebarWidth}px;`}>
	<AccountSidebar
		{store}
		{workspace}
		onAddAccount={() => dialogs.openAddAccount()}
		onManageTags={() => dialogs.openManageTags()}
		onBrowser={(accountId) => void dialogs.openManagedBrowser(accountId)}
		onImportFile={() => void dialogs.openImportFile()} />

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
					onAddAccount={() => dialogs.openAddAccount()}
					onBrowser={() => void dialogs.openManagedBrowser(null)}
					onImportFile={() => void dialogs.openImportFile()} />
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
							{notifications}
							onManageTags={(accountId) =>
								dialogs.openManageTags(accountId)}
							onJoinServer={(placeId, serverId) =>
								launchOptions.fillServer({
									placeId: String(placeId),
									serverId,
								})}
							onFollowUser={(user) => launchOptions.fillFollowUser(user)}
							onReplaceCookie={() => dialogs.openAddAccount()}
							onEditDescription={(description) =>
								dialogs.openDescriptionEditor(description)}
							{onShowSocial}
							{onShowCreations}
							onOpenBrowser={(accountId) =>
								void dialogs.openManagedBrowser(accountId)} />
					</div>
				{/key}
			{/if}
		</div>
		<LaunchDrawer {store} {workspace} {notifications} />
	</div>
</main>
<StatusBar {store} />

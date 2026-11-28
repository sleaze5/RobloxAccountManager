<script lang="ts">
	import type { AccountStore } from "../accounts/account-store.svelte"
	import { browserStore } from "../browser/browser-store.svelte"
	import type { NotificationCenter } from "../notifications/notification-center.svelte"
	import type { SocialStore } from "../social/social-store.svelte"
	import AddAccountDialog from "./AddAccountDialog.svelte"
	import type { AppDialogs } from "./app-dialogs.svelte"
	import BrowserImportDialog from "./BrowserImportDialog.svelte"
	import BrowserRuntimeDialog from "./BrowserRuntimeDialog.svelte"
	import DescriptionEditDialog from "./DescriptionEditDialog.svelte"
	import ImportVaultDialog from "./ImportVaultDialog.svelte"
	import LockVaultDialog from "./LockVaultDialog.svelte"
	import ManageTagsDialog from "./ManageTagsDialog.svelte"
	import NicknameDialog from "./NicknameDialog.svelte"
	import RelationshipActionDialog from "./RelationshipActionDialog.svelte"
	import RemoveAccountDialog from "./RemoveAccountDialog.svelte"
	import RemoveConnectionsDialog from "./RemoveConnectionsDialog.svelte"
	import RenewCookieDialog from "./RenewCookieDialog.svelte"
	import CheckMasterPasswordDialog from "./CheckMasterPasswordDialog.svelte"

	let {
		dialogs,
		store,
		social,
		notifications,
	}: {
		dialogs: AppDialogs
		store: AccountStore
		social: SocialStore
		notifications: NotificationCenter
	} = $props()

	const dialog = $derived(dialogs.open)
</script>

{#if dialog?.kind === "add-account"}
	<AddAccountDialog {store} onClose={() => dialogs.closeAddAccount()} />
{:else if dialog?.kind === "import-vault"}
	<ImportVaultDialog
		{store}
		{notifications}
		path={dialog.path}
		onClose={() => dialogs.close()} />
{:else if dialog?.kind === "edit-description"}
	{@const profile = dialog.profile}
	<DescriptionEditDialog
		account={dialog.account}
		description={dialog.description}
		{notifications}
		onSaved={(description) => profile.setDescription(description)}
		onClose={() => (dialogs.open = null)} />
{:else if dialog?.kind === "manage-tags"}
	<ManageTagsDialog
		{store}
		accountId={dialog.accountId}
		onClose={() => dialogs.close()} />
{:else if dialog?.kind === "remove-account"}
	{@const accountId = dialog.accountId}
	{@const account = store.accounts.find((saved) => saved.id === accountId)}
	{#if account}
		<RemoveAccountDialog {store} {account} onClose={() => dialogs.close()} />
	{/if}
{:else if dialog?.kind === "renew-cookie"}
	{@const accountId = dialog.accountId}
	{@const account = store.accounts.find((saved) => saved.id === accountId)}
	{#if account}
		<RenewCookieDialog {store} {account} onClose={() => dialogs.close()} />
	{/if}
{:else if dialog?.kind === "nickname"}
	<NicknameDialog
		store={social}
		account={store.selectedAccount}
		person={dialog.person}
		nickname={dialog.nickname}
		onClose={() => dialogs.closeNickname()} />
{:else if dialog?.kind === "relationship"}
	<RelationshipActionDialog
		store={social}
		person={dialog.person}
		actionId={dialog.actionId}
		onClose={() => dialogs.closeRelationshipAction()} />
{:else if dialog?.kind === "remove-connections"}
	<RemoveConnectionsDialog
		store={social}
		account={store.selectedAccount}
		onClose={() => dialogs.closeRemoveConnections()} />
{:else if dialog?.kind === "check-password"}
	<CheckMasterPasswordDialog {store} onClose={() => dialogs.closePasswordCheck()} />
{:else if dialog?.kind === "browser-import"}
	<BrowserImportDialog
		browser={browserStore}
		accounts={store}
		{notifications}
		onClose={() => (dialogs.open = null)} />
{:else if dialog?.kind === "browser-runtime"}
	<BrowserRuntimeDialog
		browser={browserStore}
		onDismiss={() => dialogs.dismissBrowserRuntime()} />
{:else if dialog?.kind === "lock-vault"}
	<LockVaultDialog
		{store}
		effects={dialog.effects}
		onClose={() => (dialogs.open = null)} />
{/if}

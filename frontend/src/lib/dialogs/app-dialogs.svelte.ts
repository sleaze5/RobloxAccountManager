import type { Account } from "../accounts/account-model"
import type { AccountProfileState } from "../accounts/account-profile-state.svelte"
import type { AccountStore } from "../accounts/account-store.svelte"
import { FileState, RuntimeStatus } from "../backend/bridge"
import type { ShutdownEffects, SocialUser } from "../backend/bridge"
import { browserStore } from "../browser/browser-store.svelte"
import type { WorkspaceState } from "../layout/workspace-state.svelte"
import { unsavedChangesGuard } from "../layout/unsaved-changes-guard"
import { notifyLoadFailure } from "../notifications/error-notifications.svelte"
import type { NotificationCenter } from "../notifications/notification-center.svelte"
import type { RelationshipActionId } from "../social/relationship"
import type { SocialStore } from "../social/social-store.svelte"
import { chooseExportFile } from "./vault-transfer"

// AppDialog is the dialog that the app shows over every page, with what it
// works on.
type AppDialog =
	| { kind: "add-account" }
	| { kind: "import-vault"; path: string }
	| {
			kind: "edit-description"
			account: Account
			profile: AccountProfileState
			description: string
	  }
	| { kind: "manage-tags"; accountId: number | null }
	| { kind: "remove-account"; accountId: number }
	| { kind: "renew-cookie"; accountId: number }
	| { kind: "test-password" }
	| { kind: "relationship"; person: SocialUser; actionId: RelationshipActionId }
	| { kind: "nickname"; person: SocialUser; nickname: string }
	| { kind: "remove-connections" }
	| { kind: "browser-import" }
	| { kind: "browser-runtime" }
	| { kind: "lock-vault"; effects: ShutdownEffects }

const dialogsWithOwnEscape = new Set<AppDialog["kind"]>([
	"browser-import",
	"edit-description",
	"import-vault",
])

const passwordReminderNotificationId = "password-test-reminder"

// AppDialogs opens and closes the dialogs that the app shows over every page.
// One shows at a time, and every one closes when the vault locks.
export class AppDialogs {
	open = $state<AppDialog | null>(null)
	// The browser to open once the browser runtime is ready: a page in the
	// browser of an account, or the import browser when accountId is null.
	#pendingBrowserOpening = $state<
		{ accountId: number | null; pageUrl: string } | undefined
	>(undefined)
	#isPasswordReminderShown = $state(false)
	readonly #store: AccountStore
	readonly #social: SocialStore
	readonly #workspace: WorkspaceState
	readonly #notifications: NotificationCenter

	constructor(
		store: AccountStore,
		social: SocialStore,
		workspace: WorkspaceState,
		notifications: NotificationCenter,
	) {
		this.#store = store
		this.#social = social
		this.#workspace = workspace
		this.#notifications = notifications
		$effect(() => {
			if (browserStore.runtimeReady && this.#pendingBrowserOpening) {
				this.#openPendingBrowser()
			}
		})
		$effect(() => {
			const runtime = browserStore.snapshot.runtime
			if (
				this.open?.kind !== "browser-runtime" &&
				this.#pendingBrowserOpening &&
				!browserStore.runtimeDownloadActive &&
				runtime.status === RuntimeStatus.RuntimeMissing &&
				!runtime.error
			) {
				this.#pendingBrowserOpening = undefined
			}
		})
		$effect(() => this.#showPasswordReminderWhenDue())
		$effect(() => {
			if (!store.vault.unlocked && this.open !== null) {
				this.open = null
				this.#pendingBrowserOpening = undefined
			}
		})
	}

	openAddAccount(): void {
		this.#prepareAccountDialog()
		this.open = { kind: "add-account" }
	}

	async openImportFile(): Promise<void> {
		this.#prepareAccountDialog()
		const path = await chooseExportFile(this.#notifications)
		if (path) this.open = { kind: "import-vault", path }
	}

	openManageTags(accountId: number | null = null): void {
		this.#prepareAccountDialog()
		this.open = { kind: "manage-tags", accountId }
	}

	openRemoveAccount(accountId: number): void {
		this.#prepareAccountDialog()
		this.#store.selectAccount(accountId)
		this.open = { kind: "remove-account", accountId }
	}

	openRenewCookie(accountId: number): void {
		if (this.#store.busy) return
		this.#prepareAccountDialog()
		this.#store.selectAccount(accountId)
		this.open = { kind: "renew-cookie", accountId }
	}

	openDescriptionEditor(description: string): void {
		if (this.#store.busy || !this.#store.vault.unlocked) return
		this.#workspace.closeAccountMenus()
		const account = this.#store.selectedAccount
		this.open = {
			kind: "edit-description",
			account,
			profile: this.#store.getAccountProfile(account.id),
			description,
		}
	}

	async openManagedBrowser(
		accountId: number | null,
		pageUrl = "https://www.roblox.com/home",
	): Promise<void> {
		this.#workspace.closeAccountMenus()
		await browserStore.refresh()
		if (!browserStore.runtimeReady) {
			this.#pendingBrowserOpening = { accountId, pageUrl }
			this.open = { kind: "browser-runtime" }
			return
		}
		if (accountId === null) this.open = { kind: "browser-import" }
		else await browserStore.openAccount(accountId, pageUrl)
	}

	dismissBrowserRuntime(): void {
		if (!browserStore.runtimeDownloadActive) {
			this.#pendingBrowserOpening = undefined
		}
		this.open = null
	}

	// requestLockVault asks before locking discards unsaved changes, because
	// locking closes every page.
	requestLockVault(): void {
		unsavedChangesGuard.leave(() => void this.#openLockVault())
	}

	openPasswordTest(): void {
		this.#store.clearError()
		this.#notifications.dismiss(passwordReminderNotificationId)
		this.open = { kind: "test-password" }
	}

	closePasswordTest(): void {
		if (this.#store.busy) return
		this.open = null
		this.#isPasswordReminderShown = false
	}

	openRelationshipAction(person: SocialUser, actionId: RelationshipActionId): void {
		this.#social.relationshipError = ""
		this.open = { kind: "relationship", person, actionId }
	}

	closeRelationshipAction(): void {
		if (this.#social.pendingAction) return
		this.#social.relationshipError = ""
		this.open = null
	}

	openNickname(person: SocialUser, nickname: string): void {
		this.#social.nicknameError = ""
		this.open = { kind: "nickname", person, nickname }
	}

	closeNickname(): void {
		if (this.#social.savingNickname) return
		this.open = null
	}

	openRemoveConnections(): void {
		this.#social.removeSelectedError = ""
		this.open = { kind: "remove-connections" }
	}

	closeRemoveConnections(): void {
		if (this.#social.removingSelected) return
		this.#social.removeSelectedError = ""
		this.open = null
	}

	// close closes the open dialog unless it still works, such as while it
	// saves.
	close(): void {
		const kind = this.open?.kind
		if (kind === "relationship") this.closeRelationshipAction()
		else if (kind === "remove-connections") this.closeRemoveConnections()
		else if (kind === "nickname") this.closeNickname()
		else if (kind === "test-password") this.closePasswordTest()
		else if (!this.#store.busy) this.open = null
	}

	// closeOnEscape closes the open dialog for the Escape key, except a dialog
	// that handles the key itself.
	closeOnEscape(): void {
		const kind = this.open?.kind
		if (kind === "browser-runtime") this.dismissBrowserRuntime()
		else if (kind && !dialogsWithOwnEscape.has(kind)) this.close()
	}

	closeAddAccount(): void {
		const summary = this.#store.consumeImportSummary()
		this.close()
		if (!summary || summary.created + summary.updated === 0) return
		const accountsLabel = summary.created === 1 ? "account" : "accounts",
			cookiesLabel = summary.updated === 1 ? "cookie" : "cookies"
		let message = ""
		if (summary.created > 0 && summary.updated > 0) {
			message = `Added ${summary.created} new ${accountsLabel}, and updated ${summary.updated} saved account ${cookiesLabel}.`
		} else if (summary.created > 0) {
			message = `Added ${summary.created} new ${accountsLabel}.`
		} else {
			message = `Updated ${summary.updated} saved account ${cookiesLabel}.`
		}
		this.#notifications.show({ message, title: "Accounts saved", tone: "success" })
	}

	#prepareAccountDialog(): void {
		this.#store.clearError()
		this.#workspace.closeAccountMenus()
	}

	#openPendingBrowser(): void {
		const opening = this.#pendingBrowserOpening
		this.#pendingBrowserOpening = undefined
		this.open = null
		if (opening) void this.openManagedBrowser(opening.accountId, opening.pageUrl)
	}

	async #openLockVault(): Promise<void> {
		let effects: ShutdownEffects
		try {
			effects = await browserStore.effects()
		} catch (error) {
			notifyLoadFailure(
				this.#notifications,
				"lock-effects-load",
				"Vault not locked",
				error,
			)
			return
		}
		this.open = { kind: "lock-vault", effects }
	}

	#showPasswordReminderWhenDue(): void {
		const vault = this.#store.vault,
			isReminderDue =
				vault.passwordReminderDue &&
				vault.unlocked &&
				vault.fileState === FileState.FileStateReady
		if (isReminderDue && !this.#isPasswordReminderShown) {
			this.#isPasswordReminderShown = true
			this.#notifications.show({
				actions: [
					{
						emphasis: true,
						label: "Test",
						onClick: () => this.openPasswordTest(),
					},
					{
						label: "Ignore",
						onClick: () => void this.#ignorePasswordReminder(),
					},
				],
				dismissible: false,
				id: passwordReminderNotificationId,
				message: "Confirm that you still know your portable vault password.",
				title: "Master password check",
			})
		} else if (!isReminderDue) {
			this.#isPasswordReminderShown = false
			this.#notifications.dismiss(passwordReminderNotificationId)
		}
	}

	async #ignorePasswordReminder(): Promise<void> {
		if (await this.#store.dismissPasswordReminder()) {
			this.#notifications.dismiss(passwordReminderNotificationId)
		}
	}
}

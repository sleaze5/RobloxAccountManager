import { Events } from "@wailsio/runtime"
import {
	accountBackend,
	AccountCopyField,
	FileState,
	ImportStatus,
	LaunchMethod,
	TagKind,
} from "../backend/bridge"
import type {
	AccountCursor,
	AccountView,
	AvatarHeadshotView,
	BackupInfo,
	ImportPreview,
	LaunchInput,
	LaunchResult,
	TagView,
	UserPresence,
	VaultState,
} from "../backend/bridge"
import {
	accountFromView,
	accountWithPresence,
	emptyAccount,
	isOnlinePresence,
} from "./account-model"
import type { Account } from "./account-model"
import { AccountProfileState } from "./account-profile-state.svelte"
import { isValidTagName, tagNameError } from "./tag-name"
import { appSettings } from "../settings/settings-store.svelte"
import type { NotificationCenter } from "../notifications/notification-center.svelte"

const presenceRefreshError = "Account presence could not be refreshed."

export class AccountStore {
	accounts = $state<Account[]>([])
	tags = $state<TagView[]>([])
	selectedAccountIds = $state<number[]>([])
	query = $state("")
	selectedTagIds = $state<number[]>([])
	vault = $state<VaultState>({
		automaticUnlock: false,
		dpapiRecovery: false,
		fileState: FileState.$zero,
		initialized: false,
		lastPasswordTestedAt: "",
		passwordReminderDue: false,
		passwordTestIntervalDays: 30,
		unlocked: false,
		validAutomaticUnlock: false,
	})
	busy = $state(false)
	copying = $state(false)
	launchInput = $state<LaunchInput>(emptyLaunchInput())
	launching = $state(false)
	copyingLaunchOptions = $state(false)
	findingServer = $state(false)
	refreshing = $state(false)
	presenceRefreshing = $state(false)
	error = $state("")
	nextCursor = $state<AccountCursor | null>(null)
	loadingMore = $state(false)

	readonly #avatarUrls = new Map<number, string>()
	readonly #presences = new Map<number, UserPresence>()
	readonly #presenceVersions = new Map<number, number>()
	readonly #presenceRequests = new Map<
		number,
		ReturnType<typeof accountBackend.GetAccountPresences>
	>()
	readonly #avatarRequests = new Map<
		number,
		ReturnType<typeof accountBackend.GetAvatarHeadshots>
	>()
	#remoteDataVersion = 0
	readonly #profiles = new Map<number, AccountProfileState>()
	#refreshPromise: Promise<void> | null = null
	#refreshQueued = false
	#presenceRequest = 0
	#filterTimer = 0
	#importSummary: { created: number; updated: number } | null = null

	constructor(private readonly notifications: NotificationCenter) {}

	async copyAccountFields(
		accountIds: number[],
		field: AccountCopyField,
	): Promise<boolean> {
		const labels = {
			[AccountCopyField.CopyDisplayName]: "Display name",
			[AccountCopyField.CopyUsername]: "Username",
			[AccountCopyField.CopyUserID]: "User ID",
			[AccountCopyField.CopyCookie]: "Cookie",
			[AccountCopyField.$zero]: "Account field",
		}
		return this.copyToClipboard(
			() => accountBackend.CopyAccountFields(accountIds, field),
			accountIds.length > 1
				? `${labels[field]} copied for ${accountIds.length} accounts, one per line.`
				: `${labels[field]} copied to clipboard.`,
		)
	}

	async copyAccountImage(imageUrl: string, urlOnly: boolean): Promise<boolean> {
		return this.copyToClipboard(
			() =>
				urlOnly
					? navigator.clipboard.writeText(imageUrl)
					: navigator.clipboard.write([
							new ClipboardItem({
								"image/png": this.loadClipboardImage(imageUrl),
							}),
						]),
			urlOnly ? "Image URL copied to clipboard." : "Image copied to clipboard.",
		)
	}

	private async loadClipboardImage(imageUrl: string): Promise<Blob> {
		const response = await fetch(imageUrl, {
			credentials: "omit",
			signal: AbortSignal.timeout(15_000),
		})
		if (!response.ok) {
			throw new Error("The profile image could not be loaded. Try again.")
		}
		const image = await response.blob()
		if (image.type !== "image/png") {
			throw new Error("The profile image is not a PNG.")
		}
		if (!this.vault.unlocked) throw new Error("The vault is locked.")
		return image
	}

	private async copyToClipboard(
		action: () => Promise<void>,
		message: string,
	): Promise<boolean> {
		if (this.copying || !this.vault.unlocked) return false
		this.copying = true
		this.error = ""
		try {
			await action()
			if (!this.vault.unlocked) return false
			this.notifications.show({ id: "clipboard-copy", title: "Copied", message })
			return true
		} catch (error) {
			if (this.vault.unlocked) this.error = getErrorMessage(error)
			return false
		} finally {
			this.copying = false
		}
	}

	get selectedAccountId(): number | null {
		return this.selectedAccountIds[0] ?? null
	}

	get multipleSelected(): boolean {
		return this.selectedAccountIds.length > 1
	}

	get selectedAccounts(): Account[] {
		const selected = new Set(this.selectedAccountIds)
		return this.accounts.filter((account) => selected.has(account.id))
	}

	get selectedAccount(): Account {
		return (
			this.accounts.find((account) => account.id === this.selectedAccountId) ??
			emptyAccount
		)
	}

	get favoriteTag(): TagView | null {
		return this.tags.find((tag) => tag.kind === TagKind.TagKindFavorite) ?? null
	}

	get customTags(): TagView[] {
		return this.tags.filter((tag) => tag.kind === TagKind.TagKindCustom)
	}

	get filteredAccounts(): Account[] {
		return this.accounts
	}

	get onlineCount(): number {
		return this.accounts.filter((account) =>
			isOnlinePresence(account.presence.userPresenceType),
		).length
	}

	get attentionCount(): number {
		return this.accounts.filter((account) => account.needsAttention).length
	}

	getAccountProfile(accountId: number): AccountProfileState {
		let profile = this.#profiles.get(accountId)
		if (!profile) {
			profile = new AccountProfileState(accountId, this)
			this.#profiles.set(accountId, profile)
		}
		return profile
	}

	mount(): () => void {
		const refresh = () => void this.refreshAccounts(),
			removeListeners = [
				Events.On("account:added", refresh),
				Events.On("account:updated", refresh),
				Events.On("account:removed", (event) => {
					this.#removeProfile(event.data.accountId)
					refresh()
				}),
				Events.On("account:session-state-changed", refresh),
				Events.On("tag:changed", refresh),
				Events.On("vault:locked", () => {
					this.resetLockedState()
				}),
			]

		void this.initialize()

		return () => {
			removeListeners.forEach((remove) => {
				remove()
			})
			window.clearTimeout(this.#filterTimer)
			this.resetLockedState()
		}
	}

	setQuery(value: string): void {
		this.query = value
		window.clearTimeout(this.#filterTimer)
		this.#filterTimer = window.setTimeout(() => void this.refreshAccounts(), 150)
	}

	toggleSelectedTag(tagId: number): void {
		if (tagId <= 0) {
			return
		}
		this.selectedTagIds = this.selectedTagIds.includes(tagId)
			? this.selectedTagIds.filter((selectedTagId) => selectedTagId !== tagId)
			: [...this.selectedTagIds, tagId]
		void this.refreshAccounts()
	}

	clearSelectedTags(): void {
		if (this.selectedTagIds.length === 0) {
			return
		}
		this.selectedTagIds = []
		void this.refreshAccounts()
	}

	clearError(): void {
		this.error = ""
	}

	clearJoinTarget(): void {
		Object.assign(this.launchInput, {
			method: LaunchMethod.MethodPlace,
			placeId: "",
			jobId: "",
			link: "",
			user: "",
			resolveCurrentGame: false,
		})
	}

	selectAccount(accountId: number): void {
		this.selectedAccountIds = accountId > 0 ? [accountId] : []
	}

	clearSelection(): void {
		this.selectedAccountIds = []
	}

	toggleAccount(accountId: number): void {
		this.selectedAccountIds = this.selectedAccountIds.includes(accountId)
			? this.selectedAccountIds.filter((id) => id !== accountId)
			: [...this.selectedAccountIds, accountId]
	}

	async initialize(): Promise<void> {
		try {
			this.vault = await accountBackend.GetVaultState()
			if (
				this.vault.unlocked &&
				this.vault.fileState === FileState.FileStateReady
			) {
				await this.refreshAccounts()
			}
		} catch (error) {
			this.error = getErrorMessage(error)
		}
	}

	async unlockVault(password: string, automaticUnlock: boolean): Promise<boolean> {
		if (!password || this.busy) {
			return false
		}

		this.busy = true
		this.error = ""
		try {
			await accountBackend.UnlockVault(password)
			this.vault = await accountBackend.GetVaultState()
			await this.refreshAccounts()
			await accountBackend.SetAutomaticUnlock(automaticUnlock)
			this.vault = await accountBackend.GetVaultState()
			return true
		} catch (error) {
			this.error = getErrorMessage(error)
			return false
		} finally {
			this.busy = false
		}
	}

	async createVault(
		password: string,
		confirmation: string,
		hint: string,
		automaticUnlock: boolean,
	): Promise<boolean> {
		if (this.busy) {
			return false
		}
		this.busy = true
		this.error = ""
		try {
			await accountBackend.CreateVault(
				password,
				confirmation,
				hint,
				automaticUnlock,
			)
			this.vault = await accountBackend.GetVaultState()
			await this.refreshAccounts()
			return true
		} catch (error) {
			this.error = getErrorMessage(error)
			return false
		} finally {
			this.busy = false
		}
	}

	async lockVault(): Promise<void> {
		if (this.busy || !this.vault.unlocked) {
			return
		}

		this.busy = true
		this.error = ""
		try {
			await accountBackend.LockVault()
			this.resetLockedState()
		} catch (error) {
			this.error = getErrorMessage(error)
		} finally {
			this.busy = false
		}
	}

	async testVaultPassword(password: string): Promise<boolean> {
		if (!password || this.busy) {
			return false
		}
		this.busy = true
		this.error = ""
		try {
			await accountBackend.TestVaultPassword(password)
			this.vault = await accountBackend.GetVaultState()
			return true
		} catch (error) {
			this.error = getErrorMessage(error)
			return false
		} finally {
			this.busy = false
		}
	}

	async dismissPasswordReminder(): Promise<boolean> {
		if (this.busy) {
			return false
		}
		this.busy = true
		this.error = ""
		try {
			await accountBackend.DismissPasswordReminder()
			this.vault = await accountBackend.GetVaultState()
			return true
		} catch (error) {
			this.error = getErrorMessage(error)
			return false
		} finally {
			this.busy = false
		}
	}

	async setPasswordTestIntervalDays(days: number): Promise<void> {
		if (this.busy) {
			return
		}
		this.busy = true
		this.error = ""
		try {
			await accountBackend.SetPasswordTestIntervalDays(days)
			this.vault = await accountBackend.GetVaultState()
		} catch (error) {
			this.error = getErrorMessage(error)
		} finally {
			this.busy = false
		}
	}

	async setAutomaticUnlock(enabled: boolean): Promise<void> {
		if (this.busy || !this.vault.unlocked) {
			return
		}
		this.busy = true
		this.error = ""
		try {
			await accountBackend.SetAutomaticUnlock(enabled)
			this.vault = await accountBackend.GetVaultState()
		} catch (error) {
			this.error = getErrorMessage(error)
		} finally {
			this.busy = false
		}
	}

	async changeVaultPassword(
		currentPassword: string,
		newPassword: string,
		confirmation: string,
		hint: string,
		automaticUnlock: boolean,
	): Promise<boolean> {
		if (this.busy) {
			return false
		}
		this.busy = true
		this.error = ""
		try {
			await accountBackend.ChangeVaultPassword(
				currentPassword,
				newPassword,
				confirmation,
				hint,
				automaticUnlock,
			)
			this.vault = await accountBackend.GetVaultState()
			if (this.vault.fileState === FileState.FileStateReady) {
				await this.refreshAccounts()
			}
			return true
		} catch (error) {
			this.error = getErrorMessage(error)
			return false
		} finally {
			this.busy = false
		}
	}

	async listBackups(): Promise<BackupInfo[]> {
		try {
			return (await accountBackend.ListBackups()) ?? []
		} catch (error) {
			this.error = getErrorMessage(error)
			return []
		}
	}

	async restoreBackup(name: string, password: string): Promise<boolean> {
		if (this.busy) {
			return false
		}
		this.busy = true
		this.error = ""
		try {
			await accountBackend.RestoreBackup(name, password)
			this.vault = await accountBackend.GetVaultState()
			return true
		} catch (error) {
			this.error = getErrorMessage(error)
			return false
		} finally {
			this.busy = false
		}
	}

	async resetVault(confirmation: string): Promise<boolean> {
		if (this.busy) {
			return false
		}
		this.busy = true
		this.error = ""
		try {
			await accountBackend.ResetVault(confirmation)
			this.vault = await accountBackend.GetVaultState()
			return true
		} catch (error) {
			this.error = getErrorMessage(error)
			return false
		} finally {
			this.busy = false
		}
	}

	async refreshAccountList(): Promise<void> {
		if (!this.vault.unlocked || this.refreshing) return
		this.refreshing = true
		try {
			await this.refreshAccounts()
			await this.refreshPresences()
		} finally {
			this.refreshing = false
		}
	}

	async refreshAccounts(): Promise<void> {
		if (!this.vault.unlocked) {
			return
		}
		if (this.#refreshPromise) {
			this.#refreshQueued = true
			return this.#refreshPromise
		}

		this.#refreshPromise = this.performQueuedRefresh()
		try {
			await this.#refreshPromise
		} finally {
			this.#refreshPromise = null
		}
	}

	private async performQueuedRefresh(): Promise<void> {
		this.#refreshQueued = false
		await this.performRefresh()
		if (this.#refreshQueued && this.vault.unlocked) {
			await this.performQueuedRefresh()
		}
	}

	private async performRefresh(): Promise<void> {
		const version = this.#remoteDataVersion
		try {
			const targetCount = Math.max(this.accounts.length, 10),
				tagRecords = await accountBackend.ListTags()
			if (version !== this.#remoteDataVersion || !this.vault.unlocked) return
			this.tags = [...(tagRecords ?? [])].sort(compareTags)
			const availableTagIds = new Set(this.tags.map((tag) => tag.id))
			this.selectedTagIds = this.selectedTagIds.filter((tagId) =>
				availableTagIds.has(tagId),
			)
			let cursor: AccountCursor | null = null
			const records: AccountView[] = []
			const loadPage = async (): Promise<void> => {
				const page = await accountBackend.ListAccounts({
					cursor,
					search: this.query,
					tagIds: this.selectedTagIds,
				})
				records.push(...(page.accounts ?? []))
				cursor = page.nextCursor ?? null
				if (cursor && records.length < targetCount) {
					await loadPage()
				}
			}
			await loadPage()
			if (version !== this.#remoteDataVersion || !this.vault.unlocked) return
			const selectFirst =
				this.accounts.length === 0 || this.selectedAccountIds.length > 0
			this.nextCursor = cursor
			this.accounts = records.map((record) => this.createAccount(record))
			const availableIds = new Set(this.accounts.map((account) => account.id))
			this.selectedAccountIds = this.selectedAccountIds.filter((id) =>
				availableIds.has(id),
			)
			if (selectFirst && this.selectedAccountIds.length === 0) {
				this.selectAccount(this.accounts[0]?.id ?? 0)
			}

			const userIDs = records.map((record) => record.robloxUserId)
			void this.refreshAvatarHeadshots(userIDs)
		} catch (error) {
			if (version !== this.#remoteDataVersion || !this.vault.unlocked) return
			this.error = getErrorMessage(error)
		}
	}

	async loadMore(): Promise<void> {
		if (!this.nextCursor || this.loadingMore || this.#refreshPromise) {
			return
		}
		this.loadingMore = true
		const version = this.#remoteDataVersion
		try {
			const page = await accountBackend.ListAccounts({
					cursor: this.nextCursor,
					search: this.query,
					tagIds: this.selectedTagIds,
				}),
				records = page.accounts ?? []
			if (version !== this.#remoteDataVersion || !this.vault.unlocked) return
			this.accounts = [
				...this.accounts,
				...records.map((record) => this.createAccount(record)),
			]
			this.nextCursor = page.nextCursor ?? null
			void this.refreshAvatarHeadshots(
				records.map((record) => record.robloxUserId),
			)
		} catch (error) {
			if (version !== this.#remoteDataVersion || !this.vault.unlocked) return
			this.error = getErrorMessage(error)
		} finally {
			this.loadingMore = false
		}
	}

	async moveAccount(
		accountId: number,
		targetId: number,
		afterTarget: boolean,
	): Promise<void> {
		if (this.busy || this.multipleSelected || accountId === targetId) {
			return
		}

		const previous = [...this.accounts],
			next = [...this.accounts],
			sourceIndex = next.findIndex((account) => account.id === accountId)
		if (sourceIndex < 0) {
			return
		}
		const [moved] = next.splice(sourceIndex, 1)
		let targetIndex = next.findIndex((account) => account.id === targetId)
		if (targetIndex < 0) {
			return
		}
		if (afterTarget) {
			targetIndex++
		}
		next.splice(targetIndex, 0, moved)
		if (next.every((account, index) => account.id === previous[index]?.id)) {
			return
		}

		this.accounts = next
		this.busy = true
		this.error = ""
		try {
			const movedIndex = next.findIndex((account) => account.id === accountId)
			await accountBackend.MoveAccount(
				accountId,
				next[movedIndex - 1]?.id ?? null,
				next[movedIndex + 1]?.id ?? null,
			)
			await this.refreshAccounts()
		} catch (error) {
			this.accounts = previous
			this.error = getErrorMessage(error)
		} finally {
			this.busy = false
		}
	}

	async validateCookies(cookieInput: string): Promise<ImportPreview | null> {
		if (this.busy) {
			return null
		}
		if (!cookieInput.trim()) {
			this.error = "Enter at least one .ROBLOSECURITY cookie."
			return null
		}

		this.busy = true
		this.error = ""
		try {
			const preview = await accountBackend.ValidateCookies(cookieInput),
				items = preview.items ?? []
			preview.items = items
			return preview
		} catch (error) {
			this.error = getErrorMessage(error)
			return null
		} finally {
			this.busy = false
		}
	}

	async saveValidatedCookies(
		batchId: string,
		selectedIndexes: number[],
	): Promise<boolean> {
		if (this.busy || !batchId || selectedIndexes.length < 1) {
			return false
		}

		this.busy = true
		this.error = ""
		try {
			const result = await accountBackend.SaveValidatedCookies(
					batchId,
					selectedIndexes,
				),
				items = result.items ?? [],
				saved = items.filter(
					(item) =>
						item.status === ImportStatus.ImportCreated ||
						item.status === ImportStatus.ImportUpdated,
				),
				duplicates = items.filter(
					(item) => item.status === ImportStatus.ImportDuplicate,
				),
				failed = items.filter(
					(item) => item.status === ImportStatus.ImportFailed,
				)

			if (saved.length > 0) {
				this.recordImportSummary(saved.map((item) => item.status))
				this.selectAccount(saved.at(-1)?.account.id ?? 0)
				await this.refreshAccounts()
			}

			if (saved.length === selectedIndexes.length) {
				return true
			}
			if (failed.length > 0) {
				this.error = `${failed.length} ${failed.length === 1 ? "account" : "accounts"} could not be saved. Validate the cookies again.`
			} else if (duplicates.length > 0) {
				this.error = `${duplicates.length} ${duplicates.length === 1 ? "account is" : "accounts are"} already in the vault. Their cookies were not changed.`
			}
			return false
		} catch (error) {
			this.error = getErrorMessage(error)
			return false
		} finally {
			this.busy = false
		}
	}

	private recordImportSummary(statuses: ImportStatus[]): void {
		const created = statuses.filter(
				(status) => status === ImportStatus.ImportCreated,
			).length,
			updated = statuses.filter(
				(status) => status === ImportStatus.ImportUpdated,
			).length
		this.#importSummary = {
			created: (this.#importSummary?.created ?? 0) + created,
			updated: (this.#importSummary?.updated ?? 0) + updated,
		}
	}

	consumeImportSummary(): { created: number; updated: number } | null {
		const summary = this.#importSummary
		this.#importSummary = null
		return summary
	}

	async discardValidatedCookies(batchId: string): Promise<void> {
		if (!batchId) {
			return
		}
		try {
			await accountBackend.DiscardValidatedCookies(batchId)
		} catch {}
	}

	async createTag(name: string, accountId: number | null): Promise<TagView | null> {
		if (this.busy) {
			return null
		}
		if (!isValidTagName(name)) {
			this.error = tagNameError(name)
			return null
		}

		this.busy = true
		this.error = ""
		try {
			const tag = await accountBackend.CreateTag(name)
			if (accountId !== null) {
				await accountBackend.SetAccountsTag([accountId], tag.id, true)
			}
			await this.refreshAccounts()
			return tag
		} catch (error) {
			this.error = getErrorMessage(error)
			return null
		} finally {
			this.busy = false
		}
	}

	async removeTag(tagId: number): Promise<void> {
		if (tagId <= 0 || this.busy) {
			return
		}

		this.busy = true
		this.error = ""
		try {
			await accountBackend.RemoveTag(tagId)
			this.selectedTagIds = this.selectedTagIds.filter(
				(selectedTagId) => selectedTagId !== tagId,
			)
			await this.refreshAccounts()
		} catch (error) {
			this.error = getErrorMessage(error)
		} finally {
			this.busy = false
		}
	}

	async setAccountsTag(
		accountIds: number[],
		tag: TagView,
		selected: boolean,
	): Promise<void> {
		if (accountIds.length === 0 || accountIds.includes(0) || this.busy) {
			return
		}

		this.busy = true
		this.error = ""
		try {
			await accountBackend.SetAccountsTag(accountIds, tag.id, selected)
			await this.refreshAccounts()
		} catch (error) {
			this.error = getErrorMessage(error)
		} finally {
			this.busy = false
		}
	}

	async removeAccount(accountId: number): Promise<boolean> {
		if (accountId === 0 || this.busy) {
			return false
		}

		this.busy = true
		this.error = ""
		try {
			await accountBackend.RemoveAccount(accountId)
			this.#removeProfile(accountId)
			await this.refreshAccounts()
			return true
		} catch (error) {
			this.error = getErrorMessage(error)
			return false
		} finally {
			this.busy = false
		}
	}

	async fillNearestServer(): Promise<void> {
		const placeInput = this.launchInput.placeId.trim(),
			placeId = Number(placeInput),
			accountId = this.selectedAccountId ?? 0,
			region = appSettings.roValraRegion,
			regionLabel = appSettings.preferredRegionLabel
		if (
			this.findingServer ||
			this.launching ||
			!/^\d+$/.test(placeInput) ||
			!Number.isSafeInteger(placeId) ||
			placeId <= 0
		) {
			return
		}
		this.findingServer = true
		this.error = ""
		try {
			const server = await accountBackend.FindNearestGameServer(
				placeId,
				accountId,
			)
			if (
				!this.vault.unlocked ||
				this.launchInput.placeId.trim() !== placeInput ||
				(this.selectedAccountId ?? 0) !== accountId ||
				appSettings.roValraRegion !== region
			) {
				return
			}
			this.launchInput.jobId = server.jobId
			if (server.region.code !== region) {
				this.notifications.show({
					id: "nearest-server",
					title: "Nearest server filled",
					message: `No server was found in ${regionLabel}, so a server in the nearest available region (${server.region.name}) was filled.`,
				})
			}
		} catch (error) {
			if (this.vault.unlocked) this.error = getErrorMessage(error)
		} finally {
			this.findingServer = false
		}
	}

	async launchSelectedAccounts(): Promise<void> {
		await this.performLaunchAction(false)
	}

	async copyLaunchOptions(): Promise<boolean> {
		return this.performLaunchAction(true)
	}

	private async performLaunchAction(copyOptions: boolean): Promise<boolean> {
		const selected = this.selectedAccounts,
			accountIds = selected.map((account) => account.id)
		if (accountIds.length === 0 || this.launching || !this.vault.unlocked) {
			return false
		}

		this.launching = true
		this.copyingLaunchOptions = copyOptions
		this.error = ""
		try {
			const input = { ...this.launchInput }
			if (copyOptions) await accountBackend.CopyLaunchOptions(accountIds, input)
			else {
				const result = await accountBackend.LaunchAccounts(accountIds, input)
				if (!this.vault.unlocked) return false
				return this.showLaunchResult(result, selected)
			}
			return this.vault.unlocked
		} catch (error) {
			if (this.vault.unlocked) this.error = getErrorMessage(error)
			return false
		} finally {
			this.launching = false
			this.copyingLaunchOptions = false
		}
	}

	private showLaunchResult(result: LaunchResult, selected: Account[]): boolean {
		const failures = result.failures ?? [],
			launched = result.launched?.length ?? 0
		this.error = failures
			.map((failure) => {
				const account = selected.find((item) => item.id === failure.accountId)
				return `${account ? `@${account.username}: ` : ""}${failure.message}`
			})
			.join("\n")
		if (selected.length > 1 && (launched > 0 || failures.length > 0)) {
			this.notifications.show({
				id: "batch-launch",
				title: result.cancelled ? "Launch stopped" : "Account launch complete",
				message: `Launch requested for ${launched} of ${selected.length} accounts.${failures.length > 0 ? ` ${failures.length} failed. See the error details.` : ""}`,
			})
		}
		return !result.cancelled && failures.length === 0
	}

	private createAccount(record: AccountView): Account {
		return accountFromView(
			record,
			this.#avatarUrls.get(record.robloxUserId) ?? "",
			this.#presences.get(record.robloxUserId),
		)
	}

	async refreshPresences(accountId: number | null = null): Promise<void> {
		if (!this.vault.unlocked) return
		const key = accountId ?? 0,
			pending = this.#presenceRequests.get(key)
		if (pending) {
			await pending.catch(() => undefined)
			return
		}
		const version = this.#remoteDataVersion,
			sequence = ++this.#presenceRequest,
			request = accountBackend.GetAccountPresences(accountId)
		this.#presenceRequests.set(key, request)
		if (accountId === null) this.presenceRefreshing = true
		try {
			const presences = (await request) ?? []
			if (version !== this.#remoteDataVersion || !this.vault.unlocked) return
			this.installPresences(presences, sequence)
			if (this.error === presenceRefreshError) {
				this.error = ""
			}
		} catch {
			if (version !== this.#remoteDataVersion || !this.vault.unlocked) return
			this.error = presenceRefreshError
		} finally {
			if (this.#presenceRequests.get(key) === request) {
				this.#presenceRequests.delete(key)
				if (accountId === null) this.presenceRefreshing = false
			}
		}
	}

	private installPresences(presences: UserPresence[], sequence: number): void {
		for (const presence of presences) {
			if ((this.#presenceVersions.get(presence.userId) ?? 0) > sequence) continue
			this.#presences.set(presence.userId, presence)
			this.#presenceVersions.set(presence.userId, sequence)
		}
		this.accounts = this.accounts.map((account) => {
			const presence = this.#presences.get(Number(account.userId))
			return presence ? accountWithPresence(account, presence) : account
		})
	}

	async refreshAccountAvatar(accountId: number): Promise<void> {
		const account = this.accounts.find((item) => item.id === accountId)
		if (!account || !this.vault.unlocked) return
		await this.refreshAvatarHeadshots([Number(account.userId)], true)
	}

	private async refreshAvatarHeadshots(
		userIds: number[],
		force = false,
	): Promise<void> {
		const version = this.#remoteDataVersion,
			requests = this.getAvatarRequests(userIds, force)
		if (requests.size === 0) return
		try {
			const headshots = (await Promise.all(requests)).flatMap(
				(result) => result ?? [],
			)
			if (version !== this.#remoteDataVersion || !this.vault.unlocked) return
			this.installHeadshots(headshots)
			if (
				force &&
				!headshots.some((headshot) => headshot.robloxUserId === userIds[0])
			) {
				throw new Error(
					"The profile image is not available yet. Refresh to retry.",
				)
			}
		} catch {
			if (force) {
				throw new Error(
					"The profile image could not be refreshed. Refresh to retry.",
				)
			}
		} finally {
			for (const userId of userIds) {
				const pending = this.#avatarRequests.get(userId)
				if (pending && requests.has(pending)) {
					this.#avatarRequests.delete(userId)
				}
			}
		}
	}

	private getAvatarRequests(
		userIds: number[],
		force: boolean,
	): Set<ReturnType<typeof accountBackend.GetAvatarHeadshots>> {
		const requests = new Set<
				ReturnType<typeof accountBackend.GetAvatarHeadshots>
			>(),
			missing: number[] = []
		for (const userId of userIds) {
			const pending = this.#avatarRequests.get(userId)
			if (pending) requests.add(pending)
			else if (force || !this.#avatarUrls.has(userId)) missing.push(userId)
		}
		if (missing.length > 0) {
			const request = accountBackend.GetAvatarHeadshots(missing)
			requests.add(request)
			for (const userId of missing) this.#avatarRequests.set(userId, request)
		}
		return requests
	}

	private installHeadshots(headshots: AvatarHeadshotView[]): void {
		for (const headshot of headshots) {
			this.#avatarUrls.set(headshot.robloxUserId, headshot.imageUrl)
		}

		this.accounts = this.accounts.map((account) => ({
			...account,
			avatarUrl:
				this.#avatarUrls.get(Number(account.userId)) ?? account.avatarUrl,
		}))
	}

	private resetLockedState(): void {
		this.#remoteDataVersion++
		this.vault.unlocked = false
		this.launchInput = emptyLaunchInput()
		this.#clearProfiles()
		this.accounts = []
		this.tags = []
		this.selectedTagIds = []
		this.selectedAccountIds = []
		this.nextCursor = null
		this.#avatarUrls.clear()
		this.#presences.clear()
		this.#presenceVersions.clear()
		for (const request of new Set(this.#avatarRequests.values())) {
			void request.cancel()
		}
		this.#avatarRequests.clear()
		for (const request of this.#presenceRequests.values()) void request.cancel()
		this.#presenceRequests.clear()
		this.presenceRefreshing = false
	}

	#removeProfile(accountId: number): void {
		this.#profiles.get(accountId)?.dispose()
		this.#profiles.delete(accountId)
	}

	#clearProfiles(): void {
		for (const profile of this.#profiles.values()) profile.dispose()
		this.#profiles.clear()
	}
}

function emptyLaunchInput(): LaunchInput {
	return {
		method: LaunchMethod.MethodPlace,
		placeId: "",
		jobId: "",
		link: "",
		user: "",
		resolveCurrentGame: false,
		teleport: false,
		directLaunch: false,
		launchData: "",
	}
}

function compareTags(left: TagView, right: TagView): number {
	if (left.kind !== right.kind) {
		return left.kind === TagKind.TagKindFavorite ? -1 : 1
	}

	return left.name.localeCompare(right.name)
}

function getErrorMessage(error: unknown): string {
	return error instanceof Error ? error.message : "The operation failed."
}

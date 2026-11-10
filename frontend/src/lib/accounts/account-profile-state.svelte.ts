import {
	AccountProfileSection,
	AgeVerification,
	accountBackend,
} from "../backend/bridge"
import type { AccountProfileSnapshot } from "../backend/bridge"
import { isFresh } from "../shared/freshness"
import type { AccountStore } from "./account-store.svelte"
import { errorMessage } from "../shared/errors"

// Sections load in this order, so that the header fills first and the
// paginated sections, which take longest, come last. Roblox requests for one
// account run a few at a time.
const profileSections: {
	section: AccountProfileSection
	fields: (keyof AccountProfileSnapshot)[]
}[] = [
	{ section: AccountProfileSection.ProfileSectionRobux, fields: ["robux"] },
	{
		section: AccountProfileSection.ProfileSectionAccount,
		fields: ["ageBracket", "countryCode", "premium", "plus"],
	},
	{
		section: AccountProfileSection.ProfileSectionAbout,
		fields: ["description", "verifiedBadge"],
	},
	{ section: AccountProfileSection.ProfileSectionFriends, fields: ["friendCount"] },
	{
		section: AccountProfileSection.ProfileSectionFollowers,
		fields: ["followerCount"],
	},
	{
		section: AccountProfileSection.ProfileSectionFollowing,
		fields: ["followingCount"],
	},
	{
		section: AccountProfileSection.ProfileSectionTransactions,
		fields: ["pendingRobux", "incomingRobuxPastMonth", "outgoingRobuxPastMonth"],
	},
	{
		section: AccountProfileSection.ProfileSectionTwoStep,
		fields: ["twoStepEnabled", "twoStepMethods"],
	},
	{
		section: AccountProfileSection.ProfileSectionAge,
		fields: ["ageGroup", "ageVerification"],
	},
	{ section: AccountProfileSection.ProfileSectionVoiceChat, fields: ["voiceChat"] },
	{
		section: AccountProfileSection.ProfileSectionPrimaryGroup,
		fields: ["primaryGroup"],
	},
	{
		section: AccountProfileSection.ProfileSectionCommunities,
		fields: ["communityCount"],
	},
	{
		section: AccountProfileSection.ProfileSectionSocialLinks,
		fields: ["socialLinks"],
	},
	{
		section: AccountProfileSection.ProfileSectionRobloxBadges,
		fields: ["robloxBadges"],
	},
	{
		section: AccountProfileSection.ProfileSectionActivity,
		fields: ["unreadMessageCount", "inboundTradeCount", "unreadNotificationCount"],
	},
	{
		section: AccountProfileSection.ProfileSectionPreviousNames,
		fields: ["previousNames"],
	},
	{
		section: AccountProfileSection.ProfileSectionCollectibles,
		fields: ["collectibleCount", "collectibleRecentAveragePriceTotal"],
	},
]

function emptyProfile(accountId: number): AccountProfileSnapshot {
	return {
		accountId,
		section: AccountProfileSection.$zero,
		isComplete: false,
		robux: null,
		pendingRobux: null,
		incomingRobuxPastMonth: null,
		outgoingRobuxPastMonth: null,
		ageBracket: null,
		ageGroup: "",
		ageVerification: AgeVerification.$zero,
		countryCode: "",
		premium: null,
		plus: null,
		twoStepEnabled: null,
		twoStepMethods: null,
		voiceChat: null,
		description: "",
		socialLinks: null,
		previousNames: null,
		verifiedBadge: null,
		friendCount: null,
		followerCount: null,
		followingCount: null,
		primaryGroup: null,
		communityCount: null,
		robloxBadges: null,
		collectibleCount: null,
		collectibleRecentAveragePriceTotal: null,
		unreadMessageCount: null,
		inboundTradeCount: null,
		unreadNotificationCount: null,
	}
}

// The profile stays in memory until the vault locks or the account is removed,
// so a revisit shows the last values at once while they refresh.
export class AccountProfileState {
	readonly snapshot: AccountProfileSnapshot
	loadedSections = $state<AccountProfileSection[]>([])
	loadingSections = $state<AccountProfileSection[]>([])
	failedSections = $state<AccountProfileSection[]>([])
	isValidating = $state(false)
	error = $state("")
	#active = true
	#fetchedAt = 0
	#refreshCount = 0
	readonly #requests = new Set<ReturnType<typeof accountBackend.GetAccountProfile>>()
	#validationRequest: ReturnType<typeof accountBackend.ValidateAccount> | null = null
	readonly #accountId: number
	readonly #store: AccountStore

	constructor(accountId: number, store: AccountStore) {
		this.#accountId = accountId
		this.#store = store
		this.snapshot = $state(emptyProfile(accountId))
	}

	get loading(): boolean {
		return this.loadingSections.length > 0 || this.isValidating
	}

	get hasAnySection(): boolean {
		return this.loadedSections.length > 0
	}

	hasSection(section: AccountProfileSection): boolean {
		return this.loadedSections.includes(section)
	}

	isSectionPending(section: AccountProfileSection): boolean {
		return !this.hasSection(section) && this.loadingSections.includes(section)
	}

	isSectionUnavailable(section: AccountProfileSection): boolean {
		return this.failedSections.includes(section)
	}

	#current(refreshCount: number): boolean {
		return (
			this.#active &&
			this.#store.vault.unlocked &&
			refreshCount === this.#refreshCount
		)
	}

	enter(): void {
		if (isFresh(this.#fetchedAt)) return
		void this.refresh()
	}

	async refresh(validateIdentity = false): Promise<void> {
		if (!this.#active || !this.#store.vault.unlocked || this.loading) return
		const refreshCount = ++this.#refreshCount
		this.error = ""
		this.failedSections = []
		this.loadingSections = profileSections.map(({ section }) => section)
		this.isValidating = validateIdentity
		const sectionLoads = profileSections.map(({ section, fields }) =>
			this.#loadSection(section, fields, refreshCount),
		)
		const [identity] = await Promise.allSettled([
			validateIdentity ? this.#refreshIdentity() : null,
			...sectionLoads,
		])
		if (!this.#current(refreshCount)) return
		this.isValidating = false
		this.#validationRequest = null
		if (identity.status === "rejected") this.#showError(identity.reason)
		if (!this.error) this.#fetchedAt = Date.now()
	}

	async #loadSection(
		section: AccountProfileSection,
		fields: (keyof AccountProfileSnapshot)[],
		refreshCount: number,
	): Promise<void> {
		const request = accountBackend.GetAccountProfile(this.#accountId, section)
		this.#requests.add(request)
		try {
			const sectionSnapshot = await request
			if (!this.#current(refreshCount)) return
			if (
				sectionSnapshot.accountId !== this.#accountId ||
				sectionSnapshot.section !== section
			) {
				throw new Error("The account changed while loading. Refresh to retry.")
			}
			if (!sectionSnapshot.isComplete) {
				this.failedSections.push(section)
				return
			}
			for (const field of fields) {
				Object.assign(this.snapshot, { [field]: sectionSnapshot[field] })
			}
			if (!this.loadedSections.includes(section))
				this.loadedSections.push(section)
		} catch (error) {
			if (!this.#current(refreshCount)) return
			this.failedSections.push(section)
			// A rejected section means the session failed, so the other sections fail too.
			this.#showError(error)
			this.#cancelRequests()
		} finally {
			this.#requests.delete(request)
			if (this.#current(refreshCount)) {
				this.loadingSections = this.loadingSections.filter(
					(loadingSection) => loadingSection !== section,
				)
			}
		}
	}

	#showError(error: unknown): void {
		this.error ||= errorMessage(
			error,
			"Could not refresh profile details. Refresh to retry.",
		)
	}

	async #refreshIdentity(): Promise<void> {
		this.#validationRequest = accountBackend.ValidateAccount(this.#accountId)
		const results = await Promise.allSettled([
			this.#validationRequest,
			this.#store.refreshAccountAvatar(this.#accountId),
			this.#store.refreshPresences(this.#accountId),
		])
		for (const settled of results) {
			if (settled.status === "rejected") throw settled.reason
		}
	}

	#cancelRequests(): void {
		for (const request of this.#requests) void request.cancel()
		this.#requests.clear()
	}

	setDescription(description: string): void {
		if (this.#active && this.#store.vault.unlocked) {
			this.snapshot.description = description
		}
	}

	dispose(): void {
		this.#active = false
		this.#refreshCount++
		this.#cancelRequests()
		void this.#validationRequest?.cancel()
		this.#validationRequest = null
		Object.assign(this.snapshot, emptyProfile(this.#accountId))
		this.loadedSections = []
		this.loadingSections = []
		this.failedSections = []
		this.isValidating = false
		this.error = ""
	}
}

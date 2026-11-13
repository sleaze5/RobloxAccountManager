import {
	AccountProfileSection,
	AgeVerification,
	accountBackend,
} from "../backend/bridge"
import type { AccountProfileSnapshot } from "../backend/bridge"
import { detailsKeepMs, isFresh } from "../shared/freshness"
import { SectionLoad } from "../shared/section-load.svelte"
import type { SectionFields } from "../shared/section-load.svelte"
import type { AccountStore } from "./account-store.svelte"
import { errorMessage } from "../shared/errors"

// Sections load in this order, so that the header fills first and the
// paginated sections, which take longest, come last. Roblox requests for one
// account run a few at a time.
const profileSections: SectionFields<AccountProfileSnapshot, AccountProfileSection>[] =
	[
		{ section: AccountProfileSection.ProfileSectionRobux, fields: ["robux"] },
		{
			section: AccountProfileSection.ProfileSectionAccount,
			fields: ["ageBracket", "countryCode", "premium", "plus"],
		},
		{
			section: AccountProfileSection.ProfileSectionAbout,
			fields: ["description", "verifiedBadge"],
		},
		{
			section: AccountProfileSection.ProfileSectionFriends,
			fields: ["friendCount"],
		},
		{
			section: AccountProfileSection.ProfileSectionFollowers,
			fields: ["followerCount"],
		},
		{
			section: AccountProfileSection.ProfileSectionFollowing,
			fields: ["followingCount"],
		},
		{
			section: AccountProfileSection.ProfileSectionPremium,
			fields: ["premiumSubscription"],
		},
		{
			section: AccountProfileSection.ProfileSectionTransactions,
			fields: [
				"pendingRobux",
				"incomingRobuxPastMonth",
				"outgoingRobuxPastMonth",
			],
		},
		{
			section: AccountProfileSection.ProfileSectionTwoStep,
			fields: ["twoStepEnabled", "twoStepMethods"],
		},
		{
			section: AccountProfileSection.ProfileSectionAge,
			fields: ["ageGroup", "ageVerification"],
		},
		{
			section: AccountProfileSection.ProfileSectionVoiceChat,
			fields: ["voiceChat"],
		},
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
			fields: ["unreadNotificationCount"],
		},
		{
			section: AccountProfileSection.ProfileSectionTrading,
			fields: ["tradeEligibility"],
		},
		{
			section: AccountProfileSection.ProfileSectionPreviousNames,
			fields: ["previousNames"],
		},
		{
			section: AccountProfileSection.ProfileSectionCollectibles,
			fields: ["collectibleCount", "collectibleRecentAveragePriceTotal"],
		},
		{
			section: AccountProfileSection.ProfileSectionConnections,
			fields: ["connections"],
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
		unreadNotificationCount: null,
		premiumSubscription: null,
		tradeEligibility: null,
		connections: null,
	}
}

// A profile on screen or still loading is checked again this long after the
// keep time ends, so that it is removed only after the user leaves it.
const expiryRecheckMs = 60_000

// The profile stays in memory for the keep time after it last loaded, so a
// revisit shows the last values at once while they refresh.
export class AccountProfileState {
	readonly sections: SectionLoad<AccountProfileSnapshot, AccountProfileSection>
	isValidating = $state(false)
	identityError = $state("")
	#active = true
	#fetchedAt = 0
	#expiryTimer = 0
	#validationRequest: ReturnType<typeof accountBackend.ValidateAccount> | null = null
	readonly #accountId: number
	readonly #store: AccountStore

	constructor(accountId: number, store: AccountStore) {
		this.#accountId = accountId
		this.#store = store
		this.sections = new SectionLoad(
			profileSections,
			() => emptyProfile(accountId),
			"Could not refresh profile details. Refresh to retry.",
		)
	}

	get snapshot(): AccountProfileSnapshot {
		return this.sections.snapshot
	}

	get loading(): boolean {
		return this.sections.isLoading || this.isValidating
	}

	get error(): string {
		return this.sections.error || this.identityError
	}

	#current(): boolean {
		return this.#active && this.#store.vault.unlocked
	}

	enter(): void {
		if (isFresh(this.#fetchedAt)) return
		void this.refresh()
	}

	async refresh(validateIdentity = false): Promise<void> {
		if (!this.#current() || this.loading) return
		this.identityError = ""
		this.isValidating = validateIdentity
		const [identity, sectionLoad] = await Promise.allSettled([
			validateIdentity ? this.#refreshIdentity() : null,
			this.sections.load(
				(section) => accountBackend.GetAccountProfile(this.#accountId, section),
				() => this.#current(),
			),
		])
		if (!this.#current()) return
		this.isValidating = false
		this.#validationRequest = null
		this.#scheduleExpiry(detailsKeepMs)
		if (identity.status === "rejected") {
			this.identityError = errorMessage(
				identity.reason,
				"Could not refresh profile details. Refresh to retry.",
			)
		}
		if (sectionLoad.status === "fulfilled" && sectionLoad.value && !this.error) {
			this.#fetchedAt = Date.now()
		}
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

	#scheduleExpiry(delayMs: number): void {
		window.clearTimeout(this.#expiryTimer)
		this.#expiryTimer = window.setTimeout(() => this.#expire(), delayMs)
	}

	#expire(): void {
		if (this.loading || this.#store.selectedAccountId === this.#accountId) {
			this.#scheduleExpiry(expiryRecheckMs)
			return
		}
		this.#store.forgetAccountProfile(this.#accountId)
	}

	setDescription(description: string): void {
		if (this.#current()) this.sections.snapshot.description = description
	}

	dispose(): void {
		this.#active = false
		window.clearTimeout(this.#expiryTimer)
		this.sections.reset()
		void this.#validationRequest?.cancel()
		this.#validationRequest = null
		this.isValidating = false
		this.identityError = ""
	}
}

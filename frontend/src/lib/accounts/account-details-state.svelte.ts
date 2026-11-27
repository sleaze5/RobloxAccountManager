import {
	AccountDetailsSection,
	AgeVerification,
	PinState,
	accountBackend,
} from "../backend/bridge"
import type { AccountDetailsSnapshot } from "../backend/bridge"
import { isFresh } from "../shared/freshness"
import { SectionLoad } from "../shared/section-load.svelte"
import type { SectionFields } from "../shared/section-load.svelte"
import type { AccountStore } from "./account-store.svelte"
import { AccountScope } from "./account-scope"

const infoSections: SectionFields<AccountDetailsSnapshot, AccountDetailsSection>[] = [
	{
		section: AccountDetailsSection.InfoSectionEmail,
		fields: ["email", "emailVerified"],
	},
	{
		section: AccountDetailsSection.InfoSectionPhone,
		fields: ["phone", "phoneVerified"],
	},
	{ section: AccountDetailsSection.InfoSectionPin, fields: ["pin"] },
	{ section: AccountDetailsSection.InfoSectionGender, fields: ["gender"] },
	{
		section: AccountDetailsSection.InfoSectionAge,
		fields: ["ageGroup", "ageVerification"],
	},
	{ section: AccountDetailsSection.InfoSectionBirthdate, fields: ["birthdate"] },
	{
		section: AccountDetailsSection.InfoSectionFirstAccount,
		fields: ["firstAccount"],
	},
	{
		section: AccountDetailsSection.InfoSectionLocale,
		fields: ["language", "autoTranslations"],
	},
	{ section: AccountDetailsSection.InfoSectionLocation, fields: ["location"] },
	{ section: AccountDetailsSection.InfoSectionVoiceChat, fields: ["voiceChat"] },
	{ section: AccountDetailsSection.InfoSectionTwoStep, fields: ["twoStepMethods"] },
	{
		section: AccountDetailsSection.InfoSectionRecoveryCodes,
		fields: ["recoveryCodeCount", "recoveryCodesCreatedAtMs"],
	},
	{ section: AccountDetailsSection.InfoSectionPasskeys, fields: ["passkeyNames"] },
]

function emptyInfo(accountId: number): AccountDetailsSnapshot {
	return {
		accountId,
		section: AccountDetailsSection.$zero,
		isComplete: false,
		email: "",
		emailVerified: null,
		phone: "",
		phoneVerified: null,
		gender: "",
		birthdate: "",
		ageGroup: "",
		ageVerification: AgeVerification.$zero,
		firstAccount: null,
		language: "",
		autoTranslations: null,
		location: "",
		pin: PinState.$zero,
		voiceChat: null,
		twoStepMethods: null,
		recoveryCodeCount: null,
		recoveryCodesCreatedAtMs: 0,
		passkeyNames: null,
	}
}

const keptPageName = "account-details"

export class AccountDetailsState {
	readonly sections: SectionLoad<AccountDetailsSnapshot, AccountDetailsSection>
	#fetchedAt = 0
	readonly #scope: AccountScope

	constructor(accountId: number, store: AccountStore) {
		this.#scope = new AccountScope(accountId, store)
		const kept =
			this.#scope.readKept<
				SectionLoad<AccountDetailsSnapshot, AccountDetailsSection>
			>(keptPageName)
		this.#fetchedAt = kept?.fetchedAt ?? 0
		this.sections =
			kept?.value ??
			new SectionLoad(
				infoSections,
				() => emptyInfo(accountId),
				"Could not load account info. Refresh to try again.",
			)
	}

	get isFresh(): boolean {
		return isFresh(this.#fetchedAt)
	}

	get snapshot(): AccountDetailsSnapshot | null {
		return this.sections.hasAnySection ? this.sections.snapshot : null
	}

	get loading(): boolean {
		return this.sections.isLoading
	}

	get error(): string {
		return this.sections.error
	}

	async refresh(): Promise<void> {
		if (!this.#scope.isCurrent || this.loading) return
		const isCurrentLoad = await this.sections.load(
			(section) => accountBackend.GetAccountInfo(this.#scope.accountId, section),
			() => this.#scope.isCurrent,
		)
		if (isCurrentLoad && !this.error) this.#fetchedAt = Date.now()
	}

	// The loaded sections are kept, so that the next visit shows them at once.
	dispose(): void {
		this.sections.cancel()
		if (this.sections.hasAnySection) {
			this.#scope.keep(keptPageName, {
				value: this.sections,
				fetchedAt: this.#fetchedAt,
			})
		}
		this.#scope.dispose()
	}
}

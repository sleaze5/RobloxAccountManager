import {
	AccountInfoSection,
	AgeVerification,
	PinState,
	accountBackend,
} from "../backend/bridge"
import type { AccountInfoSnapshot } from "../backend/bridge"
import { isFresh } from "../shared/freshness"
import { SectionLoad } from "../shared/section-load.svelte"
import type { SectionFields } from "../shared/section-load.svelte"
import type { AccountStore } from "./account-store.svelte"
import { AccountScope } from "./account-scope"

const infoSections: SectionFields<AccountInfoSnapshot, AccountInfoSection>[] = [
	{
		section: AccountInfoSection.InfoSectionEmail,
		fields: ["email", "emailVerified"],
	},
	{
		section: AccountInfoSection.InfoSectionPhone,
		fields: ["phone", "phoneVerified"],
	},
	{ section: AccountInfoSection.InfoSectionPin, fields: ["pin"] },
	{ section: AccountInfoSection.InfoSectionGender, fields: ["gender"] },
	{
		section: AccountInfoSection.InfoSectionAge,
		fields: ["ageGroup", "ageVerification"],
	},
	{ section: AccountInfoSection.InfoSectionBirthdate, fields: ["birthdate"] },
	{ section: AccountInfoSection.InfoSectionFirstAccount, fields: ["firstAccount"] },
	{
		section: AccountInfoSection.InfoSectionLocale,
		fields: ["language", "autoTranslations"],
	},
	{ section: AccountInfoSection.InfoSectionLocation, fields: ["location"] },
	{ section: AccountInfoSection.InfoSectionVoiceChat, fields: ["voiceChat"] },
	{ section: AccountInfoSection.InfoSectionTwoStep, fields: ["twoStepMethods"] },
	{
		section: AccountInfoSection.InfoSectionRecoveryCodes,
		fields: ["recoveryCodeCount", "recoveryCodesCreatedAtMs"],
	},
	{ section: AccountInfoSection.InfoSectionPasskeys, fields: ["passkeyNames"] },
]

function emptyInfo(accountId: number): AccountInfoSnapshot {
	return {
		accountId,
		section: AccountInfoSection.$zero,
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

const keptPageName = "account-info"

export class AccountInfoState {
	readonly sections: SectionLoad<AccountInfoSnapshot, AccountInfoSection>
	#fetchedAt = 0
	readonly #scope: AccountScope

	constructor(accountId: number, store: AccountStore) {
		this.#scope = new AccountScope(accountId, store)
		const kept =
			this.#scope.readKept<SectionLoad<AccountInfoSnapshot, AccountInfoSection>>(
				keptPageName,
			)
		this.#fetchedAt = kept?.fetchedAt ?? 0
		this.sections =
			kept?.value ??
			new SectionLoad(
				infoSections,
				() => emptyInfo(accountId),
				"Could not load account info. Refresh to retry.",
			)
	}

	get isFresh(): boolean {
		return isFresh(this.#fetchedAt)
	}

	get snapshot(): AccountInfoSnapshot | null {
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

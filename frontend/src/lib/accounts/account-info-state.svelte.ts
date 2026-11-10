import {
	AccountInfoSection,
	AgeVerification,
	PinState,
	accountBackend,
} from "../backend/bridge"
import type { AccountInfoSnapshot } from "../backend/bridge"
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

export class AccountInfoState {
	readonly sections: SectionLoad<AccountInfoSnapshot, AccountInfoSection>
	readonly #scope: AccountScope

	constructor(accountId: number, store: AccountStore) {
		this.#scope = new AccountScope(accountId, store)
		this.sections = new SectionLoad(
			infoSections,
			() => emptyInfo(accountId),
			"Could not load account info. Refresh to retry.",
		)
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
		await this.sections.load(
			(section) => accountBackend.GetAccountInfo(this.#scope.accountId, section),
			() => this.#scope.isCurrent,
		)
	}

	dispose(): void {
		this.#scope.dispose()
		this.sections.reset()
	}
}

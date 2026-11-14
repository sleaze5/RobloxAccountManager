import codeIcon from "@phosphor-icons/core/regular/code.svg?raw"
import eyeSlashIcon from "@phosphor-icons/core/regular/eye-slash.svg?raw"
import gameControllerIcon from "@phosphor-icons/core/regular/game-controller.svg?raw"
import globeIcon from "@phosphor-icons/core/regular/globe.svg?raw"
import globeXIcon from "@phosphor-icons/core/regular/globe-x.svg?raw"
import { AgeVerification, PresenceType, SessionState, TagKind } from "../backend/bridge"
import type {
	AccountView,
	Moderation,
	TagView,
	UserPresence,
	VoiceSuspension,
} from "../backend/bridge"

export interface Account {
	id: number
	username: string
	displayName: string
	hasSavedPassword: boolean
	avatarUrl: string
	presence: UserPresence
	userId: string
	tags: TagView[]
	createdAtMs: number
	favorite: boolean
	state: SessionState
	stateReason: string
	moderation: Moderation | null
	voiceSuspension: VoiceSuspension | null
	cookieExpiresAtMs: number | null
	importedAtMs: number
	lastValidatedAtMs: number | null
	rotatedAtMs: number | null
}

export const ageVerificationLabels: Partial<Record<AgeVerification, string>> = {
	[AgeVerification.AgeUnverified]: "Unverified",
	[AgeVerification.AgeVerifiedFaceScan]: "Verified (facial scan)",
	[AgeVerification.AgeVerifiedID]: "Verified (ID)",
}

export const emptyAccount: Account = {
	avatarUrl: "",
	cookieExpiresAtMs: null,
	createdAtMs: 0,
	displayName: "No account selected",
	favorite: false,
	hasSavedPassword: false,
	id: 0,
	importedAtMs: 0,
	lastValidatedAtMs: null,
	presence: offlinePresence(0),
	rotatedAtMs: null,
	state: SessionState.StateUnknown,
	stateReason: "",
	moderation: null,
	voiceSuspension: null,
	tags: [],
	userId: "N/A",
	username: "",
}

export function accountFromView(
	record: AccountView,
	avatarUrl: string,
	presence?: UserPresence,
): Account {
	const displayName = record.displayName || record.username,
		tags = record.tags ?? []

	return {
		avatarUrl,
		cookieExpiresAtMs: record.cookieExpiresAtMs ?? null,
		createdAtMs: record.createdAtMs,
		displayName,
		favorite: tags.some((tag) => tag.kind === TagKind.TagKindFavorite),
		hasSavedPassword: record.hasSavedPassword,
		id: record.id,
		importedAtMs: record.importedAtMs,
		lastValidatedAtMs: record.lastValidatedAtMs ?? null,
		presence: presence ?? offlinePresence(record.robloxUserId),
		rotatedAtMs: record.rotatedAtMs ?? null,
		state: record.state,
		stateReason: record.stateReason ?? "",
		moderation: record.moderation ?? null,
		voiceSuspension: record.voiceSuspension ?? null,
		tags,
		userId: String(record.robloxUserId),
		username: record.username,
	}
}

export function accountWithPresence(account: Account, presence: UserPresence): Account {
	return { ...account, presence }
}

export function presenceLabel(presence: UserPresence): string {
	switch (presence.userPresenceType) {
		case PresenceType.PresenceTypeOnline:
			return "Online"
		case PresenceType.PresenceTypeInGame:
			return "In game"
		case PresenceType.PresenceTypeInStudio:
			return "In Studio"
		case PresenceType.PresenceTypeInvisible:
			return "Invisible"
		default:
			return "Offline"
	}
}

export function presenceDotClass(presence: UserPresence): string {
	switch (presence.userPresenceType) {
		case PresenceType.PresenceTypeOnline:
			return "bg-presence-online"
		case PresenceType.PresenceTypeInGame:
			return "bg-presence-game"
		case PresenceType.PresenceTypeInStudio:
			return "bg-presence-studio"
		default:
			return "bg-fg-faint"
	}
}

export function presenceTextClass(presence: UserPresence): string {
	switch (presence.userPresenceType) {
		case PresenceType.PresenceTypeOnline:
			return "text-presence-online"
		case PresenceType.PresenceTypeInGame:
			return "text-presence-game"
		case PresenceType.PresenceTypeInStudio:
			return "text-presence-studio"
		default:
			return "text-fg-muted"
	}
}

export function presenceIcon(presence: UserPresence): string {
	switch (presence.userPresenceType) {
		case PresenceType.PresenceTypeOnline:
			return globeIcon
		case PresenceType.PresenceTypeInGame:
			return gameControllerIcon
		case PresenceType.PresenceTypeInStudio:
			return codeIcon
		case PresenceType.PresenceTypeInvisible:
			return eyeSlashIcon
		default:
			return globeXIcon
	}
}

export function presenceDetail(presence: UserPresence, experience = ""): string {
	const location = presence.lastLocation?.trim() ?? ""
	switch (presence.userPresenceType) {
		case PresenceType.PresenceTypeInGame:
			return experience || location
		case PresenceType.PresenceTypeInStudio:
			return experience || (location.toLowerCase() === "studio" ? "" : location)
		case PresenceType.PresenceTypeOnline:
			return location.toLowerCase() === "website" ? "" : location
		default:
			return ""
	}
}

export function presenceTooltip(presence: UserPresence): string {
	const label = presenceLabel(presence),
		detail = presenceDetail(presence)
	return detail ? `${label}: ${detail}` : label
}

export function hasTag(account: Account, tagId: number): boolean {
	return account.tags.some((tag) => tag.id === tagId)
}

export function getCustomTags(account: Account): TagView[] {
	return account.tags.filter((tag) => tag.kind === TagKind.TagKindCustom)
}

function offlinePresence(userId: number): UserPresence {
	return { userId, userPresenceType: PresenceType.PresenceTypeOffline }
}

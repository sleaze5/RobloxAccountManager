import codeIcon from "@phosphor-icons/core/regular/code.svg?raw"
import eyeSlashIcon from "@phosphor-icons/core/regular/eye-slash.svg?raw"
import gameControllerIcon from "@phosphor-icons/core/regular/game-controller.svg?raw"
import globeIcon from "@phosphor-icons/core/regular/globe.svg?raw"
import globeXIcon from "@phosphor-icons/core/regular/globe-x.svg?raw"
import {
	AgeVerification,
	OnlineStatusType,
	SessionState,
	TagKind,
} from "../backend/bridge"
import type {
	AccountView,
	Moderation,
	TagView,
	OnlineStatus,
	VoiceSuspension,
} from "../backend/bridge"

export interface Account {
	id: number
	username: string
	displayName: string
	hasSavedPassword: boolean
	avatarUrl: string
	onlineStatus: OnlineStatus
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
	[AgeVerification.AgeUnverified]: "Not verified",
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
	onlineStatus: offlineOnlineStatus(0),
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
	onlineStatus?: OnlineStatus,
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
		onlineStatus: onlineStatus ?? offlineOnlineStatus(record.robloxUserId),
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

export function accountWithOnlineStatus(
	account: Account,
	onlineStatus: OnlineStatus,
): Account {
	return { ...account, onlineStatus }
}

export function onlineStatusLabel(onlineStatus: OnlineStatus): string {
	switch (onlineStatus.statusType) {
		case OnlineStatusType.OnlineStatusTypeOnline:
			return "Online"
		case OnlineStatusType.OnlineStatusTypeInGame:
			return "In game"
		case OnlineStatusType.OnlineStatusTypeInStudio:
			return "In Studio"
		case OnlineStatusType.OnlineStatusTypeInvisible:
			return "Invisible"
		default:
			return "Offline"
	}
}

export function onlineStatusDotClass(onlineStatus: OnlineStatus): string {
	switch (onlineStatus.statusType) {
		case OnlineStatusType.OnlineStatusTypeOnline:
			return "bg-presence-online"
		case OnlineStatusType.OnlineStatusTypeInGame:
			return "bg-presence-game"
		case OnlineStatusType.OnlineStatusTypeInStudio:
			return "bg-presence-studio"
		default:
			return "bg-fg-faint"
	}
}

export function onlineStatusTextClass(onlineStatus: OnlineStatus): string {
	switch (onlineStatus.statusType) {
		case OnlineStatusType.OnlineStatusTypeOnline:
			return "text-presence-online"
		case OnlineStatusType.OnlineStatusTypeInGame:
			return "text-presence-game"
		case OnlineStatusType.OnlineStatusTypeInStudio:
			return "text-presence-studio"
		default:
			return "text-fg-muted"
	}
}

export function onlineStatusIcon(onlineStatus: OnlineStatus): string {
	switch (onlineStatus.statusType) {
		case OnlineStatusType.OnlineStatusTypeOnline:
			return globeIcon
		case OnlineStatusType.OnlineStatusTypeInGame:
			return gameControllerIcon
		case OnlineStatusType.OnlineStatusTypeInStudio:
			return codeIcon
		case OnlineStatusType.OnlineStatusTypeInvisible:
			return eyeSlashIcon
		default:
			return globeXIcon
	}
}

export function onlineStatusDetail(onlineStatus: OnlineStatus, gameName = ""): string {
	const location = onlineStatus.lastLocation?.trim() ?? ""
	switch (onlineStatus.statusType) {
		case OnlineStatusType.OnlineStatusTypeInGame:
			return gameName || location
		case OnlineStatusType.OnlineStatusTypeInStudio:
			return gameName || (location.toLowerCase() === "studio" ? "" : location)
		case OnlineStatusType.OnlineStatusTypeOnline:
			return location.toLowerCase() === "website" ? "" : location
		default:
			return ""
	}
}

export function onlineStatusTooltip(onlineStatus: OnlineStatus): string {
	const label = onlineStatusLabel(onlineStatus),
		detail = onlineStatusDetail(onlineStatus)
	return detail ? `${label}: ${detail}` : label
}

// sessionAttentionLabel names a session state that the user must fix before
// the account works again, or returns "" when the session needs nothing.
export function sessionAttentionLabel(state: SessionState): string {
	switch (state) {
		case SessionState.StateReauthRequired:
			return "Sign-in required"
		case SessionState.StateChallenged:
			return "Security challenge"
		default:
			return ""
	}
}

export function hasTag(account: Account, tagId: number): boolean {
	return account.tags.some((tag) => tag.id === tagId)
}

export function getCustomTags(account: Account): TagView[] {
	return account.tags.filter((tag) => tag.kind === TagKind.TagKindCustom)
}

function offlineOnlineStatus(userId: number): OnlineStatus {
	return { userId, statusType: OnlineStatusType.OnlineStatusTypeOffline }
}

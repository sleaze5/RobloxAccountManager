import { PresenceType, SessionState, TagKind } from "../backend/bridge"
import type { AccountView, TagView, UserPresence } from "../backend/bridge"

export interface Account {
	id: number
	username: string
	displayName: string
	avatarUrl: string
	presence: UserPresence
	userId: string
	tags: TagView[]
	createdAtMs: number
	favorite: boolean
	needsAttention: boolean
	state: SessionState
	stateReason: string
	cookieExpiresAtMs: number | null
	importedAtMs: number
	lastValidatedAtMs: number | null
	rotatedAtMs: number | null
}

export const emptyAccount: Account = {
	avatarUrl: "",
	cookieExpiresAtMs: null,
	createdAtMs: 0,
	displayName: "No account selected",
	favorite: false,
	id: 0,
	importedAtMs: 0,
	lastValidatedAtMs: null,
	needsAttention: false,
	presence: unknownPresence(0),
	rotatedAtMs: null,
	state: SessionState.StateUnknown,
	stateReason: "",
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
		id: record.id,
		importedAtMs: record.importedAtMs,
		lastValidatedAtMs: record.lastValidatedAtMs ?? null,
		needsAttention: sessionNeedsAttention(record.state),
		presence: presence ?? unknownPresence(record.robloxUserId),
		rotatedAtMs: record.rotatedAtMs ?? null,
		state: record.state,
		stateReason: record.stateReason ?? "",
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
		case PresenceType.PresenceTypeOffline:
			return "Offline"
		case PresenceType.PresenceTypeOnline:
			return "Online"
		case PresenceType.PresenceTypeInGame:
			return "In game"
		case PresenceType.PresenceTypeInStudio:
			return "In Studio"
		case PresenceType.PresenceTypeInvisible:
			return "Invisible"
		default:
			return "Unknown"
	}
}

export function presenceClass(presence: UserPresence): string {
	switch (presence.userPresenceType) {
		case PresenceType.PresenceTypeOnline:
			return "presence-online"
		case PresenceType.PresenceTypeInGame:
			return "presence-in-game"
		case PresenceType.PresenceTypeInStudio:
			return "presence-in-studio"
		case PresenceType.PresenceTypeInvisible:
			return "presence-invisible"
		case PresenceType.PresenceTypeOffline:
			return "presence-offline"
		default:
			return "presence-unknown"
	}
}

export function presenceTooltip(presence: UserPresence): string {
	const label = presenceLabel(presence),
		location = presence.lastLocation?.trim()

	return location && location.toLowerCase() !== label.toLowerCase()
		? `${label}: ${location}`
		: label
}

export function isOnlinePresence(type: PresenceType): boolean {
	return (
		type === PresenceType.PresenceTypeOnline ||
		type === PresenceType.PresenceTypeInGame ||
		type === PresenceType.PresenceTypeInStudio
	)
}

export function hasTag(account: Account, tagId: number): boolean {
	return account.tags.some((tag) => tag.id === tagId)
}

export function getCustomTags(account: Account): TagView[] {
	return account.tags.filter((tag) => tag.kind === TagKind.TagKindCustom)
}

function unknownPresence(userId: number): UserPresence {
	return { userId, userPresenceType: PresenceType.PresenceTypeUnknown }
}

function sessionNeedsAttention(state: SessionState): boolean {
	return !(
		state === SessionState.StateActive ||
		state === SessionState.StateUnknown ||
		state === SessionState.StateDisabled
	)
}

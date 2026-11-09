import type { Moderation } from "../backend/bridge"

export interface CandidateListItem {
	id: string
	indexLabel?: string
	selectable: boolean
	selectionLabel: string
	duplicateText?: string
	displayName?: string
	username?: string
	robloxUserId?: number
	avatarUrl?: string
	status?: string
	moderation?: Moderation | null
	error?: string
}

import { PresenceType, UserJoinability } from "../backend/bridge"
import type { UserPresence } from "../backend/bridge"

// PresenceJoin tells how the launch options can follow a user from their
// online status.
interface PresenceJoin {
	hasVisibleServer: boolean
	placeId: number
	jobId: string
	// canFollowUser reports that the profile offers to join a user whose
	// server the status hides.
	canFollowUser: boolean
	isNotJoinable: boolean
}

// presenceJoinFor reads the join options of a user. isFollowAvailable tells
// whether the page can follow a user without a server, which only a profile
// of someone else can.
export function presenceJoinFor(
	presence: UserPresence,
	joinability: UserJoinability,
	isFollowAvailable: boolean,
): PresenceJoin {
	const placeId = presence.placeId ?? 0,
		hasVisibleServer =
			presence.userPresenceType === PresenceType.PresenceTypeInGame &&
			placeId > 0 &&
			!!presence.gameId
	return {
		hasVisibleServer,
		placeId,
		jobId: presence.gameId ?? "",
		canFollowUser:
			!hasVisibleServer &&
			isFollowAvailable &&
			joinability === UserJoinability.UserJoinable,
		isNotJoinable: joinability === UserJoinability.UserNotJoinable,
	}
}

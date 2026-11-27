import { OnlineStatusType, UserJoinability } from "../backend/bridge"
import type { OnlineStatus } from "../backend/bridge"

// OnlineStatusJoin tells how the launch options can follow a user from their
// online status.
interface OnlineStatusJoin {
	hasVisibleServer: boolean
	placeId: number
	serverId: string
	// canFollowUser reports that the profile offers to join a user whose
	// server the status hides.
	canFollowUser: boolean
	isNotJoinable: boolean
}

// onlineStatusJoinFor reads the join options of a user. isFollowAvailable tells
// whether the page can follow a user without a server, which only a profile
// of someone else can.
export function onlineStatusJoinFor(
	onlineStatus: OnlineStatus,
	joinability: UserJoinability,
	isFollowAvailable: boolean,
): OnlineStatusJoin {
	const placeId = onlineStatus.placeId ?? 0,
		hasVisibleServer =
			onlineStatus.statusType === OnlineStatusType.OnlineStatusTypeInGame &&
			placeId > 0 &&
			!!onlineStatus.serverId
	return {
		hasVisibleServer,
		placeId,
		serverId: onlineStatus.serverId ?? "",
		canFollowUser:
			!hasVisibleServer &&
			isFollowAvailable &&
			joinability === UserJoinability.UserJoinable,
		isNotJoinable: joinability === UserJoinability.UserNotJoinable,
	}
}

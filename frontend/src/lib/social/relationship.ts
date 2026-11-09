import handshakeIcon from "@phosphor-icons/core/regular/handshake.svg?raw"
import paperPlaneTiltIcon from "@phosphor-icons/core/regular/paper-plane-tilt.svg?raw"
import prohibitIcon from "@phosphor-icons/core/regular/prohibit.svg?raw"
import shieldCheckIcon from "@phosphor-icons/core/regular/shield-check.svg?raw"
import shieldSlashIcon from "@phosphor-icons/core/regular/shield-slash.svg?raw"
import shieldWarningIcon from "@phosphor-icons/core/regular/shield-warning.svg?raw"
import userCheckIcon from "@phosphor-icons/core/regular/user-check.svg?raw"
import userCircleMinusIcon from "@phosphor-icons/core/regular/user-circle-minus.svg?raw"
import userCirclePlusIcon from "@phosphor-icons/core/regular/user-circle-plus.svg?raw"
import userFocusIcon from "@phosphor-icons/core/regular/user-focus.svg?raw"
import userMinusIcon from "@phosphor-icons/core/regular/user-minus.svg?raw"
import userPlusIcon from "@phosphor-icons/core/regular/user-plus.svg?raw"
import { FriendStatus, TrustedStatus } from "../backend/bridge"
import type { SocialRelationship, SocialUser } from "../backend/bridge"

// The parts of a relationship that both a profile and a saved account
// connection report.
export type RelationshipState = Pick<
	SocialRelationship,
	"friend" | "trusted" | "following" | "followedBy"
>

export type RelationshipActionId =
	| "add-friend"
	| "cancel-friend"
	| "accept-friend"
	| "decline-friend"
	| "unfriend"
	| "add-trusted"
	| "cancel-trusted"
	| "accept-trusted"
	| "decline-trusted"
	| "remove-trusted"
	| "follow"
	| "follow-back"
	| "unfollow"

// neutral adds or accepts a connection, warning withdraws a request the
// account sent, and danger ends or refuses a connection.
export type RelationshipTone = "neutral" | "warning" | "danger"

export interface RelationshipAction {
	id: RelationshipActionId
	tone: RelationshipTone
	label: string
	progress: string
	icon: string
	question: (name: string) => string
	consequence: string
	done: string
	result: (name: string) => string
	failure: string
}

export interface RelationshipBadge {
	label: string
	icon: string
	tooltip: string
	isStrong: boolean
}

export const relationshipActionsById: Record<RelationshipActionId, RelationshipAction> =
	{
		"add-friend": {
			id: "add-friend",
			tone: "neutral",
			label: "Add friend",
			progress: "Sending…",
			icon: userPlusIcon,
			question: (name) => `Send a friend request to ${name}?`,
			consequence: "They can accept or decline it on Roblox.",
			done: "Friend request sent",
			result: (name) => `${name} can now accept the request.`,
			failure: "Could not send the friend request. Try again.",
		},
		"cancel-friend": {
			id: "cancel-friend",
			tone: "warning",
			label: "Cancel friend request",
			progress: "Canceling…",
			icon: userMinusIcon,
			question: (name) => `Cancel your friend request to ${name}?`,
			consequence:
				"They can no longer accept it. You can send a new request later.",
			done: "Friend request canceled",
			result: (name) => `${name} can no longer accept the friend request.`,
			failure: "Could not cancel the friend request. Try again.",
		},
		"accept-friend": {
			id: "accept-friend",
			tone: "neutral",
			label: "Accept friend",
			progress: "Accepting…",
			icon: userPlusIcon,
			question: (name) => `Accept the friend request from ${name}?`,
			consequence: "You become friends on Roblox.",
			done: "Friend request accepted",
			result: (name) => `${name} is now a friend.`,
			failure: "Could not accept the friend request. Try again.",
		},
		"decline-friend": {
			id: "decline-friend",
			tone: "danger",
			label: "Decline friend",
			progress: "Declining…",
			icon: prohibitIcon,
			question: (name) => `Decline the friend request from ${name}?`,
			consequence: "They can send a new request later.",
			done: "Friend request declined",
			result: (name) => `The request from ${name} was declined.`,
			failure: "Could not decline the friend request. Try again.",
		},
		"unfriend": {
			id: "unfriend",
			tone: "danger",
			label: "Unfriend",
			progress: "Unfriending…",
			icon: userMinusIcon,
			question: (name) => `Unfriend ${name}?`,
			consequence:
				"This ends the friendship and any trusted friendship. To be friends again, one of you must send a new request.",
			done: "Unfriended",
			result: (name) => `${name} is no longer a friend.`,
			failure: "Could not unfriend this user. Try again.",
		},
		"add-trusted": {
			id: "add-trusted",
			tone: "neutral",
			label: "Add trusted friend",
			progress: "Sending…",
			icon: shieldCheckIcon,
			question: (name) => `Send a trusted friend request to ${name}?`,
			consequence: "They can accept or decline it on Roblox.",
			done: "Trusted friend request sent",
			result: (name) => `${name} can now accept the request.`,
			failure: "Could not send the trusted friend request. Try again.",
		},
		"cancel-trusted": {
			id: "cancel-trusted",
			tone: "warning",
			label: "Cancel trusted request",
			progress: "Canceling…",
			icon: shieldSlashIcon,
			question: (name) => `Cancel your trusted friend request to ${name}?`,
			consequence: "You stay friends. You can send a new trusted request later.",
			done: "Trusted friend request canceled",
			result: (name) =>
				`${name} can no longer accept the trusted friend request.`,
			failure: "Could not cancel the trusted friend request. Try again.",
		},
		"accept-trusted": {
			id: "accept-trusted",
			tone: "neutral",
			label: "Accept trusted friend",
			progress: "Accepting…",
			icon: shieldCheckIcon,
			question: (name) => `Accept the trusted friend request from ${name}?`,
			consequence: "You become trusted friends on Roblox.",
			done: "Trusted friend request accepted",
			result: (name) => `${name} is now a trusted friend.`,
			failure: "Could not accept the trusted friend request. Try again.",
		},
		"decline-trusted": {
			id: "decline-trusted",
			tone: "danger",
			label: "Decline trusted friend",
			progress: "Declining…",
			icon: shieldWarningIcon,
			question: (name) => `Decline the trusted friend request from ${name}?`,
			consequence: "You stay friends. They can send a new trusted request later.",
			done: "Trusted friend request declined",
			result: (name) => `The trusted friend request from ${name} was declined.`,
			failure: "Could not decline the trusted friend request. Try again.",
		},
		"remove-trusted": {
			id: "remove-trusted",
			tone: "danger",
			label: "Remove trusted friend",
			progress: "Removing…",
			icon: shieldSlashIcon,
			question: (name) => `Remove ${name} as a trusted friend?`,
			consequence: "You stay friends, but adding them back takes a new request.",
			done: "Trusted friend removed",
			result: (name) => `${name} is no longer a trusted friend.`,
			failure: "Could not remove this trusted friend. Try again.",
		},
		"follow": {
			id: "follow",
			tone: "neutral",
			label: "Follow",
			progress: "Following…",
			icon: userCirclePlusIcon,
			question: (name) => `Follow ${name}?`,
			consequence: "You can unfollow them at any time.",
			done: "Followed",
			result: (name) => `${name} is now followed.`,
			failure: "Could not follow this user. Try again.",
		},
		"follow-back": {
			id: "follow-back",
			tone: "neutral",
			label: "Follow back",
			progress: "Following…",
			icon: userCirclePlusIcon,
			question: (name) => `Follow ${name} back?`,
			consequence: "You can unfollow them at any time.",
			done: "Followed",
			result: (name) => `${name} is now followed.`,
			failure: "Could not follow this user. Try again.",
		},
		"unfollow": {
			id: "unfollow",
			tone: "danger",
			label: "Unfollow",
			progress: "Unfollowing…",
			icon: userCircleMinusIcon,
			question: (name) => `Unfollow ${name}?`,
			consequence: "You can follow them again at any time.",
			done: "Unfollowed",
			result: (name) => `${name} is no longer followed.`,
			failure: "Could not unfollow this user. Try again.",
		},
	}

// A friend request in the Requests list implies the relationship before the
// profile loads, so its actions are available at once.
export function requestRelationship(person: SocialUser): SocialRelationship | null {
	if (!person.request) return null
	return {
		self: false,
		friend: FriendStatus.FriendStatusRequestReceived,
		trusted: TrustedStatus.TrustedStatusNone,
		following: false,
		followedBy: false,
		nickname: person.nickname,
	}
}

function friendActionIds(relationship: SocialRelationship): RelationshipActionId[] {
	switch (relationship.friend) {
		case FriendStatus.FriendStatusFriends:
			return ["unfriend"]
		case FriendStatus.FriendStatusRequestSent:
			return ["cancel-friend"]
		case FriendStatus.FriendStatusRequestReceived:
			return ["accept-friend", "decline-friend"]
		default:
			return ["add-friend"]
	}
}

// Roblox offers trusted friendship only between friends.
function trustedActionIds(relationship: SocialRelationship): RelationshipActionId[] {
	switch (relationship.trusted) {
		case TrustedStatus.TrustedStatusTrusted:
			return ["remove-trusted"]
		case TrustedStatus.TrustedStatusRequestSent:
			return ["cancel-trusted"]
		case TrustedStatus.TrustedStatusRequestReceived:
			return ["accept-trusted", "decline-trusted"]
		default:
			return relationship.friend === FriendStatus.FriendStatusFriends
				? ["add-trusted"]
				: []
	}
}

function followActionIds(relationship: SocialRelationship): RelationshipActionId[] {
	if (relationship.following) return ["unfollow"]
	return relationship.followedBy ? ["follow-back"] : ["follow"]
}

// pendingRequestActionIds answer or withdraw a friend or trusted friend request
// that is still pending, in either direction.
export const pendingRequestActionIds: ReadonlySet<RelationshipActionId> = new Set([
	"accept-friend",
	"decline-friend",
	"cancel-friend",
	"accept-trusted",
	"decline-trusted",
	"cancel-trusted",
])

// relationshipActions lists what the browsing account can do with a user,
// leaving out the parts of the relationship that could not be read.
export function relationshipActions(
	relationship: SocialRelationship,
	unavailable: string[],
): RelationshipAction[] {
	if (relationship.self) return []
	return [
		...(unavailable.includes("friendship") ? [] : friendActionIds(relationship)),
		...(unavailable.includes("trusted") ? [] : trustedActionIds(relationship)),
		...(unavailable.includes("follow") ? [] : followActionIds(relationship)),
	].map((id) => relationshipActionsById[id])
}

const buttonToneClasses: Record<RelationshipTone, string> = {
		neutral: "button",
		warning: "button-warning",
		danger: "button-danger",
	},
	menuItemToneClasses: Record<RelationshipTone, string> = {
		neutral: "menu-item",
		warning: "menu-item-warning",
		danger: "menu-item-danger",
	}

export function relationshipButtonClass(tone: RelationshipTone): string {
	return buttonToneClasses[tone]
}

export function relationshipMenuItemClass(tone: RelationshipTone): string {
	return menuItemToneClasses[tone]
}

// relationshipBadges describes a relationship. A trusted friendship replaces
// the friend badge, because a trusted friend is always a friend.
export function relationshipBadges(
	relationship: RelationshipState,
): RelationshipBadge[] {
	const badges: RelationshipBadge[] = []
	if (relationship.trusted === TrustedStatus.TrustedStatusTrusted) {
		badges.push({
			label: "Trusted friends",
			icon: shieldCheckIcon,
			tooltip: "Trusted friends on Roblox",
			isStrong: true,
		})
	} else if (relationship.friend === FriendStatus.FriendStatusFriends) {
		badges.push({
			label: "Friends",
			icon: handshakeIcon,
			tooltip: "Friends on Roblox",
			isStrong: true,
		})
	}
	if (relationship.friend === FriendStatus.FriendStatusRequestSent) {
		badges.push({
			label: "Friend request pending",
			icon: paperPlaneTiltIcon,
			tooltip: "This account sent a friend request that they have not answered",
			isStrong: false,
		})
	} else if (relationship.friend === FriendStatus.FriendStatusRequestReceived) {
		badges.push({
			label: "Friend request received",
			icon: userPlusIcon,
			tooltip: "They sent a friend request",
			isStrong: false,
		})
	}
	if (relationship.trusted === TrustedStatus.TrustedStatusRequestSent) {
		badges.push({
			label: "Trusted request pending",
			icon: paperPlaneTiltIcon,
			tooltip:
				"This account sent a trusted friend request that they have not answered",
			isStrong: false,
		})
	} else if (relationship.trusted === TrustedStatus.TrustedStatusRequestReceived) {
		badges.push({
			label: "Trusted request received",
			icon: shieldCheckIcon,
			tooltip: "They sent a trusted friend request",
			isStrong: false,
		})
	}
	if (relationship.followedBy) {
		badges.push({
			label: "Follower",
			icon: userCheckIcon,
			tooltip: "They follow this account",
			isStrong: false,
		})
	}
	if (relationship.following) {
		badges.push({
			label: "Following",
			icon: userFocusIcon,
			tooltip: "This account follows them",
			isStrong: false,
		})
	}
	return badges
}

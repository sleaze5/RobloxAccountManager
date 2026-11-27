import type { ConfirmationText } from "../shared/roblox-confirmations"

// joinConfirmation asks before the account joins a group on Roblox, or asks
// to join one that reviews join requests.
export function joinConfirmation(group: {
	name: string
	isPublicEntryAllowed: boolean
}): ConfirmationText {
	return group.isPublicEntryAllowed
		? {
				title: "Join group on Roblox?",
				description: `This account becomes a member of ${group.name}.`,
				confirmLabel: "Join group",
				progressLabel: "Joining…",
			}
		: {
				title: "Request to join on Roblox?",
				description: `${group.name} reviews the request before this account becomes a member.`,
				confirmLabel: "Request to join",
				progressLabel: "Joining…",
			}
}

// primaryGroupConfirmation asks before the account makes a group its primary
// group on Roblox, or removes it as the primary group when isPrimary is true.
export function primaryGroupConfirmation(
	groupName: string,
	isPrimary: boolean,
): ConfirmationText {
	return isPrimary
		? {
				title: "Remove primary group on Roblox?",
				description: `${groupName} no longer shows on the profile of this account. The account stays a member.`,
				confirmLabel: "Remove primary group",
				progressLabel: "Removing…",
			}
		: {
				title: "Make primary group on Roblox?",
				description: `${groupName} shows on the profile of this account. It replaces the current primary group.`,
				confirmLabel: "Make primary group",
				progressLabel: "Saving…",
			}
}

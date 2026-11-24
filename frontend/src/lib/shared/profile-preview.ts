import { createContext } from "svelte"

// ProfilePreviewSubject names a user, a group, or a game that another page
// links to. Its preview shows the profile's information in a dialog above the
// page, and opens the full profile on its own page on request.
export type ProfilePreviewSubject =
	| { kind: "User"; userId: number; username: string; displayName: string }
	| { kind: "Group"; groupId: number }
	| {
			kind: "Place"
			placeId: number
			// universeId makes the full profile list every place of the
			// universe, with the place opened.
			universeId?: number
			// gamePassId opens the full profile on the game pass.
			gamePassId?: number
	  }

interface ProfilePreviews {
	show: (subject: ProfilePreviewSubject) => void
}

export const [getProfilePreviews, setProfilePreviews] = createContext<ProfilePreviews>()

export function isSameProfilePreviewSubject(
	first: ProfilePreviewSubject,
	second: ProfilePreviewSubject,
): boolean {
	if (first.kind === "User" && second.kind === "User")
		return first.userId === second.userId
	if (first.kind === "Group" && second.kind === "Group")
		return first.groupId === second.groupId
	if (first.kind === "Place" && second.kind === "Place")
		return (
			first.placeId === second.placeId &&
			first.universeId === second.universeId &&
			first.gamePassId === second.gamePassId
		)
	return false
}

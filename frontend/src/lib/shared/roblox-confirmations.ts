// ConfirmationText is what a confirmation says before a change on Roblox.
export interface ConfirmationText {
	title: string
	description: string
	confirmLabel: string
	progressLabel: string
}

// favoriteConfirmation asks before the account adds a game or an item to its
// favorites on Roblox, or removes it.
export function favoriteConfirmation(
	name: string,
	username: string,
	isAddingFavorite: boolean,
): ConfirmationText {
	return isAddingFavorite
		? {
				title: "Add to favorites on Roblox?",
				description: `${name} shows in the favorites of @${username}, which other users can see.`,
				confirmLabel: "Add to favorites",
				progressLabel: "Adding…",
			}
		: {
				title: "Remove from favorites on Roblox?",
				description: `${name} leaves the favorites of @${username}.`,
				confirmLabel: "Remove from favorites",
				progressLabel: "Removing…",
			}
}

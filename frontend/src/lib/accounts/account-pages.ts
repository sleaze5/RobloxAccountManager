// AccountPage names a page that opens in place of an account's profile.
export type AccountPage =
	| "avatar"
	| "inventory"
	| "chats"
	| "notifications"
	| "settings"

// accountPageLabels names the pages of an account profile, in the order of
// their tabs.
export const accountPageLabels: Record<AccountPage, string> = {
	avatar: "Avatar",
	inventory: "Inventory",
	chats: "Chats",
	notifications: "Notifications",
	settings: "Settings",
}

export const accountPages = Object.keys(accountPageLabels) as AccountPage[]

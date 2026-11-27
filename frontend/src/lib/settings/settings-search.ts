import { matchesSearch } from "../shared/search"

// settingSearchWords lists, for each setting, the words that find it in the
// Settings search: its label, its description, and other words people use for
// the same thing.
const settingSearchWords = {
	vaultStatus:
		"vault status locked unlocked lock security check test master password last checked tested",
	automaticUnlock:
		"automatic unlock auto unlock autounlock remember password skip password startup start login sign in operating system user account windows hello keychain on off",
	changeMasterPassword:
		"change master password new password reset password rotate encryption key security",
	passwordCheckReminder:
		"password check reminder interval how often remind confirm master password check memory forget",
	storageLocation:
		"storage location data folder directory path where saved files portable standard appdata copy path",
	moveStorage:
		"move data storage relocate portable standard folder vault backups settings restart",
	exportData:
		"export data backup save file encrypted transfer move computer copy accounts cookies passwords tags bookmarks settings password hint",
	importData:
		"import data restore load file transfer move computer merge overwrite replace accounts cookies passwords tags bookmarks settings",
	onlineStatusUpdates:
		"online status presence updates refresh interval seconds minutes polling in game offline friends",
	onlineStatusScopes:
		"account list all saved accounts active profile selected account social listed users",
	logging: "logging logs log levels diagnostics troubleshooting verbose",
	logLevels:
		"trace debug info warning warn error detailed requests diagnostic activity problems",
	allLogLevels: "all log levels logging logs every level turn on off",
	motion: "motion animations animation effects reduce reduced reduced motion full system transitions accessibility",
	timestampFormat:
		"timestamp format date time clock display tokens dollar 12-hour 24-hour",
	hoverFormat: "hover format tooltip timestamp date time relative tokens dollar",
	timestampPreview: "timestamp preview example hover tooltip date time",
	timestampActions: "save timestamp formats reset to defaults restore",
	browser:
		"browser chrome for testing chromium runtime download redownload remove delete uninstall disk space size version hosts sign in storage.googleapis.com",
	serverData:
		"rovalra server data regions server age nearest server preferred region country location ping launch join integration api hosts privacy data sent apis.rovalra.com",
	sharedPrivateServers:
		"rbxservers shared private servers vip free private servers integration api hosts privacy data sent rbxservers.xyz api.rbxservers.xyz",
	multiClient:
		"multi-client multi client multiple clients multi-instance multi instance multiple roblox several games alt accounts at once same time windows",
	linuxClient: "linux client sober mocktail flatpak wine join games",
	cleanBeforeLaunch:
		"clean before launch clear cache caches cookies sign-in data traces fresh logout",
	cleanRoblox:
		"clean roblox data clear cache caches cookies delete logs administrator admin rights uac elevation",
	uninstallRoblox:
		"uninstall roblox remove studio delete registry shortcuts reinstall",
	updates:
		"updates update check for updates latest version new release download install restart upgrade",
	about: "about version build platform data folder started launch id startup timings components dependencies licenses diagnostics",
} as const

type SettingSearchKey = keyof typeof settingSearchWords

export const settingsCategories = {
	"vault": {
		label: "Vault",
		keys: [
			"vaultStatus",
			"automaticUnlock",
			"changeMasterPassword",
			"passwordCheckReminder",
			"storageLocation",
			"moveStorage",
			"exportData",
			"importData",
		],
	},
	"general": {
		label: "General",
		keys: [
			"onlineStatusUpdates",
			"onlineStatusScopes",
			"logging",
			"logLevels",
			"allLogLevels",
		],
	},
	"user-interface": {
		label: "User interface",
		keys: [
			"motion",
			"timestampFormat",
			"hoverFormat",
			"timestampPreview",
			"timestampActions",
		],
	},
	"roblox": {
		label: "Roblox",
		keys: [
			"multiClient",
			"linuxClient",
			"cleanBeforeLaunch",
			"cleanRoblox",
			"uninstallRoblox",
		],
	},
	"integrations": {
		label: "Integrations",
		keys: ["browser", "serverData", "sharedPrivateServers"],
	},
	"about": { label: "About", keys: ["updates", "about"] },
} as const satisfies Record<
	string,
	{ label: string; keys: readonly SettingSearchKey[] }
>

type SettingsCategoryId = keyof typeof settingsCategories

const categoryLabelByKey = new Map<SettingSearchKey, string>(
	Object.values(settingsCategories).flatMap((category) =>
		category.keys.map((key) => [key, category.label] as const),
	),
)

// settingWords returns the words of a setting with its category label, so that
// searching for a category shows all of its settings.
export function settingWords(key: SettingSearchKey): string {
	return `${categoryLabelByKey.get(key) ?? ""} ${settingSearchWords[key]}`
}

export function matchesSetting(terms: string[], key: SettingSearchKey): boolean {
	return matchesSearch(terms, settingWords(key))
}

// matchesCategory keeps a category in the sidebar when the search finds any of
// its settings.
export function matchesCategory(
	terms: string[],
	categoryId: SettingsCategoryId,
): boolean {
	return settingsCategories[categoryId].keys.some((key) => matchesSetting(terms, key))
}

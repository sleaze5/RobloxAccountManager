import { createContext } from "svelte"

export function robloxUserProfileUrl(userId: number | string): string {
	return `https://www.roblox.com/users/${userId}/profile`
}

export function robloxGroupUrl(groupId: number): string {
	return `https://www.roblox.com/communities/${groupId}`
}

export function robloxGameUrl(placeId: number): string {
	return `https://www.roblox.com/games/${placeId}`
}

export function robloxCatalogItemUrl(assetId: number): string {
	return `https://www.roblox.com/catalog/${assetId}`
}

export function robloxBundleUrl(bundleId: number): string {
	return `https://www.roblox.com/bundles/${bundleId}`
}

export function robloxBadgeUrl(badgeId: number): string {
	return `https://www.roblox.com/badges/${badgeId}`
}

export function robloxGamePassUrl(gamePassId: number): string {
	return `https://www.roblox.com/game-pass/${gamePassId}`
}

// RobloxPageOpener opens a Roblox page in the managed browser of the selected
// account, which is signed in, so that the page shows what the account sees.
// The browser needs a saved account, so canOpen is false without one.
interface RobloxPageOpener {
	readonly canOpen: boolean
	open: (pageUrl: string) => void
}

export const openInBrowserUnavailableReason = "Add an account to open Roblox pages"

export const [getRobloxPageOpener, setRobloxPageOpener] =
	createContext<RobloxPageOpener>()

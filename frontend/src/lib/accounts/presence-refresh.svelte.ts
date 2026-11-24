import { untrack } from "svelte"
import { appSettings } from "../settings/settings-store.svelte"
import type { AccountStore } from "./account-store.svelte"

// refreshPresencesWhileShown reads the presence of the saved accounts at the
// interval of the presence settings while the Accounts page shows, and at
// once when it shows after the interval passed.
export function refreshPresencesWhileShown(
	store: AccountStore,
	isAccountsPageShown: () => boolean,
): void {
	$effect(() => {
		const presence = appSettings.presence.accounts,
			intervalMs = presence.intervalSeconds * 1000
		if (
			!isAccountsPageShown() ||
			!appSettings.initialized ||
			!store.vault.unlocked ||
			!presence.enabled
		) {
			return undefined
		}
		untrack(() => {
			if (Date.now() - store.presencesRefreshedAt >= intervalMs) {
				void store.refreshPresences()
			}
		})
		const timer = window.setInterval(
			() => void store.refreshPresences(),
			intervalMs,
		)
		return () => window.clearInterval(timer)
	})
}

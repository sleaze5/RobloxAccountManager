import { untrack } from "svelte"
import { appSettings } from "../settings/settings-store.svelte"
import type { AccountStore } from "./account-store.svelte"

// refreshOnlineStatusesWhileShown reads the online status of the saved accounts at the
// interval of the online status settings while the Accounts page shows, and at
// once when it shows after the interval passed.
export function refreshOnlineStatusesWhileShown(
	store: AccountStore,
	isAccountsPageShown: () => boolean,
): void {
	$effect(() => {
		const onlineStatus = appSettings.onlineStatus.accounts,
			intervalMs = onlineStatus.intervalSeconds * 1000
		if (
			!isAccountsPageShown() ||
			!appSettings.initialized ||
			!store.vault.unlocked ||
			!onlineStatus.enabled
		) {
			return undefined
		}
		untrack(() => {
			if (Date.now() - store.onlineStatusesRefreshedAt >= intervalMs) {
				void store.refreshOnlineStatuses()
			}
		})
		const timer = window.setInterval(
			() => void store.refreshOnlineStatuses(),
			intervalMs,
		)
		return () => window.clearInterval(timer)
	})
}

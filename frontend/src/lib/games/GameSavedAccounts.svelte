<script lang="ts">
	import starIcon from "@phosphor-icons/core/regular/star.svg?raw"
	import wrenchIcon from "@phosphor-icons/core/regular/wrench.svg?raw"
	import { AccountScope } from "../accounts/account-scope"
	import type { Account } from "../accounts/account-model"
	import { onlineStatusIcon, onlineStatusLabel } from "../accounts/account-model"
	import type { AccountStore } from "../accounts/account-store.svelte"
	import { isInGame } from "../accounts/online-status-place.svelte"
	import { accountBackend } from "../backend/bridge"
	import type { GamePlace, SavedAccountsGameFavorites } from "../backend/bridge"
	import type { LabeledBadge } from "../shared/labeled-badge"
	import SavedAccountRelations from "../shared/SavedAccountRelations.svelte"
	import type { SavedAccountRelation } from "../shared/SavedAccountRelations.svelte"

	// GameSavedAccounts lists the other saved accounts that marked the game as
	// a favorite, play it now, or created it. refreshCount reads the favorites
	// again when the game details refresh.
	let {
		accounts,
		place,
		refreshCount,
	}: { accounts: AccountStore; place: GamePlace; refreshCount: number } = $props()

	let gameFavorites = $state.raw<SavedAccountsGameFavorites | null>(null)

	const selectedAccountId = $derived(accounts.selectedAccount.id),
		universeId = $derived(place.universeId),
		favoriteBadge: LabeledBadge = {
			label: "Favorite",
			icon: starIcon,
			tooltip: "This account marked this game as a favorite on Roblox",
			isStrong: false,
		},
		creatorBadge: LabeledBadge = {
			label: "Creator",
			icon: wrenchIcon,
			tooltip: "This account created this game",
			isStrong: true,
		}

	$effect(() => {
		const scope = new AccountScope(selectedAccountId, accounts),
			readUniverseId = universeId
		void refreshCount
		gameFavorites = null
		if (readUniverseId <= 0 || !scope.isCurrent || scope.accountId <= 0) return
		void (async () => {
			try {
				const favorites = await scope.track(
					accountBackend.GetSavedAccountsGameFavorites(
						scope.accountId,
						readUniverseId,
					),
				)
				if (scope.isCurrent) gameFavorites = favorites
			} catch {
				// The backend logs the failure. The section still shows the
				// accounts that play or created the game.
			}
		})()
		return () => scope.dispose()
	})

	function relationBadges(account: Account): LabeledBadge[] {
		const isPlaying =
			isInGame(account.onlineStatus) &&
			account.onlineStatus.universeId === universeId
		return [
			...(place.creatorType === "User" &&
			String(place.creatorId) === account.userId
				? [creatorBadge]
				: []),
			...(isPlaying
				? [
						{
							label: onlineStatusLabel(account.onlineStatus),
							icon: onlineStatusIcon(account.onlineStatus),
							isStrong: true,
						},
					]
				: []),
			...(gameFavorites?.accountIds?.includes(account.id) ? [favoriteBadge] : []),
		]
	}

	const relations = $derived(
		accounts.accounts.flatMap((account): SavedAccountRelation[] => {
			if (account.id === selectedAccountId) return []
			const badges = relationBadges(account)
			return badges.length > 0 ? [{ account, badges }] : []
		}),
	)
</script>

<SavedAccountRelations
	{relations}
	failedAccountCount={gameFavorites?.failedAccountCount ?? 0} />

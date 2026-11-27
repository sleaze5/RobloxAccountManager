<script lang="ts">
	import checkCircleIcon from "@phosphor-icons/core/regular/check-circle.svg?raw"
	import starIcon from "@phosphor-icons/core/regular/star.svg?raw"
	import tShirtIcon from "@phosphor-icons/core/regular/t-shirt.svg?raw"
	import type { Account } from "../accounts/account-model"
	import type {
		SavedAccountItemRelation,
		SavedAccountsItemRelations,
	} from "../backend/bridge"
	import type { LabeledBadge } from "../shared/labeled-badge"
	import SavedAccountRelations from "../shared/SavedAccountRelations.svelte"
	import type { SavedAccountRelation } from "../shared/SavedAccountRelations.svelte"

	// ItemSavedAccounts lists the other saved accounts that own, wear, or
	// marked the item as a favorite.
	let {
		otherAccounts,
		itemRelations,
	}: {
		// The saved accounts other than the one that the dialog acts for.
		otherAccounts: Account[]
		// undefined while it loads, and null when it failed.
		itemRelations: SavedAccountsItemRelations | null | undefined
	} = $props()

	const ownedBadge: LabeledBadge = {
			label: "Owned",
			icon: checkCircleIcon,
			tooltip: "This account owns this item",
			isStrong: true,
		},
		wornBadge: LabeledBadge = {
			label: "Worn",
			icon: tShirtIcon,
			tooltip: "This account wears this item now",
			isStrong: true,
		},
		favoriteBadge: LabeledBadge = {
			label: "Favorite",
			icon: starIcon,
			tooltip: "This account marked this item as a favorite on Roblox",
			isStrong: false,
		}

	function relationBadges(relation: SavedAccountItemRelation): LabeledBadge[] {
		return [
			...(relation.isOwned ? [ownedBadge] : []),
			...(relation.isWorn ? [wornBadge] : []),
			...(relation.isFavorite ? [favoriteBadge] : []),
		]
	}

	const relations = $derived(
		(itemRelations?.relations ?? []).flatMap((relation): SavedAccountRelation[] => {
			const account = otherAccounts.find(
				(otherAccount) => otherAccount.id === relation.accountId,
			)
			return account ? [{ account, badges: relationBadges(relation) }] : []
		}),
	)
</script>

<SavedAccountRelations
	{relations}
	failedAccountCount={itemRelations?.failedAccountCount ?? 0} />

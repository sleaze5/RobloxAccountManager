<script lang="ts">
	import type { FriendConnection } from "../backend/bridge"
	import type { AccountStore } from "../accounts/account-store.svelte"
	import SavedAccountRelations from "../shared/SavedAccountRelations.svelte"
	import type { SavedAccountRelation } from "../shared/SavedAccountRelations.svelte"
	import { relationshipBadges } from "./relationship"

	let {
		connections,
		accounts,
	}: { connections: FriendConnection[]; accounts: AccountStore } = $props()

	const relations = $derived(
		connections.flatMap((connection): SavedAccountRelation[] => {
			const account = accounts.accounts.find(
				(savedAccount) => savedAccount.id === connection.accountId,
			)
			return account ? [{ account, badges: relationshipBadges(connection) }] : []
		}),
	)
</script>

<SavedAccountRelations {relations} />

<script lang="ts">
	import type { Account } from "../accounts/account-model"
	import AccountAvatar from "../accounts/AccountAvatar.svelte"

	// RobloxAccountLine names the saved account that makes a change on Roblox,
	// in the confirmation of the change.
	let { account }: { account: Account } = $props()
</script>

<p
	class="flex min-w-0 items-center gap-2 rounded-md border border-line bg-raised px-2 py-1.5 text-xs text-fg-muted">
	<AccountAvatar {account} className="size-5 text-2xs" />
	<span class="min-w-0 truncate"
		>On Roblox as <span class="font-medium text-fg">{account.displayName}</span>
		@{account.username}</span>
</p>

<script lang="ts">
	import type { Snippet } from "svelte"
	import type { Account } from "../accounts/account-model"
	import AccountAvatar from "../accounts/AccountAvatar.svelte"

	// AccountConfirmationActions is the bottom row of a confirmation that
	// changes Roblox or spends Robux: the saved account that acts, and the
	// actions. details adds a second line under the username, such as the
	// Robux balance.
	let {
		account,
		details,
		children,
	}: { account: Account; details?: Snippet; children: Snippet } = $props()
</script>

<div
	class="mt-3 flex items-center justify-between gap-3 border-t border-line-subtle pt-3">
	<div class="flex min-w-0 items-center gap-2">
		<AccountAvatar {account} className="size-7" />
		<div class="flex min-w-0 flex-col leading-tight">
			<span class="truncate text-xs font-medium text-fg"
				>@{account.username}</span>
			{@render details?.()}
		</div>
	</div>
	<div class="flex shrink-0 gap-2">{@render children()}</div>
</div>

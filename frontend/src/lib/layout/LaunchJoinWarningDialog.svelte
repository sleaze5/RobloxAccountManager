<script lang="ts">
	import prohibitIcon from "@phosphor-icons/core/regular/prohibit.svg?raw"
	import xIcon from "@phosphor-icons/core/regular/x.svg?raw"
	import type { AccountJoinCheck } from "../backend/bridge"
	import type { AccountStore } from "../accounts/account-store.svelte"
	import Dialog from "../shared/Dialog.svelte"
	import Icon from "../shared/Icon.svelte"

	// LaunchJoinWarningDialog stops a launch that Roblox would likely refuse
	// and lets the user launch anyway. Roblox decides only when the game opens.
	let {
		restrictions,
		store,
		onLaunchAnyway,
		onCancel,
	}: {
		restrictions: AccountJoinCheck[]
		store: AccountStore
		onLaunchAnyway: () => void
		onCancel: () => void
	} = $props()

	const dialogId = $props.id(),
		isSeveral = $derived(store.selectedAccountIds.length > 1),
		title = $derived(
			isSeveral
				? `${restrictions.length === 1 ? "1 account" : `${restrictions.length} accounts`} will likely not get in`
				: "This account will likely not get in",
		)

	function accountLabel(accountId: number): string {
		const account = store.accounts.find(
			(savedAccount) => savedAccount.id === accountId,
		)
		return account ? `@${account.username}` : "An account"
	}
</script>

<svelte:window
	onkeydown={(event) => {
		if (event.key === "Escape") {
			event.preventDefault()
			onCancel()
		}
	}} />

<Dialog
	class="max-w-md text-left"
	role="alertdialog"
	aria-labelledby={`${dialogId}-title`}
	aria-describedby={`${dialogId}-description`}
	onBackdropClick={onCancel}>
	<div class="dialog-heading mb-2">
		<div class="min-w-0">
			<h2 id={`${dialogId}-title`}>{title}</h2>
			<p id={`${dialogId}-description`}>
				Roblox's join check expects {restrictions.length === 1
					? "this join"
					: "these joins"} to fail. You can still launch and try.
			</p>
		</div>
		<button type="button" aria-label="Close dialog" onclick={onCancel}>
			<Icon svg={xIcon} size={14} aria-hidden="true" />
		</button>
	</div>
	<ul class="mb-3 flex flex-col gap-1.5 text-xs text-fg select-text">
		{#each restrictions as accountCheck (accountCheck.accountId)}
			<li class="flex items-start gap-1.5">
				<Icon
					svg={prohibitIcon}
					size={14}
					class="mt-px shrink-0 text-warning"
					aria-hidden="true" />
				<span
					>{#if isSeveral}<strong class="font-semibold"
							>{accountLabel(accountCheck.accountId)}:</strong>
					{/if}{accountCheck.check.message}</span>
			</li>
		{/each}
	</ul>
	<div class="dialog-actions">
		<button class="button" type="button" onclick={onCancel}>Cancel</button>
		<button class="button-warning" type="button" onclick={onLaunchAnyway}
			>Launch anyway</button>
	</div>
</Dialog>

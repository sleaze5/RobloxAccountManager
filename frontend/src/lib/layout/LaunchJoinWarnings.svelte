<script lang="ts">
	import circleNotchIcon from "@phosphor-icons/core/regular/circle-notch.svg?raw"
	import hourglassIcon from "@phosphor-icons/core/regular/hourglass.svg?raw"
	import infoIcon from "@phosphor-icons/core/regular/info.svg?raw"
	import prohibitIcon from "@phosphor-icons/core/regular/prohibit.svg?raw"
	import Icon from "../shared/Icon.svelte"
	import { JoinOutcome } from "../backend/bridge"
	import type { AccountStore } from "../accounts/account-store.svelte"
	import type { LaunchJoinCheck } from "./launch-join-check.svelte"

	// LaunchJoinWarnings tells, before a launch, why an account would likely
	// not get in. It only warns: the Launch button stays enabled.
	let { joinCheck, store }: { joinCheck: LaunchJoinCheck; store: AccountStore } =
		$props()

	const warnings = $derived(
			joinCheck.checks.filter((accountCheck) => accountCheck.check.message),
		),
		isSeveral = $derived(store.selectedAccountIds.length > 1),
		hasRestriction = $derived(
			warnings.some(
				(accountCheck) =>
					accountCheck.check.outcome === JoinOutcome.JoinOutcomeRestricted,
			),
		)

	function accountLabel(accountId: number): string {
		const account = store.accounts.find(
			(savedAccount) => savedAccount.id === accountId,
		)
		return account ? `@${account.username}` : "An account"
	}

	function outcomeIcon(outcome: JoinOutcome): string {
		switch (outcome) {
			case JoinOutcome.JoinOutcomeRestricted:
				return prohibitIcon
			case JoinOutcome.JoinOutcomeWaiting:
				return hourglassIcon
			default:
				return infoIcon
		}
	}
</script>

{#if joinCheck.isChecking}
	<p class="flex items-center gap-1.5 text-xs text-fg-muted" role="status">
		<Icon svg={circleNotchIcon} class="spinner" size={14} aria-hidden="true" />
		Checking whether {isSeveral ? "the accounts" : "the account"} can join…
	</p>
{:else if warnings.length > 0}
	<div
		class={[
			"flex max-w-lg animate-enter flex-col gap-1",
			hasRestriction ? "notice-warning" : "notice-note",
		]}
		role="status">
		<ul class="flex flex-col gap-1" aria-label="Join warnings">
			{#each warnings as accountCheck (accountCheck.accountId)}
				<li class="flex items-start gap-1.5">
					<Icon
						svg={outcomeIcon(accountCheck.check.outcome)}
						size={14}
						class="mt-px shrink-0"
						aria-hidden="true" />
					<span
						>{#if isSeveral}<strong class="font-semibold"
								>{accountLabel(accountCheck.accountId)}:</strong>
						{/if}{accountCheck.check.message}</span>
				</li>
			{/each}
		</ul>
		{#if hasRestriction}
			<p class="text-fg-muted">
				You can still launch. Roblox decides when the game opens.
			</p>
		{/if}
	</div>
{/if}

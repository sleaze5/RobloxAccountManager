<script lang="ts">
	import circleNotchIcon from "@phosphor-icons/core/regular/circle-notch.svg?raw"
	import coinsIcon from "@phosphor-icons/core/regular/coins.svg?raw"
	import Icon from "../shared/Icon.svelte"
	import Select from "../shared/Select.svelte"
	import Timestamp from "../shared/Timestamp.svelte"
	import { RobuxTransactionType } from "../backend/bridge"
	import type { RobuxTransaction } from "../backend/bridge"
	import type { RobuxHistoryState } from "./robux-history-state.svelte"

	let { history }: { history: RobuxHistoryState } = $props()

	const numbers = new Intl.NumberFormat("en-US"),
		typeOptions = [
			{ value: RobuxTransactionType.RobuxPurchases, label: "Purchases" },
			{ value: RobuxTransactionType.RobuxSales, label: "Sales" },
			{ value: RobuxTransactionType.RobuxBought, label: "Robux bought" },
			{
				value: RobuxTransactionType.RobuxPremiumStipends,
				label: "Premium stipends",
			},
			{ value: RobuxTransactionType.RobuxGroupPayouts, label: "Group payouts" },
		],
		// Roblox item types in transaction details.
		itemTypes: Record<string, string> = {
			GamePass: "Game pass",
			DeveloperProduct: "Developer product",
			PrivateServer: "Private server",
			Asset: "Item",
			Bundle: "Bundle",
		}

	function title(transaction: RobuxTransaction): string {
		return transaction.itemName || transaction.agentName || "Robux transaction"
	}

	function detail(transaction: RobuxTransaction): string {
		return [
			itemTypes[transaction.itemType] ?? "",
			transaction.placeName,
			transaction.itemName ? transaction.agentName : "",
		]
			.filter(Boolean)
			.join(" - ")
	}
</script>

{#if history.error}
	<div class="notice-error" role="alert">{history.error}</div>
{/if}

<section class="settings-section" aria-labelledby="robux-history-title">
	<h3 class="flex items-center justify-between gap-2">
		<span id="robux-history-title">Transactions</span>
		<Select
			label="Transaction type"
			value={history.transactionType}
			options={typeOptions}
			className="w-40"
			disabled={history.loading}
			onChange={(transactionType) => history.selectType(transactionType)} />
	</h3>
	{#if !history.transactions}
		{#if history.loading}
			<div class="flex items-center gap-2 text-xs text-fg-muted" role="status">
				<Icon
					svg={circleNotchIcon}
					class="spinner"
					size={16}
					aria-hidden="true" />
				Loading transactions…
			</div>
		{/if}
	{:else if history.transactions.length === 0}
		<div class="empty-state">
			<Icon svg={coinsIcon} size={28} aria-hidden="true" />
			<h4>No transactions</h4>
			<p>Roblox has no transactions of this type for this account.</p>
		</div>
	{:else}
		<ul class="settings-rows" aria-label="Transactions" aria-busy={history.loading}>
			{#each history.transactions as transaction, index (`${transaction.id}-${index}`)}
				<li class="settings-row animate-enter justify-start gap-3">
					<div class="settings-row-copy">
						<strong class="flex min-w-0 items-center gap-1.5">
							<span class="truncate">{title(transaction)}</span>
							{#if transaction.isPending}<span class="shrink-0 server-tag"
									>Pending</span
								>{/if}
						</strong>
						{#if detail(transaction)}<span class="truncate"
								>{detail(transaction)}</span
							>{/if}
					</div>
					<span class="ml-auto flex shrink-0 flex-col items-end gap-0.5">
						<span class="font-mono text-xs text-fg tabular-nums"
							>{transaction.amount > 0 ? "+" : ""}{numbers.format(
								transaction.amount,
							)} R$</span>
						{#if transaction.createdAtMs}<span class="text-2xs"
								><Timestamp value={transaction.createdAtMs} /></span
							>{/if}
					</span>
				</li>
			{/each}
		</ul>
		{#if history.nextCursor}
			<button
				class="button self-start"
				type="button"
				aria-busy={history.loadingMore}
				disabled={history.loadingMore || history.loading}
				onclick={() => void history.loadMore()}>
				{history.loadingMore ? "Loading…" : "Load more"}
			</button>
		{/if}
	{/if}
</section>

<script lang="ts">
	import caretLeftIcon from "@phosphor-icons/core/regular/caret-left.svg?raw"
	import checkCircleIcon from "@phosphor-icons/core/regular/check-circle.svg?raw"
	import funnelIcon from "@phosphor-icons/core/regular/funnel.svg?raw"
	import medalIcon from "@phosphor-icons/core/regular/medal.svg?raw"
	import userPlusIcon from "@phosphor-icons/core/regular/user-plus.svg?raw"
	import warningIcon from "@phosphor-icons/core/regular/warning.svg?raw"
	import { onMount, untrack } from "svelte"
	import type { AccountStore } from "../accounts/account-store.svelte"
	import CompactNumber from "../shared/CompactNumber.svelte"
	import Icon from "../shared/Icon.svelte"
	import ProfileImage from "../shared/ProfileImage.svelte"
	import RefreshIcon from "../shared/RefreshIcon.svelte"
	import Skeleton from "../shared/Skeleton.svelte"
	import SkeletonList from "../shared/SkeletonList.svelte"
	import SkeletonMediaRow from "../shared/SkeletonMediaRow.svelte"
	import Timestamp from "../shared/Timestamp.svelte"
	import type { GamesStore } from "./games-store.svelte"

	let {
		store,
		accounts,
		onBack,
	}: {
		store: GamesStore
		accounts: AccountStore
		onBack: () => void
	} = $props()

	type BadgeFilter = "all" | "earned" | "unearned"

	const place = untrack(() => store.selected!),
		universeBadges = untrack(() => store.badgesFor(place.universeId)),
		filterID = $props.id(),
		numbers = new Intl.NumberFormat(),
		percent = new Intl.NumberFormat(undefined, {
			style: "percent",
			maximumFractionDigits: 1,
		})
	let heading = $state<HTMLHeadingElement | undefined>(undefined),
		filter = $state<BadgeFilter>("all")
	const account = $derived(accounts.selectedAccount),
		badges = $derived(universeBadges.badges ?? []),
		awards = $derived(universeBadges.awardedAtByBadgeId),
		// Disabled badges cannot be earned anymore, so they do not count toward
		// completion.
		enabledBadges = $derived(badges.filter((badge) => badge.enabled)),
		earnedCount = $derived(
			awards
				? enabledBadges.filter((badge) => awards.has(badge.badgeId)).length
				: 0,
		),
		completion = $derived(
			enabledBadges.length ? earnedCount / enabledBadges.length : 0,
		),
		shownBadges = $derived(
			!awards || filter === "all"
				? badges
				: badges.filter(
						(badge) => awards.has(badge.badgeId) === (filter === "earned"),
					),
		),
		filterOptions: { id: BadgeFilter; label: string }[] = [
			{ id: "all", label: "All" },
			{ id: "earned", label: "Earned" },
			{ id: "unearned", label: "Not earned" },
		]

	// The account list replaces its accounts when it refreshes, so the effect
	// reads the ID alone and runs only when another account is selected. It runs
	// before the first load, so that the load requests the shown account.
	const accountId = $derived(account.id)
	$effect(() => {
		const shownAccountId = accountId
		untrack(() => universeBadges.showAccount(shownAccountId))
	})

	onMount(() => {
		heading?.focus({ preventScroll: true })
		void universeBadges.load()
	})
</script>

<section class="settings-page min-w-0" aria-labelledby="game-badges-title">
	<header class="settings-header">
		<button
			class="button-ghost"
			type="button"
			aria-label="Back to game details"
			data-tooltip="Back to game details"
			onclick={onBack}>
			<Icon svg={caretLeftIcon} data-nudge="left" size={16} aria-hidden="true" />
		</button>
		<h1 id="game-badges-title" tabindex="-1" bind:this={heading}>Badges</h1>
		<span class="mr-auto min-w-0 text-xs wrap-anywhere text-fg-muted"
			>{store.displayName(place)}</span>
		<button
			class="button-ghost"
			type="button"
			aria-label={universeBadges.loading ? "Refreshing badges" : "Refresh badges"}
			data-tooltip="Refresh badges"
			data-tooltip-side="bottom-end"
			aria-busy={universeBadges.loading || universeBadges.loadingAwards}
			disabled={universeBadges.loading || universeBadges.loadingAwards}
			onclick={() => void universeBadges.refresh()}>
			<RefreshIcon
				isRefreshing={universeBadges.loading || universeBadges.loadingAwards} />
		</button>
	</header>

	<div
		class="flex flex-wrap items-end gap-x-3 gap-y-2 border-b border-line-subtle p-3"
		role="group"
		aria-label="Badge filters">
		<div class="flex flex-col gap-1 text-2xs text-fg-muted">
			<span id={`${filterID}-filter`}>Show</span>
			<div
				class="segmented"
				role="radiogroup"
				aria-labelledby={`${filterID}-filter`}>
				{#each filterOptions as option (option.id)}
					<label>
						<input
							type="radio"
							name={`${filterID}-filter`}
							value={option.id}
							checked={filter === option.id}
							disabled={!awards}
							onchange={() => (filter = option.id)} />
						<span>{option.label}</span>
					</label>
				{/each}
			</div>
		</div>
		{#if account.id !== 0 && badges.length > 0}
			<div
				class="ml-auto flex w-48 flex-col gap-1 text-2xs text-fg-muted"
				role="status"
				aria-busy={universeBadges.loadingAwards && !awards}>
				<span class="flex items-baseline justify-between gap-2">
					<span>Completion</span>
					{#if awards}
						<span class="font-mono tabular-nums"
							><strong class="font-semibold text-fg"
								>{percent.format(completion)}</strong>
							- {numbers.format(earnedCount)} of {numbers.format(
								enabledBadges.length,
							)}</span>
					{:else if universeBadges.awardsError}
						<span>Unavailable</span>
					{:else}
						<Skeleton className="w-20" label="Loading completion" />
					{/if}
				</span>
				<div
					class="progress-bar"
					role="progressbar"
					aria-label={`Badges earned by @${account.username}`}
					aria-valuemin={0}
					aria-valuemax={enabledBadges.length}
					aria-valuenow={earnedCount}>
					<div style:transform="scaleX({completion})"></div>
				</div>
			</div>
		{/if}
	</div>

	{#if universeBadges.awardsError && universeBadges.badges}
		<div class="mx-3 mt-3 flex items-center gap-3 notice-error" role="alert">
			<span class="flex-1">{universeBadges.awardsError}</span>
			<button
				class="button"
				type="button"
				onclick={() => universeBadges.reloadAwards()}>Try again</button>
		</div>
	{/if}

	<div class="min-h-0 flex-1 overflow-auto">
		{#if universeBadges.error && !universeBadges.badges}
			<div class="empty-state h-full" role="alert">
				<Icon svg={warningIcon} size={28} aria-hidden="true" />
				<h2>Badges are unavailable</h2>
				<p>{universeBadges.error}</p>
				<button
					class="button"
					type="button"
					onclick={() => void universeBadges.refresh()}>Try again</button>
			</div>
		{:else if !universeBadges.badges}
			<SkeletonList label="Loading badges">
				{#snippet row(index)}<SkeletonMediaRow
						{index}
						className="h-16" />{/snippet}
			</SkeletonList>
		{:else if badges.length === 0}
			<div class="empty-state h-full" role="status">
				<Icon svg={medalIcon} size={28} aria-hidden="true" />
				<h2>No badges</h2>
				<p>This game has no badges to earn.</p>
			</div>
		{:else if shownBadges.length === 0}
			<div class="empty-state h-full" role="status">
				<Icon svg={funnelIcon} size={28} aria-hidden="true" />
				<h2>
					{filter === "earned" ? "No earned badges" : "All badges earned"}
				</h2>
				<p>
					{filter === "earned"
						? `@${account.username} has not earned a badge in this game yet.`
						: `@${account.username} has earned every badge in this game.`}
				</p>
				<button class="button" type="button" onclick={() => (filter = "all")}
					>Show all badges</button>
			</div>
		{:else}
			{#if account.id === 0}
				<p
					class="flex items-center gap-2 border-b border-line-subtle px-3 py-2 text-xs text-fg-muted">
					<Icon svg={userPlusIcon} size={14} aria-hidden="true" />
					Add an account on the Accounts page to see which badges it earned.
				</p>
			{/if}
			<ol aria-label="Badges" aria-busy={universeBadges.loading}>
				{#each shownBadges as badge (badge.badgeId)}
					{@const awardedAtMs = awards?.get(badge.badgeId)}
					<li
						class="flex min-h-16 items-center gap-3 border-b border-line-subtle px-3 py-2">
						<span
							class={[
								"grid size-9 shrink-0 place-items-center overflow-hidden rounded-md border border-line-strong bg-raised",
								!badge.enabled && "opacity-45",
							]}
							aria-hidden="true">
							<ProfileImage
								url={badge.iconUrl}
								fallbackIcon={medalIcon} />
						</span>
						<div class="flex min-w-0 flex-1 flex-col gap-0.5">
							<strong
								class="flex min-w-0 items-center gap-1.5 text-xs font-semibold text-fg">
								<span class="truncate"
									>{badge.name || "Unnamed badge"}</span>
								{#if !badge.enabled}<span class="shrink-0 server-tag"
										>Disabled</span
									>{/if}
							</strong>
							{#if badge.description}
								<span
									class="truncate text-xs text-fg-muted"
									data-tooltip={badge.description}
									>{badge.description}</span>
							{/if}
							<span
								class="flex flex-wrap items-center gap-x-1 font-mono text-2xs text-fg-faint">
								<span
									>Won by {percent.format(badge.pastDayWinFraction)} of
									players yesterday</span>
								<span aria-hidden="true">-</span>
								<span
									><CompactNumber
										value={badge.awardedCount}
										unit="awarded" /> awarded</span>
							</span>
						</div>
						{#if account.id !== 0}
							<div
								class="flex shrink-0 flex-col items-end gap-0.5 text-xs">
								{#if awardedAtMs !== undefined}
									<span
										class="inline-flex animate-pop items-center gap-1 font-medium text-success">
										<Icon
											svg={checkCircleIcon}
											size={14}
											aria-hidden="true" />
										Earned
									</span>
									{#if awardedAtMs}
										<span class="text-2xs"
											><Timestamp value={awardedAtMs} /></span>
									{/if}
								{:else if awards}
									<span class="text-fg-muted">Not earned</span>
								{:else if universeBadges.loadingAwards}
									<Skeleton
										className="w-14"
										label="Checking if earned" />
								{/if}
							</div>
						{/if}
					</li>
				{/each}
			</ol>
		{/if}
	</div>
</section>

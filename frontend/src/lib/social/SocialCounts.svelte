<script lang="ts">
	import { SocialList } from "../backend/bridge"
	import type { SocialListKind } from "./social-store.svelte"

	let {
		friends,
		followers,
		following,
		onShow,
	}: {
		friends: number | null
		followers: number | null
		following: number | null
		onShow: (list: SocialListKind) => void
	} = $props()

	const compact = new Intl.NumberFormat("en-US", {
			notation: "compact",
			maximumFractionDigits: 1,
		}),
		exact = new Intl.NumberFormat()

	const items = $derived(
		(
			[
				{
					list: SocialList.SocialFriends,
					label: friends === 1 ? "friend" : "friends",
					value: friends,
					noun: friends === 1 ? "friend" : "friends",
				},
				{
					list: SocialList.SocialFollowers,
					label: followers === 1 ? "follower" : "followers",
					value: followers,
					noun: followers === 1 ? "follower" : "followers",
				},
				{
					list: SocialList.SocialFollowing,
					label: "following",
					value: following,
					noun: following === 1 ? "followed user" : "followed users",
				},
			] as const
		).flatMap((count) =>
			count.value === null
				? []
				: [
						{
							...count,
							tooltip: `Show ${exact.format(count.value)} ${count.noun}`,
						},
					],
		),
	)
</script>

{#if items.length}
	<div
		class="flex flex-wrap items-baseline gap-x-4 gap-y-1"
		role="group"
		aria-label="Connections">
		{#each items as count (count.list)}
			<button
				class="group/count flex items-baseline gap-1 rounded-sm focus-visible:outline-offset-2"
				type="button"
				aria-label={count.tooltip}
				data-tooltip={count.tooltip}
				data-tooltip-side="top"
				onclick={() => onShow(count.list)}>
				<span class="text-sm font-semibold text-fg tabular-nums"
					>{compact.format(count.value ?? 0)}</span>
				<span
					class="text-xs text-fg-muted transition-colors duration-100 group-hover/count:text-fg"
					>{count.label}</span>
			</button>
		{/each}
	</div>
{/if}

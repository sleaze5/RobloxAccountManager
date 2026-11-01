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
					label: "Friends",
					value: friends,
					noun: "friends",
				},
				{
					list: SocialList.SocialFollowers,
					label: "Followers",
					value: followers,
					noun: "followers",
				},
				{
					list: SocialList.SocialFollowing,
					label: "Following",
					value: following,
					noun: "followed users",
				},
			] as const
		).flatMap((item) =>
			item.value === null
				? []
				: [
						{
							...item,
							tooltip: `Show ${exact.format(item.value)} ${item.noun}`,
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
		{#each items as item (item.list)}
			<button
				class="group/count flex items-baseline gap-1 rounded-sm focus-visible:outline-offset-2"
				type="button"
				aria-label={item.tooltip}
				data-tooltip={item.tooltip}
				data-tooltip-side="top"
				onclick={() => onShow(item.list)}>
				<span class="text-sm font-semibold text-fg tabular-nums"
					>{compact.format(item.value ?? 0)}</span>
				<span
					class="text-xs text-fg-muted transition-colors duration-100 group-hover/count:text-fg"
					>{item.label}</span>
			</button>
		{/each}
	</div>
{/if}

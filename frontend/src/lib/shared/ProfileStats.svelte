<script lang="ts" module>
	export interface ProfileStat {
		// id tells the stats apart, because a label changes with its count.
		id: string
		// value is the short form of the number, such as "2.4K" or "94%", and
		// is empty while it loads.
		value: string
		// label follows the value, such as "followers" or "liked".
		label: string
		// description is the full form, such as "2,431 followers", which the
		// tooltip and the name of a button give.
		description: string
		isLoading?: boolean
		// isLive marks a count of players who are in the game now.
		isLive?: boolean
		// onShow makes the stat a button that shows its list, such as the
		// friends of a user.
		onShow?: () => void
	}
</script>

<script lang="ts">
	import Skeleton from "./Skeleton.svelte"

	// ProfileStats shows the headline numbers of a profile as one line of text,
	// each number followed by its label. A stat that is still loading holds
	// its place with a placeholder.
	let { label, stats }: { label: string; stats: ProfileStat[] } = $props()
</script>

{#snippet content(stat: ProfileStat)}
	<span
		class={[
			"flex items-center gap-1.5 text-sm font-semibold tabular-nums",
			stat.isLive ? "text-presence-game" : "text-fg",
		]}>
		{#if stat.isLive}<span
				class="size-2 shrink-0 animate-pop rounded-full bg-presence-game"
				aria-hidden="true"></span
			>{/if}
		{stat.value}
	</span>
	<span
		class="text-xs text-fg-muted transition-colors duration-100 group-hover/stat:text-fg"
		>{stat.label}</span>
{/snippet}

{#if stats.length}
	<div
		class="flex min-w-0 flex-wrap items-baseline gap-x-4 gap-y-1"
		role="group"
		aria-label={label}>
		{#each stats as stat (stat.id)}
			{#if stat.isLoading && !stat.value}
				<span class="flex items-baseline gap-1">
					<span class="text-sm"
						><Skeleton
							className="w-6"
							label={`Loading ${stat.label}…`} /></span>
					<span class="text-xs text-fg-muted">{stat.label}</span>
				</span>
			{:else if stat.onShow}
				<button
					class="group/stat flex items-baseline gap-1 rounded-sm focus-visible:outline-offset-2"
					type="button"
					aria-label={`Show ${stat.description}`}
					data-tooltip={`Show ${stat.description}`}
					data-tooltip-side="top"
					onclick={stat.onShow}>
					{@render content(stat)}
				</button>
			{:else}
				<span
					class="flex items-baseline gap-1"
					data-tooltip={stat.description}
					data-tooltip-side="top">
					{@render content(stat)}
				</span>
			{/if}
		{/each}
	</div>
{/if}

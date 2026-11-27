<script lang="ts">
	import clockIcon from "@phosphor-icons/core/regular/clock.svg?raw"
	import lockIcon from "@phosphor-icons/core/regular/lock.svg?raw"
	import starFillIcon from "@phosphor-icons/core/fill/star-fill.svg?raw"
	import Icon from "../shared/Icon.svelte"

	// GroupStatusBadges shows the states of a group for the account that the
	// facts do not show. The role of the account is a fact.
	let {
		isPrimary,
		isPendingJoin,
		isLocked,
	}: { isPrimary: boolean; isPendingJoin: boolean; isLocked: boolean } = $props()
</script>

{#if isPrimary || isPendingJoin || isLocked}
	<ul class="mt-1 flex min-h-6 flex-wrap gap-1" aria-label="Status">
		{#if isPrimary}
			<li class="tag-pill-favorite">
				<Icon svg={starFillIcon} size={12} aria-hidden="true" />
				<span>Primary group</span>
			</li>
		{/if}
		{#if isPendingJoin}
			<li class="tag-pill">
				<Icon svg={clockIcon} size={12} aria-hidden="true" />
				<span>Join request sent</span>
			</li>
		{/if}
		{#if isLocked}
			<li class="tag-pill-warning">
				<Icon svg={lockIcon} size={12} aria-hidden="true" />
				<span>Locked by Roblox</span>
			</li>
		{/if}
	</ul>
{/if}

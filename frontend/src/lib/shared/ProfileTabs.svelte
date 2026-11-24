<script lang="ts" module>
	export interface ProfileTab<TabId extends string = string> {
		// id names the page that the tab opens. The tab carries it in
		// data-subpage-button, so that closing the page can focus the tab.
		id: TabId
		label: string
		unreadCount?: number
		// unavailableReason explains why the page cannot open, such as
		// "Inventory is private".
		unavailableReason?: string
	}
</script>

<script lang="ts" generics="TabId extends string">
	// ProfileTabs lists the pages of a profile below its header. Overview is
	// the profile itself, and every other tab opens its page in place of it.
	let {
		label,
		tabs,
		onOpen,
	}: {
		label: string
		tabs: ProfileTab<TabId>[]
		onOpen: (id: TabId) => void
	} = $props()
</script>

<nav
	class="-mx-4 flex h-8 min-w-0 overflow-x-auto border-b border-line-subtle px-1"
	aria-label={label}>
	<span class="tab" aria-current="page">Overview</span>
	{#each tabs as tab (tab.id)}
		<button
			class="tab aria-disabled:cursor-not-allowed aria-disabled:opacity-45 aria-disabled:hover:text-fg-muted"
			type="button"
			aria-label={tab.unavailableReason
				? `${tab.label}: ${tab.unavailableReason}`
				: tab.unreadCount
					? `${tab.label}, ${tab.unreadCount} unread`
					: undefined}
			aria-disabled={tab.unavailableReason ? true : undefined}
			data-tooltip={tab.unavailableReason || undefined}
			data-subpage-button={tab.id}
			onclick={() => {
				if (!tab.unavailableReason) onOpen(tab.id)
			}}>
			{tab.label}
			{#if tab.unreadCount}<span
					class="min-w-4 animate-bloom rounded-sm bg-solid px-0.5 text-center text-2xs leading-4 font-bold text-on-solid"
					aria-hidden="true"
					>{tab.unreadCount > 99 ? "99+" : tab.unreadCount}</span
				>{/if}
		</button>
	{/each}
</nav>

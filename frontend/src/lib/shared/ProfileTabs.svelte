<script lang="ts" module>
	export interface ProfileTab {
		// id names the page that the tab opens. The tab carries it in
		// data-subpage-button, so that closing the page can focus the tab.
		id: string
		label: string
		unreadCount?: number
		// unavailableReason explains why the page cannot open, such as
		// "Inventory is private".
		unavailableReason?: string
	}
</script>

<script lang="ts">
	import { getProfilePages } from "./profile-pages"

	// ProfileTabs shows the pages of the open profile: Overview, which is the
	// profile itself, and the pages that the profile provides. onLeave runs a
	// tab change through the open page, which can hold it, such as for changes
	// that are not saved.
	let {
		onLeave = (navigate) => navigate(),
	}: { onLeave?: (navigate: () => void) => void } = $props()

	const pages = getProfilePages()

	function tabLabel(tab: ProfileTab): string | undefined {
		if (tab.unavailableReason) return `${tab.label}: ${tab.unavailableReason}`
		if (tab.unreadCount) return `${tab.label}, ${tab.unreadCount} unread`
		return undefined
	}
</script>

<nav class="flex h-full min-w-0 overflow-x-auto" aria-label={pages.label}>
	{#if pages.currentPage}
		<button class="tab" type="button" onclick={() => onLeave(pages.openOverview)}
			>Overview</button>
	{:else}
		<span class="tab" aria-current="page">Overview</span>
	{/if}
	{#each pages.tabs as tab (tab.id)}
		{#if tab.id === pages.currentPage}
			<span class="tab" aria-current="page">{tab.label}</span>
		{:else}
			<button
				class="tab aria-disabled:cursor-not-allowed aria-disabled:opacity-45 aria-disabled:hover:text-fg-muted"
				type="button"
				aria-label={tabLabel(tab)}
				aria-disabled={tab.unavailableReason ? true : undefined}
				data-tooltip={tab.unavailableReason || undefined}
				data-subpage-button={tab.id}
				onclick={() => {
					if (!tab.unavailableReason) onLeave(() => pages.openPage(tab.id))
				}}>
				{tab.label}
				{#if tab.unreadCount}<span
						class="min-w-4 animate-bloom rounded-sm bg-solid px-0.5 text-center text-2xs leading-4 font-bold text-on-solid"
						aria-hidden="true"
						>{tab.unreadCount > 99 ? "99+" : tab.unreadCount}</span
					>{/if}
			</button>
		{/if}
	{/each}
</nav>

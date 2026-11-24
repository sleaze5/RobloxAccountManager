<script lang="ts">
	import { onMount } from "svelte"
	import type { Snippet } from "svelte"
	import ProfileTabs from "../shared/ProfileTabs.svelte"

	// AccountPageHeader heads a page of a profile, such as its avatar, with the
	// tabs of the profile in place of a Back button. The title stays for
	// screen readers, because the selected tab already shows it.
	let {
		titleId,
		title,
		username,
		onLeave,
		children,
	}: {
		titleId: string
		title: string
		username: string
		onLeave?: (navigate: () => void) => void
		children: Snippet
	} = $props()

	let heading = $state<HTMLHeadingElement>()

	onMount(() => {
		heading?.closest("[data-workspace-scroll]")?.scrollTo({ top: 0 })
		heading?.focus({ preventScroll: true })
	})
</script>

<header class="settings-header">
	<h1 id={titleId} class="sr-only" tabindex="-1" bind:this={heading}>
		{title} of @{username}
	</h1>
	<ProfileTabs {onLeave} />
	<span class="mr-auto min-w-0 truncate text-xs text-fg-muted">@{username}</span>
	{@render children()}
</header>

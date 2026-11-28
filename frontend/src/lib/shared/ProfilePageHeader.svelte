<script lang="ts">
	import { onMount } from "svelte"
	import type { Snippet } from "svelte"
	import ProfileTabs from "./ProfileTabs.svelte"

	// ProfilePageHeader heads a page of a profile, such as its avatar or the
	// members of a group, with the tabs of the profile in place of a Back
	// button. The title stays for screen readers, because the selected tab
	// already shows it, and the status strip names the profile. subtitle names
	// the profile for screen readers, such as "@builderman".
	let {
		titleId,
		title,
		subtitle,
		onLeave,
		children,
	}: {
		titleId: string
		title: string
		subtitle: string
		onLeave?: (navigate: () => void) => void
		children?: Snippet
	} = $props()

	let heading = $state<HTMLHeadingElement>()

	onMount(() => {
		heading?.closest("[data-workspace-scroll]")?.scrollTo({ top: 0 })
		heading?.focus({ preventScroll: true })
	})
</script>

<header class="tab-bar items-center gap-2 pr-3">
	<h1 id={titleId} class="sr-only" tabindex="-1" bind:this={heading}>
		{title} of {subtitle}
	</h1>
	<ProfileTabs {onLeave} />
	<span class="mr-auto"></span>
	{@render children?.()}
</header>

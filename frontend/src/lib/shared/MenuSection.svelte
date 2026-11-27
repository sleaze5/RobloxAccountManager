<script lang="ts">
	import type { Snippet } from "svelte"

	// accessibleLabel names the group for screen readers when the visible
	// heading needs its context, such as "Copy user details" for "Copy".
	let {
		heading,
		accessibleLabel,
		isBusy,
		children,
	}: {
		heading: string
		accessibleLabel?: string
		isBusy?: boolean
		children: Snippet
	} = $props()
</script>

<div role="group" aria-label={accessibleLabel ?? heading} aria-busy={isBusy}>
	<div class="menu-heading" aria-hidden="true">{heading}</div>
	{@render children()}
</div>

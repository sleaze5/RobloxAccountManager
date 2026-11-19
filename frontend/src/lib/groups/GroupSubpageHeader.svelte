<script lang="ts">
	import caretLeftIcon from "@phosphor-icons/core/regular/caret-left.svg?raw"
	import { onMount } from "svelte"
	import type { Snippet } from "svelte"
	import Icon from "../shared/Icon.svelte"

	let {
		titleId,
		title,
		groupName,
		onBack,
		children,
	}: {
		titleId: string
		title: string
		groupName: string
		onBack: () => void
		children?: Snippet
	} = $props()

	let heading = $state<HTMLHeadingElement>()

	onMount(() => heading?.focus({ preventScroll: true }))
</script>

<header class="settings-header">
	<button
		class="button-ghost"
		type="button"
		aria-label="Back to group"
		data-tooltip="Back to group"
		onclick={onBack}>
		<Icon svg={caretLeftIcon} data-nudge="left" size={16} aria-hidden="true" />
	</button>
	<h1 id={titleId} tabindex="-1" bind:this={heading}>{title}</h1>
	<span class="mr-auto min-w-0 truncate text-xs text-fg-muted">{groupName}</span>
	{@render children?.()}
</header>

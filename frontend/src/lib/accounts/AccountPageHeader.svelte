<script lang="ts">
	import caretLeftIcon from "@phosphor-icons/core/regular/caret-left.svg?raw"
	import { onMount } from "svelte"
	import type { Snippet } from "svelte"
	import Icon from "../shared/Icon.svelte"

	let {
		titleId,
		title,
		username,
		onBack,
		children,
	}: {
		titleId: string
		title: string
		username: string
		onBack: () => void
		children: Snippet
	} = $props()

	let heading = $state<HTMLHeadingElement>()

	onMount(() => {
		heading?.closest("[data-workspace-scroll]")?.scrollTo({ top: 0 })
		heading?.focus({ preventScroll: true })
	})
</script>

<header class="settings-header">
	<button
		class="button-ghost"
		type="button"
		aria-label="Back to profile"
		data-tooltip="Back to profile"
		onclick={onBack}>
		<Icon svg={caretLeftIcon} data-nudge="left" size={16} aria-hidden="true" />
	</button>
	<h1 id={titleId} tabindex="-1" bind:this={heading}>{title}</h1>
	<span class="mr-auto min-w-0 text-xs wrap-anywhere text-fg-muted">@{username}</span>
	{@render children()}
</header>

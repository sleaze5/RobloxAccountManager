<script lang="ts">
	import type { Snippet } from "svelte"
	import Icon from "../shared/Icon.svelte"
	import { menuIn, menuOut } from "../shared/transitions"

	let {
		title,
		triggerLabel,
		triggerIcon,
		isTriggerCompact,
		isTriggerHighlighted,
		isDisabled,
		onOpen,
		children,
	}: {
		title: string
		triggerLabel: string
		triggerIcon: string
		isTriggerCompact: boolean
		isTriggerHighlighted: boolean
		isDisabled: boolean
		onOpen: () => void
		children: Snippet<[closeAndFocusTrigger: () => void]>
	} = $props()
	let isOpen = $state(false),
		root = $state<HTMLDivElement>(),
		trigger = $state<HTMLButtonElement>()
	const panelID = $props.id()

	function closeAndFocusTrigger(): void {
		trigger?.focus()
		isOpen = false
	}

	$effect(() => {
		if (isDisabled) isOpen = false
	})
</script>

<svelte:window
	onclick={(event) => {
		if (event.target instanceof Node && !root?.contains(event.target))
			isOpen = false
	}}
	onkeydown={(event) => {
		if (isOpen && event.key === "Escape") {
			event.preventDefault()
			closeAndFocusTrigger()
		}
	}} />

<div class="relative" bind:this={root}>
	<button
		class={[
			isTriggerCompact ? "button-ghost-sm" : "button-ghost",
			isTriggerHighlighted && "text-accent-strong",
		]}
		type="button"
		bind:this={trigger}
		aria-label={triggerLabel}
		aria-expanded={isOpen}
		aria-controls={isOpen ? panelID : undefined}
		data-open={isOpen || undefined}
		data-tooltip={triggerLabel}
		data-tooltip-side="top"
		disabled={isDisabled}
		onclick={() => {
			isOpen = !isOpen
			if (isOpen) onOpen()
		}}>
		<Icon svg={triggerIcon} size={isTriggerCompact ? 14 : 16} aria-hidden="true" />
	</button>
	{#if isOpen}
		<section
			class="absolute right-0 bottom-full z-10 mb-2 flex w-80 flex-col gap-3 rounded-xl border border-line-strong bg-overlay p-3 shadow-menu"
			style="--menu-origin: bottom right"
			id={panelID}
			aria-label={title}
			in:menuIn
			out:menuOut>
			<strong class="text-xs font-semibold text-fg">{title}</strong>
			{@render children(closeAndFocusTrigger)}
		</section>
	{/if}
</div>

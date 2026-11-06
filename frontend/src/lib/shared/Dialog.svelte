<script lang="ts">
	import type { Snippet } from "svelte"
	import type { HTMLAttributes } from "svelte/elements"
	import {
		modalBackdropIn,
		modalBackdropOut,
		modalCardIn,
		modalCardOut,
		modalFocus,
	} from "./presence"

	let {
		card = $bindable(),
		children,
		class: className,
		role = "dialog",
		strong = false,
		onBackdropClick,
		...rest
	}: HTMLAttributes<HTMLDivElement> & {
		card?: HTMLDivElement
		children: Snippet
		role?: "dialog" | "alertdialog"
		strong?: boolean
		onBackdropClick?: () => void
	} = $props()
</script>

<div
	class={["dialog-backdrop", strong && "z-50 bg-scrim-strong"]}
	in:modalBackdropIn
	out:modalBackdropOut
	onclick={(event) => {
		if (event.target === event.currentTarget) onBackdropClick?.()
	}}
	role="presentation">
	<div
		{...rest}
		class={["dialog-card", className]}
		{role}
		aria-modal="true"
		bind:this={card}
		use:modalFocus
		in:modalCardIn
		out:modalCardOut>
		{@render children()}
	</div>
</div>

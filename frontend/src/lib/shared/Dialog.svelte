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
		hasFlexibleBody = false,
		onBackdropClick,
		...rest
	}: HTMLAttributes<HTMLDivElement> & {
		card?: HTMLDivElement
		children: Snippet
		role?: "dialog" | "alertdialog"
		strong?: boolean
		// hasFlexibleBody lays the content out as a column, so that a region
		// with min-h-0 shrinks and scrolls on its own while the heading and the
		// actions stay in view.
		hasFlexibleBody?: boolean
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
		<div class={["dialog-content", hasFlexibleBody && "flex flex-col"]}>
			{@render children()}
		</div>
	</div>
</div>

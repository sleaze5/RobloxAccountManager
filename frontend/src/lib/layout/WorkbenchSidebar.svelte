<script lang="ts">
	import caretLeftIcon from "@phosphor-icons/core/regular/caret-left.svg?raw"
	import caretRightIcon from "@phosphor-icons/core/regular/caret-right.svg?raw"
	import type { Snippet } from "svelte"
	import Icon from "../shared/Icon.svelte"
	import SidebarResizer from "./SidebarResizer.svelte"

	let {
		label,
		name,
		icon,
		count,
		collapsed = $bindable(),
		width = $bindable(),
		title,
		actions,
		menu,
		tools,
		children,
	}: {
		label: string
		name: string
		icon: string
		count?: number
		collapsed: boolean
		width: number
		title: Snippet
		actions?: Snippet
		menu?: Snippet
		tools: Snippet
		children: Snippet
	} = $props()
</script>

<aside
	class="flex min-h-0 w-(--sidebar-width) shrink-0 flex-col bg-surface"
	aria-label={label}>
	{#if collapsed}
		<div class="flex h-full flex-col items-center gap-3 pt-2 text-fg-faint">
			<button
				class="button-ghost"
				type="button"
				aria-label={`Expand ${name} sidebar`}
				data-tooltip={`Expand the ${name} sidebar`}
				data-tooltip-side="right"
				onclick={() => (collapsed = false)}
				><Icon
					svg={caretRightIcon}
					data-nudge="right"
					size={16}
					aria-hidden="true" /></button>
			<Icon svg={icon} size={16} aria-hidden="true" />
			{#if count !== undefined}<span class="font-mono text-2xs">{count}</span
				>{/if}
		</div>
	{:else}
		<div class="flex shrink-0 flex-col gap-1 px-3 pt-1 pb-2">
			<div class="relative flex h-7 items-center">
				<h1>{@render title()}</h1>
				<div class="ml-auto flex items-center gap-px">
					{@render actions?.()}
					<button
						class="button-ghost"
						type="button"
						aria-label={`Collapse ${name} sidebar`}
						data-tooltip={`Collapse the ${name} sidebar`}
						data-tooltip-side="bottom-end"
						onclick={() => (collapsed = true)}
						><Icon
							svg={caretLeftIcon}
							data-nudge="left"
							size={16}
							aria-hidden="true" /></button>
				</div>
				{@render menu?.()}
			</div>
			{@render tools()}
		</div>
		{@render children()}
	{/if}
</aside>

<SidebarResizer bind:width {collapsed} {name} />

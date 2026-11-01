<script lang="ts">
	import caretDoubleLeftIcon from "@phosphor-icons/core/regular/caret-double-left.svg?raw"
	import caretLeftIcon from "@phosphor-icons/core/regular/caret-left.svg?raw"
	import caretRightIcon from "@phosphor-icons/core/regular/caret-right.svg?raw"
	import Icon from "../shared/Icon.svelte"

	let {
		pager,
	}: {
		pager: {
			loading: boolean
			pageNumber: number
			hasNext: boolean
			first(): void
			previous(): void
			next(): void
		}
	} = $props()
</script>

<nav class="ml-auto flex items-center gap-1" aria-label="Server pages">
	<button
		class="btn-icon"
		type="button"
		aria-label="First page"
		data-tooltip="First page"
		disabled={pager.loading || pager.pageNumber < 2}
		onclick={() => pager.first()}>
		<Icon
			svg={caretDoubleLeftIcon}
			data-nudge="left"
			size={16}
			aria-hidden="true" />
	</button>
	<button
		class="btn"
		type="button"
		disabled={pager.loading || pager.pageNumber < 2}
		onclick={() => pager.previous()}>
		<Icon
			svg={caretLeftIcon}
			data-nudge="left"
			size={16}
			aria-hidden="true" />Previous
	</button>
	<span class="px-1 text-xs text-fg-muted" aria-current="page"
		>Page {pager.pageNumber}</span>
	<button
		class="btn"
		type="button"
		disabled={pager.loading || !pager.hasNext}
		onclick={() => pager.next()}>
		Next<Icon
			svg={caretRightIcon}
			data-nudge="right"
			size={16}
			aria-hidden="true" />
	</button>
</nav>

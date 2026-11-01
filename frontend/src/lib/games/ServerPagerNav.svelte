<script lang="ts">
	import CaretLeftIcon from "phosphor-svelte/lib/CaretLeftIcon"
	import CaretRightIcon from "phosphor-svelte/lib/CaretRightIcon"
	import CaretDoubleLeftIcon from "phosphor-svelte/lib/CaretDoubleLeftIcon"

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
		<CaretDoubleLeftIcon data-nudge="left" size={16} aria-hidden="true" />
	</button>
	<button
		class="btn"
		type="button"
		disabled={pager.loading || pager.pageNumber < 2}
		onclick={() => pager.previous()}>
		<CaretLeftIcon data-nudge="left" size={16} aria-hidden="true" />Previous
	</button>
	<span class="px-1 text-xs text-fg-muted" aria-current="page"
		>Page {pager.pageNumber}</span>
	<button
		class="btn"
		type="button"
		disabled={pager.loading || !pager.hasNext}
		onclick={() => pager.next()}>
		Next<CaretRightIcon data-nudge="right" size={16} aria-hidden="true" />
	</button>
</nav>

<script lang="ts">
	import ChevronLeft from "@lucide/svelte/icons/chevron-left"
	import ChevronRight from "@lucide/svelte/icons/chevron-right"
	import ChevronsLeft from "@lucide/svelte/icons/chevrons-left"

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

<nav class="game-servers-pager" aria-label="Server pages">
	<button
		class="control-button"
		type="button"
		aria-label="First page"
		data-tooltip="First page"
		disabled={pager.loading || pager.pageNumber < 2}
		onclick={() => pager.first()}>
		<ChevronsLeft size={15} aria-hidden="true" />
	</button>
	<button
		class="control-button games-text-button"
		type="button"
		disabled={pager.loading || pager.pageNumber < 2}
		onclick={() => pager.previous()}>
		<ChevronLeft size={14} aria-hidden="true" />Previous
	</button>
	<span class="game-servers-page-number" aria-current="page"
		>Page {pager.pageNumber}</span>
	<button
		class="control-button games-text-button"
		type="button"
		disabled={pager.loading || !pager.hasNext}
		onclick={() => pager.next()}>
		Next<ChevronRight size={14} aria-hidden="true" />
	</button>
</nav>

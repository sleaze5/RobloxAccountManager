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

<nav class="game-servers-pager" aria-label="Server pages">
	<button
		class="control-button"
		type="button"
		aria-label="First page"
		data-tooltip="First page"
		disabled={pager.loading || pager.pageNumber < 2}
		onclick={() => pager.first()}>
		<CaretDoubleLeftIcon size={17} aria-hidden="true" />
	</button>
	<button
		class="control-button games-text-button"
		type="button"
		disabled={pager.loading || pager.pageNumber < 2}
		onclick={() => pager.previous()}>
		<CaretLeftIcon size={16} aria-hidden="true" />Previous
	</button>
	<span class="game-servers-page-number" aria-current="page"
		>Page {pager.pageNumber}</span>
	<button
		class="control-button games-text-button"
		type="button"
		disabled={pager.loading || !pager.hasNext}
		onclick={() => pager.next()}>
		Next<CaretRightIcon size={16} aria-hidden="true" />
	</button>
</nav>

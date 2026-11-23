<script lang="ts">
	import circleNotchIcon from "@phosphor-icons/core/regular/circle-notch.svg?raw"
	import Icon from "./Icon.svelte"
	import { loadMoreWhenVisible } from "./load-more-when-visible"

	// LoadMore ends a list that loads in pages. It loads the next page
	// when it comes into view, and its button loads the page when that does
	// not happen, such as after a failed page or in a short window. A failed
	// page waits for the button, so that it does not retry while it stays in
	// view.
	let {
		pageKey,
		label,
		loadingLabel,
		isLoadingMore,
		hasFailed,
		isDisabled = false,
		onLoadMore,
	}: {
		// pageKey changes after each loaded page, so that the end of a list that
		// stays in view loads the next page too.
		pageKey: unknown
		label: string
		loadingLabel: string
		isLoadingMore: boolean
		hasFailed: boolean
		// isDisabled holds the next page while the whole list loads.
		isDisabled?: boolean
		onLoadMore: () => void
	} = $props()
</script>

<div
	class="flex min-h-12 flex-col items-center justify-center gap-2 p-2"
	use:loadMoreWhenVisible={{
		pageKey,
		loadMore: () => {
			if (!hasFailed && !isLoadingMore && !isDisabled) onLoadMore()
		},
	}}>
	{#if isLoadingMore}
		<span class="flex items-center gap-2 text-xs text-fg-muted" role="status"
			><Icon
				svg={circleNotchIcon}
				class="spinner"
				size={16}
				aria-hidden="true" />{loadingLabel}</span>
	{:else}
		{#if hasFailed}
			<span class="text-xs text-fg-muted">Next page unavailable</span>
		{/if}
		<button class="button" type="button" disabled={isDisabled} onclick={onLoadMore}
			>{label}</button>
	{/if}
</div>

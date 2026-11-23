<script lang="ts">
	import caretLeftIcon from "@phosphor-icons/core/regular/caret-left.svg?raw"
	import caretRightIcon from "@phosphor-icons/core/regular/caret-right.svg?raw"
	import type { Snippet } from "svelte"
	import Icon from "./Icon.svelte"
	import { reducedMotion } from "./motion"

	// TileCarousel shows item tiles in one row that scrolls sideways with its
	// Previous and Next buttons. The row hides its scrollbar, and a tile that
	// gets focus scrolls into view, so the keyboard reaches every tile.
	let {
		id,
		heading,
		label,
		actions,
		children,
	}: {
		// id names the heading, which labels the region.
		id: string
		heading: Snippet
		label: string
		actions?: Snippet
		// children renders the tiles, each in an li.
		children: Snippet
	} = $props()

	let row = $state<HTMLUListElement | undefined>(undefined),
		canScrollBack = $state(false),
		canScrollForward = $state(false)

	function updateScrollState(): void {
		if (!row) return
		canScrollBack = row.scrollLeft > 0
		// One pixel of tolerance, because scrollLeft can be fractional.
		canScrollForward = row.scrollLeft + row.clientWidth < row.scrollWidth - 1
	}

	// scrollByPage moves the row by the tiles that fit in it, so that a page
	// starts at the edge of a tile instead of cutting one off.
	function scrollByPage(direction: -1 | 1): void {
		if (!row) return
		const tiles = row.children,
			tileStep =
				tiles.length > 1
					? (tiles[1] as HTMLElement).offsetLeft -
						(tiles[0] as HTMLElement).offsetLeft
					: row.clientWidth,
			gap =
				tiles.length > 1 ? tileStep - (tiles[0] as HTMLElement).offsetWidth : 0,
			tilesPerPage = Math.max(1, Math.floor((row.clientWidth + gap) / tileStep)),
			currentTile = Math.round(row.scrollLeft / tileStep)
		row.scrollTo({
			left: Math.max(0, (currentTile + direction * tilesPerPage) * tileStep),
			behavior: reducedMotion() ? "instant" : "smooth",
		})
	}

	$effect(() => {
		const observedRow = row
		if (!observedRow) return
		// The row changes size with the window, and its content with each refresh.
		const resizeObserver = new ResizeObserver(updateScrollState),
			mutationObserver = new MutationObserver(updateScrollState)
		resizeObserver.observe(observedRow)
		mutationObserver.observe(observedRow, { childList: true })
		updateScrollState()
		return () => {
			resizeObserver.disconnect()
			mutationObserver.disconnect()
		}
	})
</script>

<section class="flex min-w-0 flex-col gap-2" aria-labelledby={id}>
	<div class="flex h-6 items-center gap-1.5">
		<h3
			{id}
			class="flex min-w-0 items-center gap-1.5 text-xs font-semibold text-fg-muted">
			{@render heading()}
		</h3>
		<div class="ml-auto flex shrink-0 items-center gap-1">
			{@render actions?.()}
			<button
				class="button-ghost-sm"
				type="button"
				aria-label="Previous"
				data-tooltip="Previous"
				disabled={!canScrollBack}
				onclick={() => scrollByPage(-1)}>
				<Icon
					svg={caretLeftIcon}
					data-nudge="left"
					size={14}
					aria-hidden="true" />
			</button>
			<button
				class="button-ghost-sm"
				type="button"
				aria-label="Next"
				data-tooltip="Next"
				disabled={!canScrollForward}
				onclick={() => scrollByPage(1)}>
				<Icon
					svg={caretRightIcon}
					data-nudge="right"
					size={14}
					aria-hidden="true" />
			</button>
		</div>
	</div>
	<ul
		class="flex gap-2 overflow-x-hidden"
		aria-label={label}
		bind:this={row}
		onscroll={updateScrollState}>
		{@render children()}
	</ul>
</section>

<script lang="ts">
	import caretRightIcon from "@phosphor-icons/core/regular/caret-right.svg?raw"
	import magnifyingGlassIcon from "@phosphor-icons/core/regular/magnifying-glass.svg?raw"
	import packageIcon from "@phosphor-icons/core/regular/package.svg?raw"
	import warningCircleIcon from "@phosphor-icons/core/regular/warning-circle.svg?raw"
	import xIcon from "@phosphor-icons/core/regular/x.svg?raw"
	import { tick, untrack } from "svelte"
	import Icon from "../shared/Icon.svelte"
	import { searchTerms } from "../shared/search"
	import { nodeName, visibleRows } from "./explorer-tree"
	import type { ExplorerNode, ExplorerRow, ExplorerTree } from "./explorer-tree"
	import { classIcon } from "./instance-class-icons"

	// InstanceTree lists the instances of an explorer as a tree that keeps
	// keyboard focus itself and renders only the rows in view, so that a model
	// with thousands of instances stays responsive.
	let {
		tree,
		expandedKeys,
		selectedKey,
		searchQuery = $bindable(),
		onSelect,
		onToggle,
	}: {
		tree: ExplorerTree
		expandedKeys: ReadonlySet<string>
		selectedKey: string
		searchQuery: string
		onSelect: (node: ExplorerNode) => void
		onToggle: (node: ExplorerNode, isExpanded: boolean) => void
	} = $props()

	const elementIdPrefix = $props.id()
	// overscanRowCount renders rows beyond the edges of the view, so that a
	// fast scroll does not show empty space before the next frame.
	const overscanRowCount = 8

	let treeElement = $state<HTMLDivElement>(),
		searchInput = $state<HTMLInputElement>(),
		scrollTop = $state(0),
		viewportHeight = $state(0),
		// Every row is h-7. The first rendered row measures it, so that the
		// tree follows the spacing scale without repeating it.
		rowHeight = $state(28)

	const terms = $derived(searchTerms(searchQuery)),
		isSearching = $derived(terms.length > 0),
		rows = $derived(visibleRows(tree, expandedKeys, terms)),
		matchCount = $derived(rows.filter((row) => row.isMatch).length),
		selectedIndex = $derived(rows.findIndex((row) => row.node.key === selectedKey)),
		firstShownIndex = $derived(
			Math.max(0, Math.floor(scrollTop / rowHeight) - overscanRowCount),
		),
		shownRows = $derived(
			rows.slice(
				firstShownIndex,
				Math.min(
					rows.length,
					Math.ceil((scrollTop + viewportHeight) / rowHeight) +
						overscanRowCount,
				),
			),
		)

	function rowId(node: ExplorerNode): string {
		return `${elementIdPrefix}-${node.key.replace(":", "-")}`
	}

	$effect(() => {
		const firstRow = treeElement?.querySelector<HTMLElement>("[data-tree-row]")
		if (firstRow && firstRow.offsetHeight > 0) rowHeight = firstRow.offsetHeight
	})

	// revealRow scrolls the least distance that brings the row into view.
	function revealRow(index: number): void {
		if (!treeElement || index < 0) return
		const rowTop = index * rowHeight
		if (rowTop < treeElement.scrollTop) treeElement.scrollTop = rowTop
		else if (rowTop + rowHeight > treeElement.scrollTop + viewportHeight) {
			treeElement.scrollTop = rowTop + rowHeight - viewportHeight
		}
	}

	// A selection that changes from outside, such as a reference that the
	// properties open, scrolls its row into view.
	$effect(() => {
		const index = selectedIndex
		untrack(() => revealRow(index))
	})

	// A new search shows its results from the top.
	$effect(() => {
		void terms
		untrack(() => {
			if (treeElement) treeElement.scrollTop = 0
		})
	})

	function selectRow(row: ExplorerRow | undefined): void {
		if (row) onSelect(row.node)
	}

	function toggleRow(row: ExplorerRow): void {
		if (row.node.children.length > 0 && !isSearching) {
			onToggle(row.node, !row.isExpanded)
		}
	}

	// rowIndexAfterKey returns the index of the row that a navigation key
	// moves the selection to, or undefined for any other key.
	function rowIndexAfterKey(key: string): number | undefined {
		const pageRowCount = Math.max(1, Math.floor(viewportHeight / rowHeight) - 1),
			hasSelection = selectedIndex >= 0
		switch (key) {
			case "ArrowDown":
				return hasSelection ? Math.min(rows.length - 1, selectedIndex + 1) : 0
			case "ArrowUp":
				return hasSelection ? Math.max(0, selectedIndex - 1) : 0
			case "PageDown":
				return Math.min(rows.length - 1, selectedIndex + pageRowCount)
			case "PageUp":
				return Math.max(0, selectedIndex - pageRowCount)
			case "Home":
				return 0
			case "End":
				return rows.length - 1
			default:
				return undefined
		}
	}

	// handleExpansionKey opens and closes the selected row, or moves into or
	// out of it, and reports whether the key did that.
	function handleExpansionKey(key: string, row: ExplorerRow): boolean {
		const hasChildren = row.node.children.length > 0
		if (key === "ArrowRight") {
			if (row.isExpanded) selectRow(rows[selectedIndex + 1])
			else if (hasChildren) toggleRow(row)
		} else if (key === "ArrowLeft") {
			if (row.isExpanded && !isSearching) toggleRow(row)
			else selectRow(rows.find((candidate) => candidate.node === row.node.parent))
		} else if (key === "Enter" || key === " ") {
			toggleRow(row)
		} else {
			return false
		}
		return true
	}

	function handleKeydown(event: KeyboardEvent): void {
		if (event.ctrlKey || event.metaKey || event.altKey) return
		const nextIndex = rowIndexAfterKey(event.key),
			row = rows[selectedIndex]
		if (nextIndex !== undefined) selectRow(rows[nextIndex])
		else if (!row || !handleExpansionKey(event.key, row)) return
		event.preventDefault()
	}

	// rowOf finds the row that a pointer event happened on.
	function rowOf(event: MouseEvent): ExplorerRow | undefined {
		const rowElement =
			event.target instanceof Element
				? event.target.closest<HTMLElement>("[data-tree-row]")
				: null
		return rowElement
			? rows.find((row) => row.node.key === rowElement.dataset.treeRow)
			: undefined
	}

	function isOnCaret(event: MouseEvent): boolean {
		return (
			event.target instanceof Element &&
			!!event.target.closest("[data-tree-caret]")
		)
	}

	async function clearSearch(): Promise<void> {
		searchQuery = ""
		await tick()
		searchInput?.focus()
	}
</script>

<div class="flex shrink-0 items-center gap-2 border-b border-line-subtle p-2">
	<div class="search-field" data-closes-on-escape={searchQuery ? "" : undefined}>
		<Icon svg={magnifyingGlassIcon} size={16} aria-hidden="true" />
		<input
			bind:this={searchInput}
			type="search"
			placeholder="Search instances"
			aria-label="Search instances by name or class"
			aria-controls={`${elementIdPrefix}-tree`}
			autocomplete="off"
			spellcheck="false"
			bind:value={searchQuery}
			onkeydown={(event) => {
				if (event.key === "Escape" && searchQuery) {
					event.preventDefault()
					searchQuery = ""
				} else if (event.key === "ArrowDown") {
					event.preventDefault()
					treeElement?.focus()
					if (selectedIndex < 0) selectRow(rows[0])
				}
			}} />
		{#if searchQuery}
			<button
				type="button"
				aria-label="Clear search"
				onclick={() => void clearSearch()}>
				<Icon svg={xIcon} size={14} aria-hidden="true" />
			</button>
		{/if}
	</div>
</div>

{#if rows.length === 0}
	<div class="empty-state min-h-0 flex-1 animate-enter">
		<Icon svg={magnifyingGlassIcon} size={28} aria-hidden="true" />
		<h2>No matching instances</h2>
		<p>Search by the name or the class of an instance.</p>
		<button class="button" type="button" onclick={() => void clearSearch()}
			>Clear search</button>
	</div>
{:else}
	<div
		bind:this={treeElement}
		bind:clientHeight={viewportHeight}
		id={`${elementIdPrefix}-tree`}
		class="relative min-h-0 flex-1 scrollbar-stable overflow-y-auto overscroll-contain py-1 focus-visible:outline-0"
		role="tree"
		tabindex="0"
		aria-label="Instances"
		aria-activedescendant={selectedIndex >= 0
			? rowId(rows[selectedIndex].node)
			: undefined}
		onscroll={(event) => (scrollTop = event.currentTarget.scrollTop)}
		onkeydown={handleKeydown}
		onmousedown={(event) => {
			// The tree keeps focus, so that the arrow keys keep working after a
			// click on a row.
			if (rowOf(event)) {
				event.preventDefault()
				treeElement?.focus()
			}
		}}
		onclick={(event) => {
			const row = rowOf(event)
			if (!row) return
			if (isOnCaret(event)) toggleRow(row)
			else onSelect(row.node)
		}}
		ondblclick={(event) => {
			const row = rowOf(event)
			if (row && !isOnCaret(event)) toggleRow(row)
		}}>
		<div
			class="relative h-(--tree-height)"
			style:--tree-height={`${rows.length * rowHeight}px`}>
			<div
				class="absolute inset-x-0 top-0 translate-y-(--tree-offset)"
				style:--tree-offset={`${firstShownIndex * rowHeight}px`}>
				{#each shownRows as row (row.node.key)}
					{@const instance = row.node.instance}
					{@const name = nodeName(row.node)}
					{@const isUnavailable =
						!instance && row.node.model.unavailableMessage !== ""}
					<div
						id={rowId(row.node)}
						class="tree-row"
						style:--tree-depth={row.depth}
						role="treeitem"
						aria-level={row.depth + 1}
						aria-setsize={row.setSize}
						aria-posinset={row.positionInSet}
						aria-expanded={row.node.children.length > 0
							? row.isExpanded
							: undefined}
						aria-selected={row.node.key === selectedKey}
						data-tree-row={row.node.key}>
						<span
							class="grid size-4 shrink-0 place-items-center text-fg-faint transition-colors duration-100 hover:text-fg"
							data-tree-caret>
							{#if row.node.children.length > 0}
								<Icon
									svg={caretRightIcon}
									size={12}
									class={[
										"transition-transform duration-100 ease-out",
										row.isExpanded && "rotate-90",
									]}
									aria-hidden="true" />
							{/if}
						</span>
						<Icon
							svg={instance
								? classIcon(instance.className)
								: isUnavailable
									? warningCircleIcon
									: packageIcon}
							size={14}
							class={[
								"shrink-0",
								isUnavailable ? "text-warning" : "text-fg-muted",
							]}
							aria-hidden="true" />
						<span
							class={[
								"min-w-0 truncate",
								row.isMatch && "font-semibold text-fg-bright",
								!instance && "font-medium",
							]}>{name || "Unnamed asset"}</span>
						{#if instance && instance.name !== instance.className}
							<span
								class="min-w-0 truncate font-mono text-2xs text-fg-faint"
								>{instance.className}</span>
						{/if}
					</div>
				{/each}
			</div>
		</div>
	</div>
{/if}
{#if isSearching}
	<p class="sr-only" role="status">
		{matchCount === 1 ? "1 matching instance" : `${matchCount} matching instances`}
	</p>
{/if}

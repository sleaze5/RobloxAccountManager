<script lang="ts">
	import warningCircleIcon from "@phosphor-icons/core/regular/warning-circle.svg?raw"
	import xIcon from "@phosphor-icons/core/regular/x.svg?raw"
	import { tick, untrack } from "svelte"
	import { ModelPropertyKind } from "../backend/bridge"
	import type { InventoryItem, ModelProperty } from "../backend/bridge"
	import type { InventoryCategoryOption } from "../inventory/inventory-categories"
	import { categoryItemKey } from "../inventory/inventory-state.svelte"
	import type { InventoryState } from "../inventory/inventory-state.svelte"
	import Dialog from "../shared/Dialog.svelte"
	import Icon from "../shared/Icon.svelte"
	import DocumentViewer from "./DocumentViewer.svelte"
	import {
		ancestorKeys,
		buildExplorerTree,
		firstInstanceNode,
		initiallyExpandedKeys,
		instanceNodeKey,
	} from "./explorer-tree"
	import type { ExplorerNode } from "./explorer-tree"
	import InstanceDetails from "./InstanceDetails.svelte"
	import InstanceTree from "./InstanceTree.svelte"

	// ItemExplorerDialog shows the instances in the file of an item, or of each
	// asset of a bundle, as Roblox Studio lists them, with their properties.
	let {
		item,
		category,
		inventory,
		onClose,
	}: {
		item: InventoryItem
		category: InventoryCategoryOption
		inventory: InventoryState
		onClose: () => void
	} = $props()

	const titleId = $props.id()
	const exactNumbers = new Intl.NumberFormat("en-US")
	// The placeholder rows while the instances load: their indents and widths.
	const skeletonRows = [
		["pl-1", "w-36"],
		["pl-5", "w-28"],
		["pl-9", "w-32"],
		["pl-9", "w-24"],
		["pl-9", "w-28"],
		["pl-5", "w-32"],
		["pl-9", "w-20"],
		["pl-5", "w-28"],
	] as const

	let card = $state<HTMLDivElement>(),
		expandedKeys = $state.raw(new Set<string>()),
		selectedKey = $state(""),
		searchQuery = $state(""),
		propertyQuery = $state(""),
		collapsedGroups = $state.raw(new Set<string>()),
		// The name of the long string property that the viewer shows, on the
		// selected instance.
		openedDocumentName = $state("")

	const itemKey = $derived(categoryItemKey(category.id, item.itemId)),
		loadedTree = $derived(inventory.instanceTree.loadedFor(itemKey)),
		loadError = $derived(inventory.instanceTree.errorFor(itemKey)),
		tree = $derived(loadedTree ? buildExplorerTree(loadedTree) : null),
		imageUrlByAssetId = $derived(
			new Map(
				Object.entries(loadedTree?.imageUrlByAssetId ?? {}).flatMap(
					([assetId, imageUrl]) =>
						imageUrl ? [[Number(assetId), imageUrl] as const] : [],
				),
			),
		),
		models = $derived(loadedTree?.models ?? []),
		fileInstanceCount = $derived(
			models.reduce((count, model) => count + model.model.instanceCount, 0),
		),
		isTruncated = $derived(models.some((model) => model.model.isTruncated)),
		selectedNode = $derived(tree?.nodeByKey.get(selectedKey)),
		openedDocument = $derived(
			selectedNode?.instance?.properties?.find(
				(property) =>
					property.name === openedDocumentName &&
					property.kind === ModelPropertyKind.PropertyDocument,
			),
		)

	$effect(() => {
		const shownItem = item,
			shownCategory = category.id
		untrack(() => void inventory.loadInstanceTree(shownItem, shownCategory))
	})

	// A loaded tree opens its first levels with its first instance selected,
	// so that the properties show without a click.
	$effect(() => {
		if (!tree) return
		const shownTree = tree
		untrack(() => {
			expandedKeys = initiallyExpandedKeys(shownTree)
			selectedKey = firstInstanceNode(shownTree)?.key ?? ""
			openedDocumentName = ""
		})
	})

	function select(node: ExplorerNode): void {
		selectedKey = node.key
		openedDocumentName = ""
	}

	function toggle(node: ExplorerNode, isExpanded: boolean): void {
		const nextExpandedKeys = new Set(expandedKeys)
		if (isExpanded) nextExpandedKeys.add(node.key)
		else nextExpandedKeys.delete(node.key)
		expandedKeys = nextExpandedKeys
	}

	// selectInstance shows the instance that a reference points to, and opens
	// the instances that contain it. It clears a search that could hide it.
	async function selectInstance(instanceId: number): Promise<void> {
		const node = tree?.nodeByKey.get(instanceNodeKey(instanceId))
		if (!node) return
		searchQuery = ""
		expandedKeys = new Set([...expandedKeys, ...ancestorKeys(node)])
		select(node)
		await tick()
		card?.querySelector<HTMLElement>("[role=tree]")?.focus()
	}

	function toggleGroup(group: string, isCollapsed: boolean): void {
		const nextCollapsedGroups = new Set(collapsedGroups)
		if (isCollapsed) nextCollapsedGroups.add(group)
		else nextCollapsedGroups.delete(group)
		collapsedGroups = nextCollapsedGroups
	}

	async function openDocument(property: ModelProperty): Promise<void> {
		inventory.explorerCopyError = ""
		openedDocumentName = property.name
		await tick()
		card?.querySelector<HTMLElement>("[data-document-back]")?.focus()
	}

	// closeDocument returns to the properties with focus on the button that
	// opened the viewer.
	async function closeDocument(): Promise<void> {
		const closedName = openedDocumentName
		openedDocumentName = ""
		inventory.explorerCopyError = ""
		await tick()
		card?.querySelector<HTMLElement>(
			`[data-view-document="${CSS.escape(closedName)}"]`,
		)?.focus()
	}

	function close(): void {
		inventory.explorerCopyError = ""
		onClose()
	}
</script>

<!-- The explorer opens above the item details, so it handles Escape before
the dialog below it. A search field with text clears itself first, and the
source viewer closes before the explorer. -->
<svelte:window
	onkeydowncapture={(event) => {
		if (event.key !== "Escape") return
		if (
			event.target instanceof Element &&
			event.target.closest("[data-closes-on-escape]")
		) {
			return
		}
		event.preventDefault()
		event.stopImmediatePropagation()
		if (openedDocument) void closeDocument()
		else close()
	}} />

<Dialog
	class="h-full max-w-6xl"
	aria-labelledby={titleId}
	hasFlexibleBody
	bind:card
	onBackdropClick={close}>
	<div class="dialog-heading">
		<div>
			<h2 id={titleId} class="truncate">
				Explorer <span class="font-normal text-fg-muted"
					>· {item.name || "Unnamed item"}</span>
			</h2>
		</div>
		<button type="button" aria-label="Close dialog" onclick={close}>
			<Icon svg={xIcon} size={14} aria-hidden="true" />
		</button>
	</div>

	{#if inventory.explorerCopyError}
		<p class="mb-3 notice-error" role="alert">{inventory.explorerCopyError}</p>
	{/if}

	<div
		class="flex min-h-0 flex-1 flex-col overflow-hidden rounded-lg border border-line bg-canvas">
		{#if tree}
			<div class="flex min-h-0 flex-1">
				<section
					class="flex w-64 shrink-0 flex-col border-r border-line-subtle bg-surface lg:w-72"
					aria-label="Instances">
					<InstanceTree
						{tree}
						{expandedKeys}
						{selectedKey}
						bind:searchQuery
						onSelect={select}
						onToggle={toggle} />
				</section>
				<section class="flex min-w-0 flex-1 flex-col" aria-label="Properties">
					{#if selectedNode && openedDocument && selectedNode.instance}
						<DocumentViewer
							instanceName={selectedNode.instance.name}
							property={openedDocument}
							onBack={() => void closeDocument()}
							onCopy={(text) =>
								inventory.copyExplorerText(
									text,
									openedDocument.name,
								)} />
					{:else if selectedNode}
						<InstanceDetails
							node={selectedNode}
							{imageUrlByAssetId}
							bind:propertyQuery
							{collapsedGroups}
							onToggleGroup={toggleGroup}
							referencedClassNameOf={(instanceId) =>
								tree?.nodeByKey.get(instanceNodeKey(instanceId))
									?.instance?.className ?? ""}
							onSelectInstance={(instanceId) =>
								void selectInstance(instanceId)}
							onOpenDocument={(property) =>
								void openDocument(property)} />
					{:else}
						<div class="empty-state min-h-0 flex-1">
							<h2>No instance selected</h2>
							<p>Select an instance to show its properties.</p>
						</div>
					{/if}
				</section>
			</div>
			<footer
				class="flex h-6 shrink-0 items-center gap-2 border-t border-line-subtle bg-surface px-3 text-2xs text-fg-muted tabular-nums">
				{#if isTruncated}
					<Icon
						svg={warningCircleIcon}
						size={12}
						class="shrink-0 text-warning"
						aria-hidden="true" />
					<span class="truncate"
						>Showing the first {exactNumbers.format(tree.instanceCount)} of {exactNumbers.format(
							fileInstanceCount,
						)} instances</span>
				{:else}
					<span
						>{tree.instanceCount === 1
							? "1 instance"
							: `${exactNumbers.format(tree.instanceCount)} instances`}</span>
				{/if}
				<span class="text-fg-faint" aria-hidden="true">·</span>
				{#if models.length === 1}
					<span class="font-mono">{models[0].model.format}</span>
				{:else}
					<span>{models.length} assets</span>
				{/if}
			</footer>
		{:else if loadError}
			<div class="empty-state min-h-0 flex-1 animate-enter">
				<Icon svg={warningCircleIcon} size={28} aria-hidden="true" />
				<h2>Instances unavailable</h2>
				<p role="alert">{loadError}</p>
				<button
					class="button"
					type="button"
					onclick={() => void inventory.loadInstanceTree(item, category.id)}
					>Try again</button>
			</div>
		{:else}
			<div class="flex min-h-0 flex-1" aria-busy="true">
				<div
					class="flex w-64 shrink-0 flex-col gap-1 border-r border-line-subtle bg-surface lg:w-72">
					<div class="border-b border-line-subtle p-2">
						<div class="h-8 skeleton rounded-md"></div>
					</div>
					<div class="flex flex-col py-1">
						{#each skeletonRows as [indentClass, widthClass], index (index)}
							<div
								class={[
									"flex h-7 items-center gap-2 pr-2",
									indentClass,
								]}>
								<span class="size-4 shrink-0"></span>
								<span class="size-3.5 shrink-0 skeleton rounded-sm"
								></span>
								<span class={["skeleton-line text-xs", widthClass]}
								></span>
							</div>
						{/each}
					</div>
				</div>
				<div class="flex min-w-0 flex-1 flex-col">
					<div
						class="flex items-center gap-3 border-b border-line-subtle p-3">
						<span class="size-9 shrink-0 skeleton rounded-md"></span>
						<div class="flex min-w-0 flex-1 flex-col">
							<span class="skeleton-line w-32 text-sm"></span>
							<span class="skeleton-line w-48 text-2xs"></span>
						</div>
					</div>
					<div class="flex flex-col gap-3 p-3">
						{#each [["w-24", "w-36"], ["w-28", "w-20"], ["w-20", "w-44"], ["w-32", "w-28"]] as [labelWidth, valueWidth], index (index)}
							<div class="flex items-center gap-4">
								<span class={["skeleton-line text-xs", labelWidth]}
								></span>
								<span class={["skeleton-line text-xs", valueWidth]}
								></span>
							</div>
						{/each}
					</div>
				</div>
				<span class="sr-only" role="status">Loading instances…</span>
			</div>
		{/if}
	</div>
</Dialog>

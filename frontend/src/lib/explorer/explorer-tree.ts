import type { ExploredModel, ItemInstanceTree, ModelInstance } from "../backend/bridge"
import { matchesSearch } from "../shared/search"

// ExplorerNode is one row that the explorer can show: an instance, or an asset
// of a bundle, which holds the instances of its file.
export interface ExplorerNode {
	key: string
	parent: ExplorerNode | null
	children: ExplorerNode[]
	model: ExploredModel
	// instance is null for the node of an asset of a bundle.
	instance: ModelInstance | null
}

export interface ExplorerTree {
	roots: ExplorerNode[]
	nodeByKey: Map<string, ExplorerNode>
	// instanceCount counts the instances that the tree holds, which can be
	// fewer than the files hold.
	instanceCount: number
}

// ExplorerRow is a node as the tree shows it, with its place among its
// siblings for screen readers.
export interface ExplorerRow {
	node: ExplorerNode
	depth: number
	isExpanded: boolean
	isMatch: boolean
	positionInSet: number
	setSize: number
}

// GuideRange is the rows whose line at one level leads to the selected row:
// from the first child of the ancestor at that level to the row on the way
// to the selected row.
export interface GuideRange {
	firstIndex: number
	lastIndex: number
}

export function instanceNodeKey(instanceId: number): string {
	return `instance:${instanceId}`
}

// buildExplorerTree links the instances of the tree to their parents. The
// assets of a bundle become the roots, and the file of one item shows its
// instances as the roots.
export function buildExplorerTree(tree: ItemInstanceTree): ExplorerTree {
	const nodeByKey = new Map<string, ExplorerNode>()
	let instanceCount = 0
	const models = tree.models ?? []
	const addInstances = (
		instances: ModelInstance[] | null,
		model: ExploredModel,
		parent: ExplorerNode | null,
	): ExplorerNode[] =>
		(instances ?? []).map((instance) => {
			const node: ExplorerNode = {
				key: instanceNodeKey(instance.id),
				parent,
				children: [],
				model,
				instance,
			}
			nodeByKey.set(node.key, node)
			instanceCount++
			node.children = addInstances(instance.children, model, node)
			return node
		})
	const isBundle = models.some((model) => model.name !== "")
	const roots = isBundle
		? models.map((model) => {
				const node: ExplorerNode = {
					key: `asset:${model.assetId}`,
					parent: null,
					children: [],
					model,
					instance: null,
				}
				nodeByKey.set(node.key, node)
				node.children = addInstances(model.model.instances, model, node)
				return node
			})
		: models.flatMap((model) => addInstances(model.model.instances, model, null))
	return { roots, nodeByKey, instanceCount }
}

// firstInstanceNode is the node that the explorer selects when it opens: the
// first instance, inside the first asset of a bundle.
export function firstInstanceNode(tree: ExplorerTree): ExplorerNode | undefined {
	const firstRoot = tree.roots[0]
	return firstRoot?.instance ? firstRoot : (firstRoot?.children[0] ?? firstRoot)
}

// initiallyExpandedKeys opens the first two levels, which show what an item is
// made of without a click.
export function initiallyExpandedKeys(tree: ExplorerTree): Set<string> {
	const keys = new Set<string>()
	for (const root of tree.roots) {
		keys.add(root.key)
		for (const child of root.children) {
			if (child.children.length > 0) keys.add(child.key)
		}
	}
	return keys
}

export function nodeName(node: ExplorerNode): string {
	return node.instance ? node.instance.name : node.model.name
}

function nodeMatches(node: ExplorerNode, terms: string[]): boolean {
	return matchesSearch(
		terms,
		node.instance
			? `${node.instance.name} ${node.instance.className}`
			: node.model.name,
	)
}

// visibleRows lists the rows that the tree shows in order. While a search
// runs, it shows the matching nodes and the nodes that contain them, opened.
export function visibleRows(
	tree: ExplorerTree,
	expandedKeys: ReadonlySet<string>,
	terms: string[],
): ExplorerRow[] {
	const rows: ExplorerRow[] = []
	const isSearching = terms.length > 0
	const containsMatch = new Set<ExplorerNode>()
	if (isSearching) {
		const markMatches = (node: ExplorerNode): boolean => {
			let hasMatch = nodeMatches(node, terms)
			for (const child of node.children) {
				if (markMatches(child)) hasMatch = true
			}
			if (hasMatch) containsMatch.add(node)
			return hasMatch
		}
		for (const root of tree.roots) markMatches(root)
	}
	const addRows = (nodes: ExplorerNode[], depth: number): void => {
		const shownNodes = isSearching
			? nodes.filter((node) => containsMatch.has(node))
			: nodes
		shownNodes.forEach((node, index) => {
			const isExpanded =
				node.children.length > 0 &&
				(isSearching
					? node.children.some((child) => containsMatch.has(child))
					: expandedKeys.has(node.key))
			rows.push({
				node,
				depth,
				isExpanded,
				isMatch: isSearching && nodeMatches(node, terms),
				positionInSet: index + 1,
				setSize: shownNodes.length,
			})
			if (isExpanded) addRows(node.children, depth + 1)
		})
	}
	addRows(tree.roots, 0)
	return rows
}

// activeGuideRanges finds, by level, the rows whose lines lead from the root
// to the selected row, so that the tree can trace its path.
export function activeGuideRanges(
	rows: ExplorerRow[],
	selectedIndex: number,
): Map<number, GuideRange> {
	const ranges = new Map<number, GuideRange>()
	const indexByNode = new Map(rows.map((row, index) => [row.node, index]))
	for (
		let pathNode = rows[selectedIndex]?.node;
		pathNode?.parent;
		pathNode = pathNode.parent
	) {
		const parentIndex = indexByNode.get(pathNode.parent),
			pathIndex = indexByNode.get(pathNode)
		if (parentIndex === undefined || pathIndex === undefined) break
		ranges.set(rows[pathIndex].depth - 1, {
			firstIndex: parentIndex + 1,
			lastIndex: pathIndex,
		})
	}
	return ranges
}

// ancestorKeys lists the keys of the nodes that contain the node, so that
// opening them shows it.
export function ancestorKeys(node: ExplorerNode): string[] {
	const keys: string[] = []
	for (let ancestor = node.parent; ancestor; ancestor = ancestor.parent) {
		keys.push(ancestor.key)
	}
	return keys
}

// instancePath names the instances from the root down to the instance, as
// Roblox Studio writes it, such as Accessory.Handle.
export function instancePath(node: ExplorerNode): string {
	const names: string[] = []
	for (
		let ancestor: ExplorerNode | null = node;
		ancestor?.instance;
		ancestor = ancestor.parent
	) {
		names.unshift(ancestor.instance.name)
	}
	return names.join(".")
}

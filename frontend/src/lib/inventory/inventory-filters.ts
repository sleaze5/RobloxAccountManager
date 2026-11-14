import type { InventoryItem } from "../backend/bridge"
import { matchesSearch } from "../shared/search"

export type InventorySaleFilter = "all" | "forSale" | "offSale" | "limited"

export type InventorySortOrder = "newest" | "name" | "highestValue" | "lowestValue"

export const saleFilterOptions: { value: InventorySaleFilter; label: string }[] = [
	{ value: "all", label: "All items" },
	{ value: "forSale", label: "For sale" },
	{ value: "offSale", label: "Off sale" },
	{ value: "limited", label: "Limited" },
]

export const sortOrderOptions: { value: InventorySortOrder; label: string }[] = [
	{ value: "newest", label: "Newest" },
	{ value: "name", label: "Name" },
	{ value: "highestValue", label: "Highest value" },
	{ value: "lowestValue", label: "Lowest value" },
]

// itemValueInRobux is what the item is worth: the recent average price of a
// collectible, the cheapest resale of a limited item, or its price.
export function itemValueInRobux(item: InventoryItem): number | null {
	if (item.recentAveragePrice !== null) return item.recentAveragePrice
	if (item.market?.lowestResalePriceInRobux != null) {
		return item.market.lowestResalePriceInRobux
	}
	return item.market?.isForSale ? item.market.priceInRobux : null
}

function matchesSaleFilter(
	item: InventoryItem,
	saleFilter: InventorySaleFilter,
): boolean {
	switch (saleFilter) {
		case "forSale":
			return !!item.market?.isForSale
		case "offSale":
			return !!item.market && !item.market.isForSale
		case "limited":
			return !!item.market?.isLimited
		default:
			return true
	}
}

const nameOrder = new Intl.Collator("en-US", { sensitivity: "base", numeric: true })

// shapeInventoryItems filters the loaded items and sorts them. Newest keeps
// the order of Roblox. Items without a value go last in a value order.
export function shapeInventoryItems(
	items: InventoryItem[],
	{
		searchTerms,
		typeName,
		saleFilter,
		sortOrder,
	}: {
		searchTerms: string[]
		typeName: string
		saleFilter: InventorySaleFilter
		sortOrder: InventorySortOrder
	},
): InventoryItem[] {
	const shownItems = items.filter(
		(item) =>
			(!typeName || item.typeName === typeName) &&
			matchesSaleFilter(item, saleFilter) &&
			(searchTerms.length === 0 ||
				(searchTerms.length === 1 && searchTerms[0] === String(item.itemId)) ||
				matchesSearch(searchTerms, `${item.name} ${item.creatorName}`)),
	)
	switch (sortOrder) {
		case "name":
			return shownItems.sort((first, second) =>
				nameOrder.compare(first.name, second.name),
			)
		case "highestValue":
			return shownItems.sort((first, second) =>
				compareByValue(first, second, true),
			)
		case "lowestValue":
			return shownItems.sort((first, second) =>
				compareByValue(first, second, false),
			)
		default:
			return shownItems
	}
}

// compareByValue puts items without a value last in either order.
function compareByValue(
	first: InventoryItem,
	second: InventoryItem,
	isHighestFirst: boolean,
): number {
	const firstValue = itemValueInRobux(first),
		secondValue = itemValueInRobux(second)
	if (firstValue === null || secondValue === null) {
		return (firstValue === null ? 1 : 0) - (secondValue === null ? 1 : 0)
	}
	return isHighestFirst ? secondValue - firstValue : firstValue - secondValue
}

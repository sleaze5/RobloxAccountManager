import type { InventoryItemMarket } from "../backend/bridge"

const exactNumbers = new Intl.NumberFormat("en-US"),
	compactNumbers = new Intl.NumberFormat("en-US", {
		notation: "compact",
		maximumFractionDigits: 1,
	})

export function formatRobux(amount: number, isCompact: boolean): string {
	return `R$ ${(isCompact ? compactNumbers : exactNumbers).format(amount)}`
}

// marketPriceLabel names what the item costs now: the cheapest resale of a
// limited item, or else its sale price.
export function marketPriceLabel(
	market: InventoryItemMarket,
	isCompact: boolean,
): string {
	if (market.lowestResalePriceInRobux !== null) {
		return formatRobux(market.lowestResalePriceInRobux, isCompact)
	}
	return salePriceLabel(market, isCompact)
}

// salePriceLabel names what the seller asks for a new copy: its price, Free,
// or Off sale.
export function salePriceLabel(
	market: InventoryItemMarket,
	isCompact: boolean,
): string {
	if (!market.isForSale) return "Off sale"
	return market.priceInRobux === 0
		? "Free"
		: formatRobux(market.priceInRobux, isCompact)
}

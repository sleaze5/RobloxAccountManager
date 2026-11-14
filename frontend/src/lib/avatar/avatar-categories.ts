import sparkleIcon from "@phosphor-icons/core/regular/sparkle.svg?raw"
import { InventoryCategory } from "../backend/bridge"
import type { AvatarAsset } from "../backend/bridge"
import { inventoryCategories } from "../inventory/inventory-categories"
import type { InventoryCategoryOption } from "../inventory/inventory-categories"

// Every worn item is an asset, and the asset categories read the same item
// details, so an item of a type that no category lists reads its details as
// an accessory.
const otherItemsCategory: InventoryCategoryOption = {
	id: InventoryCategory.InventoryAccessories,
	label: "Other",
	noun: "items",
	icon: sparkleIcon,
	itemDetail: "type",
	isDeletable: false,
}

// The categories that hold items an avatar wears, in the order of the
// inventory. Bundles and emotes are left out: Roblox wears the assets of a
// bundle, and emotes go in the emote slots.
export const wearableCategories = inventoryCategories.filter((option) =>
	[
		InventoryCategory.InventoryAccessories,
		InventoryCategory.InventoryClassicClothing,
		InventoryCategory.InventoryLayeredClothing,
		InventoryCategory.InventoryHeadsAndBodies,
		InventoryCategory.InventoryFaces,
		InventoryCategory.InventoryAnimations,
		InventoryCategory.InventoryGear,
	].includes(option.id),
)

export const emoteCategory =
	inventoryCategories.find(
		(option) => option.id === InventoryCategory.InventoryEmotes,
	) ?? otherItemsCategory

export function avatarItemCategory(
	category: InventoryCategory,
): InventoryCategoryOption {
	return (
		inventoryCategories.find((option) => option.id === category) ??
		otherItemsCategory
	)
}

interface WornItemGroup {
	category: InventoryCategoryOption
	assets: AvatarAsset[]
}

// groupWornAssets groups the worn assets by category, in the order of the
// inventory, with the assets of unknown types last.
export function groupWornAssets(assets: AvatarAsset[]): WornItemGroup[] {
	const groups = wearableCategories.map((category) => ({
			category,
			assets: assets.filter((asset) => asset.category === category.id),
		})),
		otherAssets = assets.filter(
			(asset) =>
				!wearableCategories.some((category) => category.id === asset.category),
		)
	groups.push({ category: otherItemsCategory, assets: otherAssets })
	return groups.filter((group) => group.assets.length > 0)
}

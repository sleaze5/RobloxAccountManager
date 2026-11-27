import baseballCapIcon from "@phosphor-icons/core/regular/baseball-cap.svg?raw"
import coatHangerIcon from "@phosphor-icons/core/regular/coat-hanger.svg?raw"
import cubeIcon from "@phosphor-icons/core/regular/cube.svg?raw"
import diamondIcon from "@phosphor-icons/core/regular/diamond.svg?raw"
import handWavingIcon from "@phosphor-icons/core/regular/hand-waving.svg?raw"
import imageIcon from "@phosphor-icons/core/regular/image.svg?raw"
import mapTrifoldIcon from "@phosphor-icons/core/regular/map-trifold.svg?raw"
import medalIcon from "@phosphor-icons/core/regular/medal.svg?raw"
import musicNotesIcon from "@phosphor-icons/core/regular/music-notes.svg?raw"
import packageIcon from "@phosphor-icons/core/regular/package.svg?raw"
import personIcon from "@phosphor-icons/core/regular/person.svg?raw"
import personSimpleRunIcon from "@phosphor-icons/core/regular/person-simple-run.svg?raw"
import smileyIcon from "@phosphor-icons/core/regular/smiley.svg?raw"
import swordIcon from "@phosphor-icons/core/regular/sword.svg?raw"
import tShirtIcon from "@phosphor-icons/core/regular/t-shirt.svg?raw"
import ticketIcon from "@phosphor-icons/core/regular/ticket.svg?raw"
import { CatalogItemType, InventoryCategory } from "../backend/bridge"
import type { InventoryItem } from "../backend/bridge"
import {
	robloxBadgeUrl,
	robloxBundleUrl,
	robloxCatalogItemUrl,
	robloxGamePassUrl,
	robloxGameUrl,
} from "../shared/roblox-pages"

export interface InventoryCategoryOption {
	id: InventoryCategory
	label: string
	noun: string
	icon: string
	// What leads the second line of each item: its Roblox type in a category
	// that holds several types, or the recent average price of a collectible.
	itemDetail: "type" | "price" | "none"
	// Whether the account may delete its own items of the category. Roblox
	// deletes only assets, and refuses limited ones.
	isDeletable: boolean
}

export const inventoryCategoryGroups: {
	label: string
	categories: InventoryCategoryOption[]
}[] = [
	{
		label: "Avatar",
		categories: [
			{
				id: InventoryCategory.InventoryAccessories,
				label: "Accessories",
				noun: "accessories",
				icon: baseballCapIcon,
				itemDetail: "type",
				isDeletable: true,
			},
			{
				id: InventoryCategory.InventoryClassicClothing,
				label: "Classic clothing",
				noun: "classic clothing",
				icon: tShirtIcon,
				itemDetail: "type",
				isDeletable: true,
			},
			{
				id: InventoryCategory.InventoryLayeredClothing,
				label: "Layered clothing",
				noun: "layered clothing",
				icon: coatHangerIcon,
				itemDetail: "type",
				isDeletable: true,
			},
			{
				id: InventoryCategory.InventoryHeadsAndBodies,
				label: "Heads and bodies",
				noun: "heads or body parts",
				icon: personIcon,
				itemDetail: "type",
				isDeletable: true,
			},
			{
				id: InventoryCategory.InventoryFaces,
				label: "Faces",
				noun: "faces",
				icon: smileyIcon,
				itemDetail: "none",
				isDeletable: true,
			},
			{
				id: InventoryCategory.InventoryBundles,
				label: "Bundles",
				noun: "bundles",
				icon: packageIcon,
				itemDetail: "type",
				isDeletable: false,
			},
			{
				id: InventoryCategory.InventoryEmotes,
				label: "Emotes",
				noun: "emotes",
				icon: handWavingIcon,
				itemDetail: "none",
				isDeletable: true,
			},
			{
				id: InventoryCategory.InventoryAnimations,
				label: "Animations",
				noun: "animations",
				icon: personSimpleRunIcon,
				itemDetail: "type",
				isDeletable: true,
			},
		],
	},
	{
		label: "Items",
		categories: [
			{
				id: InventoryCategory.InventoryCollectibles,
				label: "Collectibles",
				noun: "collectibles",
				icon: diamondIcon,
				itemDetail: "price",
				isDeletable: false,
			},
			{
				id: InventoryCategory.InventoryGear,
				label: "Gear",
				noun: "gear",
				icon: swordIcon,
				itemDetail: "none",
				isDeletable: true,
			},
		],
	},
	{
		label: "Games",
		categories: [
			{
				id: InventoryCategory.InventoryBadges,
				label: "Badges",
				noun: "badges",
				icon: medalIcon,
				itemDetail: "none",
				isDeletable: false,
			},
			{
				id: InventoryCategory.InventoryGamePasses,
				label: "Game passes",
				noun: "game passes",
				icon: ticketIcon,
				itemDetail: "none",
				isDeletable: false,
			},
			{
				id: InventoryCategory.InventoryPlaces,
				label: "Places",
				noun: "places",
				icon: mapTrifoldIcon,
				itemDetail: "none",
				isDeletable: false,
			},
		],
	},
	{
		label: "Creations",
		categories: [
			{
				id: InventoryCategory.InventoryModels,
				label: "Models",
				noun: "models",
				icon: cubeIcon,
				itemDetail: "none",
				isDeletable: true,
			},
			{
				id: InventoryCategory.InventoryDecals,
				label: "Decals",
				noun: "decals",
				icon: imageIcon,
				itemDetail: "none",
				isDeletable: true,
			},
			{
				id: InventoryCategory.InventoryAudio,
				label: "Audio",
				noun: "audio",
				icon: musicNotesIcon,
				itemDetail: "none",
				isDeletable: true,
			},
		],
	},
]

export const inventoryCategories = inventoryCategoryGroups.flatMap(
	(group) => group.categories,
)

// otherAssetsCategory holds the assets that no category in the inventory
// lists, such as the meshes and images that an item uses.
const otherAssetsCategory: InventoryCategoryOption = {
	id: InventoryCategory.InventoryOtherAssets,
	label: "Assets",
	noun: "assets",
	icon: cubeIcon,
	itemDetail: "type",
	isDeletable: false,
}

export function inventoryCategoryOptionOf(
	category: InventoryCategory,
): InventoryCategoryOption {
	return (
		inventoryCategories.find((option) => option.id === category) ??
		otherAssetsCategory
	)
}

// Badges, game passes, and bundles have IDs of their own, which can match the
// ID of an asset. Every other category lists assets.
const categoriesWithOwnIDs = new Set<InventoryCategory>([
	InventoryCategory.InventoryBadges,
	InventoryCategory.InventoryGamePasses,
	InventoryCategory.InventoryBundles,
])

export const assetCategories: readonly InventoryCategory[] = Object.values(
	InventoryCategory,
).filter((category) => !categoriesWithOwnIDs.has(category))

// Labels for the Roblox asset and bundle types that the inventory lists.
const typeLabels: Record<string, string> = {
	Hat: "Hat",
	HairAccessory: "Hair",
	FaceAccessory: "Face",
	NeckAccessory: "Neck",
	ShoulderAccessory: "Shoulder",
	FrontAccessory: "Front",
	BackAccessory: "Back",
	WaistAccessory: "Waist",
	TShirt: "T-shirt",
	Shirt: "Shirt",
	Pants: "Pants",
	TShirtAccessory: "T-shirt",
	ShirtAccessory: "Shirt",
	PantsAccessory: "Pants",
	JacketAccessory: "Jacket",
	SweaterAccessory: "Sweater",
	ShortsAccessory: "Shorts",
	LeftShoeAccessory: "Left shoe",
	RightShoeAccessory: "Right shoe",
	DressSkirtAccessory: "Dress or skirt",
	DynamicHead: "Head",
	Head: "Classic head",
	EyebrowAccessory: "Eyebrows",
	EyelashAccessory: "Eyelashes",
	Torso: "Torso",
	RightArm: "Right arm",
	LeftArm: "Left arm",
	LeftLeg: "Left leg",
	RightLeg: "Right leg",
	ClimbAnimation: "Climb",
	FallAnimation: "Fall",
	IdleAnimation: "Idle",
	JumpAnimation: "Jump",
	RunAnimation: "Run",
	SwimAnimation: "Swim",
	WalkAnimation: "Walk",
	MoodAnimation: "Mood",
	Face: "Face",
	EarAccessory: "Ear",
	EyeAccessory: "Eye",
	DeathAnimation: "Death",
	PoseAnimation: "Pose",
	EmoteAnimation: "Emote",
	Gear: "Gear",
	Image: "Image",
	Mesh: "Mesh",
	MeshPart: "Mesh part",
	Decal: "Decal",
	Model: "Model",
	Package: "Package",
	Animation: "Animation",
	Audio: "Audio",
	Video: "Video",
	Place: "Place",
	Lua: "Script",
	Plugin: "Plugin",
	FontFamily: "Font",
	BodyParts: "Body",
	AvatarAnimations: "Animations",
	Shoes: "Shoes",
}

export function inventoryTypeLabel(typeName: string): string {
	return typeLabels[typeName] ?? ""
}

// inventoryItemPage names the ID of an item and the Roblox page that shows
// it. An asset outside the catalog, such as a decal, opens its catalog page,
// which Roblox sends to the page of its kind.
export function inventoryItemPage(
	categoryId: InventoryCategory,
	item: InventoryItem,
): { idLabel: string; pageUrl: string } {
	if (categoryId === InventoryCategory.InventoryBadges)
		return { idLabel: "Badge ID", pageUrl: robloxBadgeUrl(item.itemId) }
	if (categoryId === InventoryCategory.InventoryGamePasses)
		return { idLabel: "Game pass ID", pageUrl: robloxGamePassUrl(item.itemId) }
	if (categoryId === InventoryCategory.InventoryPlaces)
		return { idLabel: "Place ID", pageUrl: robloxGameUrl(item.itemId) }
	if (
		categoryId === InventoryCategory.InventoryBundles ||
		item.catalogItemType === CatalogItemType.CatalogBundle
	)
		return { idLabel: "Bundle ID", pageUrl: robloxBundleUrl(item.itemId) }
	return { idLabel: "Asset ID", pageUrl: robloxCatalogItemUrl(item.itemId) }
}

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
import { InventoryCategory } from "../backend/bridge"

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
		label: "Experiences",
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
	BodyParts: "Body",
	AvatarAnimations: "Animations",
	Shoes: "Shoes",
}

export function inventoryTypeLabel(typeName: string): string {
	return typeLabels[typeName] ?? ""
}

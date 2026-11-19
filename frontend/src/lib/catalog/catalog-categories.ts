import backpackIcon from "@phosphor-icons/core/regular/backpack.svg?raw"
import baseballCapIcon from "@phosphor-icons/core/regular/baseball-cap.svg?raw"
import beltIcon from "@phosphor-icons/core/regular/belt.svg?raw"
import birdIcon from "@phosphor-icons/core/regular/bird.svg?raw"
import coatHangerIcon from "@phosphor-icons/core/regular/coat-hanger.svg?raw"
import crownIcon from "@phosphor-icons/core/regular/crown.svg?raw"
import dressIcon from "@phosphor-icons/core/regular/dress.svg?raw"
import handWavingIcon from "@phosphor-icons/core/regular/hand-waving.svg?raw"
import hoodieIcon from "@phosphor-icons/core/regular/hoodie.svg?raw"
import linkIcon from "@phosphor-icons/core/regular/link.svg?raw"
import mountainsIcon from "@phosphor-icons/core/regular/mountains.svg?raw"
import packageIcon from "@phosphor-icons/core/regular/package.svg?raw"
import paintBrushIcon from "@phosphor-icons/core/regular/paint-brush.svg?raw"
import pantsIcon from "@phosphor-icons/core/regular/pants.svg?raw"
import personArmsSpreadIcon from "@phosphor-icons/core/regular/person-arms-spread.svg?raw"
import personIcon from "@phosphor-icons/core/regular/person.svg?raw"
import personSimpleRunIcon from "@phosphor-icons/core/regular/person-simple-run.svg?raw"
import scissorsIcon from "@phosphor-icons/core/regular/scissors.svg?raw"
import shieldIcon from "@phosphor-icons/core/regular/shield.svg?raw"
import shirtFoldedIcon from "@phosphor-icons/core/regular/shirt-folded.svg?raw"
import sneakerIcon from "@phosphor-icons/core/regular/sneaker.svg?raw"
import squaresFourIcon from "@phosphor-icons/core/regular/squares-four.svg?raw"
import sunglassesIcon from "@phosphor-icons/core/regular/sunglasses.svg?raw"
import swordIcon from "@phosphor-icons/core/regular/sword.svg?raw"
import tShirtIcon from "@phosphor-icons/core/regular/t-shirt.svg?raw"
import userFocusIcon from "@phosphor-icons/core/regular/user-focus.svg?raw"
import { CatalogCategory } from "../backend/bridge"

export interface CatalogCategoryOption {
	id: CatalogCategory
	label: string
	noun: string
	icon: string
}

interface CatalogCategoryGroup {
	label: string
	// parent lists every item of the group, and the other categories narrow it.
	parent: CatalogCategoryOption
	categories: CatalogCategoryOption[]
}

export const allCatalogItemsCategory: CatalogCategoryOption = {
	id: CatalogCategory.CatalogAll,
	label: "All items",
	noun: "items",
	icon: squaresFourIcon,
}

export const catalogCategoryGroups: CatalogCategoryGroup[] = [
	{
		label: "Accessories",
		parent: {
			id: CatalogCategory.CatalogAccessories,
			label: "All accessories",
			noun: "accessories",
			icon: crownIcon,
		},
		categories: [
			{
				id: CatalogCategory.CatalogHeadAccessories,
				label: "Head",
				noun: "head accessories",
				icon: baseballCapIcon,
			},
			{
				id: CatalogCategory.CatalogFaceAccessories,
				label: "Face",
				noun: "face accessories",
				icon: sunglassesIcon,
			},
			{
				id: CatalogCategory.CatalogNeckAccessories,
				label: "Neck",
				noun: "neck accessories",
				icon: linkIcon,
			},
			{
				id: CatalogCategory.CatalogShoulderAccessories,
				label: "Shoulder",
				noun: "shoulder accessories",
				icon: birdIcon,
			},
			{
				id: CatalogCategory.CatalogFrontAccessories,
				label: "Front",
				noun: "front accessories",
				icon: shieldIcon,
			},
			{
				id: CatalogCategory.CatalogBackAccessories,
				label: "Back",
				noun: "back accessories",
				icon: backpackIcon,
			},
			{
				id: CatalogCategory.CatalogWaistAccessories,
				label: "Waist",
				noun: "waist accessories",
				icon: beltIcon,
			},
			{
				id: CatalogCategory.CatalogGear,
				label: "Gear",
				noun: "gear",
				icon: swordIcon,
			},
		],
	},
	{
		label: "Clothing",
		parent: {
			id: CatalogCategory.CatalogClothing,
			label: "All clothing",
			noun: "clothing",
			icon: coatHangerIcon,
		},
		categories: [
			{
				id: CatalogCategory.CatalogShirts,
				label: "Shirts",
				noun: "shirts",
				icon: shirtFoldedIcon,
			},
			{
				id: CatalogCategory.CatalogTShirts,
				label: "T-shirts",
				noun: "T-shirts",
				icon: tShirtIcon,
			},
			{
				id: CatalogCategory.CatalogSweaters,
				label: "Sweaters",
				noun: "sweaters",
				icon: hoodieIcon,
			},
			{
				id: CatalogCategory.CatalogJackets,
				label: "Jackets",
				noun: "jackets",
				icon: hoodieIcon,
			},
			{
				id: CatalogCategory.CatalogPants,
				label: "Pants",
				noun: "pants",
				icon: pantsIcon,
			},
			{
				id: CatalogCategory.CatalogShorts,
				label: "Shorts",
				noun: "shorts",
				icon: pantsIcon,
			},
			{
				id: CatalogCategory.CatalogDressesAndSkirts,
				label: "Dresses and skirts",
				noun: "dresses or skirts",
				icon: dressIcon,
			},
			{
				id: CatalogCategory.CatalogBodysuits,
				label: "Bodysuits",
				noun: "bodysuits",
				icon: personIcon,
			},
			{
				id: CatalogCategory.CatalogShoes,
				label: "Shoes",
				noun: "shoes",
				icon: sneakerIcon,
			},
			{
				id: CatalogCategory.CatalogClassicShirts,
				label: "Classic shirts",
				noun: "classic shirts",
				icon: shirtFoldedIcon,
			},
			{
				id: CatalogCategory.CatalogClassicTShirts,
				label: "Classic T-shirts",
				noun: "classic T-shirts",
				icon: tShirtIcon,
			},
			{
				id: CatalogCategory.CatalogClassicPants,
				label: "Classic pants",
				noun: "classic pants",
				icon: pantsIcon,
			},
		],
	},
	{
		label: "Body",
		parent: {
			id: CatalogCategory.CatalogBody,
			label: "All body",
			noun: "body items",
			icon: personArmsSpreadIcon,
		},
		categories: [
			{
				id: CatalogCategory.CatalogHair,
				label: "Hair",
				noun: "hair",
				icon: scissorsIcon,
			},
			{
				id: CatalogCategory.CatalogHeads,
				label: "Heads",
				noun: "heads",
				icon: userFocusIcon,
			},
			{
				id: CatalogCategory.CatalogFullBodies,
				label: "Full bodies",
				noun: "bodies",
				icon: packageIcon,
			},
			{
				id: CatalogCategory.CatalogMakeup,
				label: "Makeup",
				noun: "makeup",
				icon: paintBrushIcon,
			},
		],
	},
	{
		label: "Animations",
		parent: {
			id: CatalogCategory.CatalogAnimations,
			label: "All animations",
			noun: "animations",
			icon: personSimpleRunIcon,
		},
		categories: [
			{
				id: CatalogCategory.CatalogEmotes,
				label: "Emotes",
				noun: "emotes",
				icon: handWavingIcon,
			},
			{
				id: CatalogCategory.CatalogAnimationBundles,
				label: "Animation packs",
				noun: "animation packs",
				icon: packageIcon,
			},
		],
	},
]

export const backgroundsCategory: CatalogCategoryOption = {
	id: CatalogCategory.CatalogBackgrounds,
	label: "Backgrounds",
	noun: "backgrounds",
	icon: mountainsIcon,
}

const catalogCategories = [
	allCatalogItemsCategory,
	...catalogCategoryGroups.flatMap((group) => [group.parent, ...group.categories]),
	backgroundsCategory,
]

export function catalogCategoryOptionOf(
	category: CatalogCategory,
): CatalogCategoryOption {
	return (
		catalogCategories.find((option) => option.id === category) ??
		allCatalogItemsCategory
	)
}

// catalogGroupOf finds the group whose parent or narrower category is the
// category, so that the parent can mark the branch of the selected category.
export function catalogGroupOf(
	category: CatalogCategory,
): CatalogCategoryGroup | undefined {
	return catalogCategoryGroups.find(
		(group) =>
			group.parent.id === category ||
			group.categories.some((option) => option.id === category),
	)
}

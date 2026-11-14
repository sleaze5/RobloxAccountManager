<script lang="ts">
	import funnelIcon from "@phosphor-icons/core/regular/funnel.svg?raw"
	import magnifyingGlassIcon from "@phosphor-icons/core/regular/magnifying-glass.svg?raw"
	import xIcon from "@phosphor-icons/core/regular/x.svg?raw"
	import { onMount } from "svelte"
	import type { InventoryCategory, InventoryItem } from "../backend/bridge"
	import InventoryItemTile from "../inventory/InventoryItemTile.svelte"
	import { shapeInventoryItems } from "../inventory/inventory-filters"
	import type { InventoryState } from "../inventory/inventory-state.svelte"
	import Dialog from "../shared/Dialog.svelte"
	import Icon from "../shared/Icon.svelte"
	import { searchTerms } from "../shared/search"
	import { emoteCategory, wearableCategories } from "./avatar-categories"
	import { isWornInDraft } from "./avatar-draft"
	import type { AvatarState } from "./avatar-state.svelte"

	let {
		avatar,
		inventory,
		emoteSlot,
		onClose,
	}: {
		avatar: AvatarState
		// inventory lists what the account owns.
		inventory: InventoryState
		// emoteSlot is the slot that the picked emote goes in, or null when the
		// picker adds items to the avatar.
		emoteSlot: number | null
		onClose: () => void
	} = $props()

	const skeletonTileIndexes = Array.from({ length: 10 }, (_unused, index) => index),
		exactNumbers = new Intl.NumberFormat("en-US")

	let filterText = $state(""),
		filterInput = $state<HTMLInputElement | undefined>(undefined)

	const categories = $derived(
			emoteSlot === null ? wearableCategories : [emoteCategory],
		),
		category = $derived(
			categories.find((option) => option.id === inventory.category) ??
				categories[0],
		),
		page = $derived(inventory.page),
		items = $derived(
			shapeInventoryItems(page?.items ?? [], {
				searchTerms: searchTerms(filterText),
				typeName: "",
				itemFilter: "all",
				wornAssetIds: new Set(),
				sortOrder: "newest",
			}),
		),
		draft = $derived(avatar.draft),
		wornCount = $derived(draft?.assets.length ?? 0)

	function showCategory(categoryId: InventoryCategory): void {
		filterText = ""
		inventory.show(categoryId)
	}

	function maxWornCountOf(item: InventoryItem): number {
		return avatar.maxWornCountByTypeName[item.typeName] ?? 0
	}

	function actionLabelOf(item: InventoryItem): string {
		if (emoteSlot !== null) return `Put in slot ${emoteSlot}`
		if (draft && isWornInDraft(draft, item.itemId)) return "Take off"
		return maxWornCountOf(item) > 0
			? "Wear"
			: "Roblox doesn't allow wearing this type"
	}

	function pick(item: InventoryItem): void {
		if (emoteSlot !== null) {
			avatar.putEmote(emoteSlot, item)
			onClose()
		} else if (draft && isWornInDraft(draft, item.itemId)) {
			avatar.takeOff(item.itemId)
		} else {
			avatar.wear(item, category.id)
		}
	}

	onMount(() => {
		if (!categories.some((option) => option.id === inventory.category)) {
			inventory.show(categories[0].id)
		} else if (!inventory.page) {
			void inventory.refresh()
		}
	})
</script>

<svelte:window
	onkeydown={(event) => {
		if (event.key === "Escape") {
			event.preventDefault()
			onClose()
		}
	}} />

<Dialog
	class="max-w-3xl text-left"
	aria-labelledby="avatar-picker-title"
	aria-describedby="avatar-picker-description"
	onBackdropClick={onClose}>
	<div class="dialog-heading">
		<div>
			<h2 id="avatar-picker-title">
				{emoteSlot === null
					? "Add items"
					: `Choose an emote for slot ${emoteSlot}`}
			</h2>
			<p id="avatar-picker-description">
				{emoteSlot === null
					? "Select items to wear or take off. An item replaces one of the same type when the avatar wears the most that Roblox allows."
					: "Select an emote that the account owns."}
			</p>
		</div>
		<button type="button" aria-label="Close" onclick={onClose}>
			<Icon svg={xIcon} size={14} aria-hidden="true" />
		</button>
	</div>

	<div class="flex h-80 min-h-0 gap-3">
		{#if categories.length > 1}
			<nav
				class="flex w-40 shrink-0 flex-col gap-1 overflow-y-auto"
				aria-label="Item categories">
				{#each categories as option (option.id)}
					<button
						class="nav-item"
						type="button"
						aria-current={category.id === option.id ? "page" : undefined}
						onclick={() => showCategory(option.id)}>
						<Icon svg={option.icon} size={16} aria-hidden="true" />
						<span>{option.label}</span>
					</button>
				{/each}
			</nav>
		{/if}

		<div class="@container flex min-w-0 flex-1 flex-col gap-2">
			<div class="search-field flex-none">
				<Icon svg={magnifyingGlassIcon} size={16} aria-hidden="true" />
				<input
					bind:this={filterInput}
					bind:value={filterText}
					type="search"
					aria-label={`Filter ${category.noun} by name, creator, or ID`}
					placeholder={`Filter ${category.noun}`}
					disabled={!page || page.items?.length === 0} />
				{#if filterText}
					<button
						type="button"
						aria-label="Clear filter"
						onclick={() => {
							filterText = ""
							filterInput?.focus()
						}}>
						<Icon svg={xIcon} size={14} aria-hidden="true" />
					</button>
				{/if}
			</div>

			<div class="min-h-0 flex-1 scrollbar-stable overflow-y-auto">
				{#key category.id}
					{#if !page && inventory.error}
						<div class="empty-state h-full" role="alert">
							<Icon svg={category.icon} size={28} aria-hidden="true" />
							<h2>{category.label} are unavailable</h2>
							<p>{inventory.error}</p>
							<button
								class="button"
								type="button"
								onclick={() => void inventory.refresh()}
								>Try again</button>
						</div>
					{:else if !page}
						<div role="status" aria-busy="true">
							<span class="sr-only">Loading {category.noun}</span>
							<ul
								class="grid grid-cols-3 gap-2 @md:grid-cols-4 @xl:grid-cols-5"
								aria-hidden="true">
								{#each skeletonTileIndexes as index (index)}
									<li
										class="flex flex-col gap-1 rounded-lg border border-line-subtle bg-surface p-1">
										<span
											class="block aspect-square w-full skeleton rounded-sm"
										></span>
										<span class="flex px-1 pb-0.5 text-xs"
											><span class="skeleton-line w-3/4"></span
											></span>
									</li>
								{/each}
							</ul>
						</div>
					{:else if (page.items?.length ?? 0) === 0}
						<div class="empty-state h-full" role="status">
							<Icon svg={category.icon} size={28} aria-hidden="true" />
							<h2>No {category.noun}</h2>
							<p>The account doesn't own any {category.noun} yet.</p>
						</div>
					{:else if items.length === 0}
						<div class="empty-state h-full" role="status">
							<Icon svg={funnelIcon} size={28} aria-hidden="true" />
							<h2>No matching {category.noun}</h2>
							<p>
								Nothing loaded matches the filter. Clear it or load more
								items.
							</p>
						</div>
					{:else}
						{#if inventory.error}
							<p class="mb-2 notice-error" role="alert">
								{inventory.error}
							</p>
						{/if}
						<ul
							class="grid animate-enter grid-cols-3 gap-2 @md:grid-cols-4 @xl:grid-cols-5"
							aria-label={category.label}>
							{#each items as item, index (`${item.itemId}:${index}`)}
								{@const isWorn =
									emoteSlot === null &&
									!!draft &&
									isWornInDraft(draft, item.itemId)}
								<li class="min-w-0">
									<InventoryItemTile
										{item}
										{category}
										{isWorn}
										isDisabled={emoteSlot === null &&
											!isWorn &&
											maxWornCountOf(item) === 0}
										actionLabel={actionLabelOf(item)}
										onOpen={pick} />
								</li>
							{/each}
						</ul>
					{/if}
					{#if page?.nextCursor && (page.items?.length ?? 0) > 0}
						<div class="flex min-h-12 items-center justify-center">
							<button
								class="button"
								type="button"
								aria-busy={inventory.loadingMore}
								disabled={inventory.loading || inventory.loadingMore}
								onclick={() => void inventory.loadMore()}
								>{inventory.loadingMore
									? "Loading…"
									: "Load more"}</button>
						</div>
					{/if}
				{/key}
			</div>
		</div>
	</div>

	{#if emoteSlot === null}
		<div class="dialog-actions items-center">
			<p class="mr-auto text-xs text-fg-muted" aria-live="polite">
				{exactNumbers.format(wornCount)}
				{wornCount === 1 ? "item" : "items"} on the avatar. Save the avatar to keep
				changes.
			</p>
			<button class="button-primary" type="button" onclick={onClose}>Done</button>
		</div>
	{/if}
</Dialog>

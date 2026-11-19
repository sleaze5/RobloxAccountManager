<script lang="ts">
	import { untrack } from "svelte"
	import { CatalogCreatorType, CatalogSalesType } from "../backend/bridge"
	import type { CatalogSearch } from "../backend/bridge"
	import {
		floatMenu,
		menuIn,
		menuOut,
		replayValidationNudge,
	} from "../shared/presence"
	import { defaultCatalogSearch } from "./catalog-store.svelte"

	let {
		anchor,
		search,
		onApply,
		onClose,
	}: {
		anchor: HTMLElement | undefined
		search: CatalogSearch
		onApply: (change: Partial<CatalogSearch>) => void
		onClose: () => void
	} = $props()

	const salesTypeOptions = [
			{ value: CatalogSalesType.CatalogAnySalesType, label: "Any" },
			{ value: CatalogSalesType.CatalogLimited, label: "Limited" },
			{ value: CatalogSalesType.CatalogTimed, label: "Timed" },
		],
		creatorTypeOptions = [
			{ value: CatalogCreatorType.CatalogCreatorUser, label: "User" },
			{ value: CatalogCreatorType.CatalogCreatorGroup, label: "Group" },
		],
		panelId = $props.id()

	// The panel edits a draft, which applies to the search on submit.
	let creatorName = $state(untrack(() => search.creatorName)),
		creatorType = $state(untrack(() => search.creatorType)),
		minPriceText = $state(untrack(() => priceText(search.minPriceInRobux))),
		maxPriceText = $state(untrack(() => priceText(search.maxPriceInRobux))),
		salesType = $state(untrack(() => search.salesType)),
		shouldIncludeHiddenItems = $state(
			untrack(() => search.shouldIncludeHiddenItems),
		),
		priceError = $state(""),
		priceRow = $state<HTMLDivElement | undefined>(undefined)

	function priceText(priceInRobux: number | null): string {
		return priceInRobux === null ? "" : String(priceInRobux)
	}

	// parsePrice reads a whole number of Robux, or null for an empty field.
	// undefined means that the text is not a valid price.
	function parsePrice(text: string): number | null | undefined {
		const trimmedText = text.trim()
		if (trimmedText === "") return null
		if (!/^\d{1,10}$/.test(trimmedText)) return undefined
		return Number(trimmedText)
	}

	function validatePrices(): { min: number | null; max: number | null } | null {
		const min = parsePrice(minPriceText),
			max = parsePrice(maxPriceText)
		if (min === undefined || max === undefined) {
			priceError = "Enter whole numbers of Robux."
		} else if (min !== null && max !== null && min > max) {
			priceError = "Enter a minimum that is not above the maximum."
		} else {
			priceError = ""
			return { min, max }
		}
		return null
	}

	// recheckPrices updates a shown price error while the user types, so that
	// the error does not disappear and move the buttons as a click begins.
	function recheckPrices(): void {
		if (priceError) validatePrices()
	}

	function apply(): void {
		const prices = validatePrices()
		if (!prices) {
			replayValidationNudge(priceRow)
			return
		}
		const trimmedCreatorName = creatorName.trim()
		onApply({
			creatorName: trimmedCreatorName,
			creatorType,
			minPriceInRobux: prices.min,
			maxPriceInRobux: prices.max,
			salesType,
			shouldIncludeHiddenItems,
		})
	}

	function resetDraft(): void {
		const defaults = defaultCatalogSearch()
		creatorName = defaults.creatorName
		creatorType = defaults.creatorType
		minPriceText = maxPriceText = priceError = ""
		salesType = defaults.salesType
		shouldIncludeHiddenItems = defaults.shouldIncludeHiddenItems
	}
</script>

<svelte:window
	onkeydown={(event) => {
		if (event.key === "Escape") {
			event.preventDefault()
			onClose()
		}
	}}
	onpointerdown={(event) => {
		if (
			event.target instanceof Node &&
			!anchor?.contains(event.target) &&
			!document.getElementById(panelId)?.contains(event.target)
		) {
			onClose()
		}
	}} />

<form
	id={panelId}
	class="fixed flex w-80 flex-col gap-3 rounded-xl border border-line-strong bg-overlay p-3 shadow-menu"
	aria-label="Marketplace filters"
	novalidate
	use:floatMenu={{ anchor, align: "end" }}
	in:menuIn
	out:menuOut
	onsubmit={(event) => {
		event.preventDefault()
		apply()
	}}>
	<div class="flex flex-col gap-1">
		<label class="text-xs text-fg-muted" for={`${panelId}-creator`}
			>Creator (optional)</label>
		<div class="flex items-center gap-2">
			<input
				id={`${panelId}-creator`}
				class="field min-w-0 flex-1"
				type="text"
				placeholder="Name"
				maxlength={50}
				autocomplete="off"
				spellcheck="false"
				bind:value={creatorName} />
			<div class="segmented shrink-0" role="group" aria-label="Creator type">
				{#each creatorTypeOptions as option (option.value)}
					<button
						type="button"
						aria-pressed={creatorType === option.value}
						onclick={() => (creatorType = option.value)}
						>{option.label}</button>
				{/each}
			</div>
		</div>
	</div>

	<div class="flex flex-col gap-1">
		<span class="text-xs text-fg-muted" id={`${panelId}-price`}
			>Price in Robux (optional)</span>
		<div
			class="flex items-center gap-2"
			role="group"
			aria-labelledby={`${panelId}-price`}
			bind:this={priceRow}>
			<input
				class="field min-w-0 flex-1 font-mono"
				type="text"
				inputmode="numeric"
				placeholder="Min"
				aria-label="Minimum price in Robux"
				aria-invalid={priceError ? "true" : undefined}
				aria-describedby={priceError ? `${panelId}-price-error` : undefined}
				maxlength={10}
				bind:value={minPriceText}
				oninput={recheckPrices} />
			<span class="text-xs text-fg-faint" aria-hidden="true">to</span>
			<input
				class="field min-w-0 flex-1 font-mono"
				type="text"
				inputmode="numeric"
				placeholder="Max"
				aria-label="Maximum price in Robux"
				aria-invalid={priceError ? "true" : undefined}
				aria-describedby={priceError ? `${panelId}-price-error` : undefined}
				maxlength={10}
				bind:value={maxPriceText}
				oninput={recheckPrices} />
			<button
				class="button shrink-0"
				type="button"
				aria-pressed={minPriceText === "0" && maxPriceText === "0"}
				onclick={() => {
					minPriceText = maxPriceText = "0"
					priceError = ""
				}}>Free</button>
		</div>
		{#if priceError}
			<p
				class="text-xs text-danger-muted"
				id={`${panelId}-price-error`}
				role="alert">
				{priceError}
			</p>
		{/if}
	</div>

	<div class="flex flex-col gap-1">
		<span class="text-xs text-fg-muted" id={`${panelId}-sales`}>Sales type</span>
		<div class="segmented" role="group" aria-labelledby={`${panelId}-sales`}>
			{#each salesTypeOptions as option (option.value)}
				<button
					type="button"
					aria-pressed={salesType === option.value}
					onclick={() => (salesType = option.value)}>{option.label}</button>
			{/each}
		</div>
	</div>

	<div class="flex items-center gap-3">
		<div class="flex min-w-0 flex-1 flex-col">
			<span class="text-xs font-medium text-fg" id={`${panelId}-hidden`}
				>Show hidden items</span>
			<span class="text-2xs text-fg-muted" id={`${panelId}-hidden-description`}
				>Include off-sale items that the Marketplace hides</span>
		</div>
		<label class="switch">
			<input
				type="checkbox"
				role="switch"
				aria-labelledby={`${panelId}-hidden`}
				aria-describedby={`${panelId}-hidden-description`}
				bind:checked={shouldIncludeHiddenItems} />
			<span aria-hidden="true"></span>
		</label>
	</div>

	<div class="flex items-center justify-end gap-2 border-t border-line-subtle pt-3">
		<button class="mr-auto button" type="button" onclick={resetDraft}>Reset</button>
		<button class="button" type="button" onclick={onClose}>Cancel</button>
		<button class="button-primary" type="submit">Apply filters</button>
	</div>
</form>

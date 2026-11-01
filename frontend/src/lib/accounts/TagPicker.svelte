<script lang="ts">
	import checkIcon from "@phosphor-icons/core/regular/check.svg?raw"
	import hashIcon from "@phosphor-icons/core/regular/hash.svg?raw"
	import minusIcon from "@phosphor-icons/core/regular/minus.svg?raw"
	import slidersIcon from "@phosphor-icons/core/regular/sliders.svg?raw"
	import starFillIcon from "@phosphor-icons/core/fill/star-fill.svg?raw"
	import starIcon from "@phosphor-icons/core/regular/star.svg?raw"
	import Icon from "../shared/Icon.svelte"
	import type { TagView } from "../backend/bridge"
	import type { AccountStore } from "./account-store.svelte"
	import { hasTag } from "./account-model"
	import type { Account } from "./account-model"

	let {
		store,
		accounts,
		onManageTags,
	}: {
		store: AccountStore
		accounts: Account[]
		onManageTags: () => void
	} = $props()

	type TagState = "on" | "off" | "mixed"

	function tagState(tag: TagView): TagState {
		const count = accounts.filter((account) => hasTag(account, tag.id)).length
		if (count === 0) return "off"
		return count === accounts.length ? "on" : "mixed"
	}

	function toggle(tag: TagView): void {
		void store.setAccountsTag(
			accounts.map((account) => account.id),
			tag,
			tagState(tag) !== "on",
		)
	}
</script>

{#snippet mark(state: TagState)}
	{#if state === "on"}
		<Icon
			svg={checkIcon}
			size={16}
			class="ml-auto shrink-0 animate-bloom"
			aria-hidden="true" />
	{:else if state === "mixed"}
		<Icon
			svg={minusIcon}
			size={16}
			class="ml-auto shrink-0 animate-bloom"
			aria-hidden="true" />
	{/if}
{/snippet}

<div class="flex flex-col" role="group" aria-label="Account tags">
	{#if store.favoriteTag}
		{@const favorite = store.favoriteTag}
		{@const state = tagState(favorite)}
		<button
			class="menu-item"
			type="button"
			aria-pressed={state === "mixed" ? "mixed" : state === "on"}
			disabled={store.busy}
			onclick={() => toggle(favorite)}>
			<Icon
				svg={state === "on" ? starFillIcon : starIcon}
				size={16}
				aria-hidden="true" />
			<span>Favorite</span>
			{@render mark(state)}
		</button>
	{/if}
	<div class="menu-heading" aria-hidden="true">Tags</div>
	<div class="max-h-52 overflow-y-auto">
		{#each store.customTags as tag (tag.id)}
			{@const state = tagState(tag)}
			<button
				class="menu-item"
				type="button"
				aria-pressed={state === "mixed" ? "mixed" : state === "on"}
				disabled={store.busy}
				onclick={() => toggle(tag)}>
				<Icon svg={hashIcon} size={16} aria-hidden="true" />
				<span>{tag.name}</span>
				{@render mark(state)}
			</button>
		{:else}
			<div class="p-2 text-xs text-fg-faint">No custom tags</div>
		{/each}
	</div>
	<div class="menu-separator"></div>
	<button
		class="menu-item"
		type="button"
		disabled={store.busy}
		onclick={onManageTags}>
		<Icon svg={slidersIcon} size={16} aria-hidden="true" /><span>Manage tags</span>
	</button>
</div>

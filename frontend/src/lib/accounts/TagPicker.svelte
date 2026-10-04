<script lang="ts">
	import CheckIcon from "phosphor-svelte/lib/CheckIcon"
	import HashIcon from "phosphor-svelte/lib/HashIcon"
	import MinusIcon from "phosphor-svelte/lib/MinusIcon"
	import SlidersIcon from "phosphor-svelte/lib/SlidersIcon"
	import StarIcon from "phosphor-svelte/lib/StarIcon"
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
		<CheckIcon size={15} class="menu-check" aria-hidden="true" />
	{:else if state === "mixed"}
		<MinusIcon size={15} class="menu-check" aria-hidden="true" />
	{/if}
{/snippet}

<div class="tag-picker" role="group" aria-label="Account tags">
	{#if store.favoriteTag}
		{@const favorite = store.favoriteTag}
		{@const state = tagState(favorite)}
		<button
			type="button"
			aria-pressed={state === "mixed" ? "mixed" : state === "on"}
			disabled={store.busy}
			onclick={() => toggle(favorite)}>
			<StarIcon
				size={16}
				aria-hidden="true"
				weight={state === "on" ? "fill" : "regular"} />
			<span>Favorite</span>
			{@render mark(state)}
		</button>
	{/if}
	<div class="account-menu-heading" aria-hidden="true">Tags</div>
	<div class="tag-picker-list">
		{#each store.customTags as tag (tag.id)}
			{@const state = tagState(tag)}
			<button
				type="button"
				aria-pressed={state === "mixed" ? "mixed" : state === "on"}
				disabled={store.busy}
				onclick={() => toggle(tag)}>
				<HashIcon size={16} aria-hidden="true" />
				<span>{tag.name}</span>
				{@render mark(state)}
			</button>
		{:else}
			<div class="tag-picker-empty">No custom tags</div>
		{/each}
	</div>
	<div class="account-menu-separator"></div>
	<button type="button" disabled={store.busy} onclick={onManageTags}>
		<SlidersIcon size={16} aria-hidden="true" /><span>Manage tags</span>
	</button>
</div>

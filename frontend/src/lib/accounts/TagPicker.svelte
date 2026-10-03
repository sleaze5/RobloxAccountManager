<script lang="ts">
	import Check from "@lucide/svelte/icons/check"
	import Hash from "@lucide/svelte/icons/hash"
	import Minus from "@lucide/svelte/icons/minus"
	import Settings2 from "@lucide/svelte/icons/settings-2"
	import Star from "@lucide/svelte/icons/star"
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
		<Check size={13} class="menu-check" aria-hidden="true" />
	{:else if state === "mixed"}
		<Minus size={13} class="menu-check" aria-hidden="true" />
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
			<Star
				size={14}
				aria-hidden="true"
				fill={state === "on" ? "currentColor" : "none"} />
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
				<Hash size={14} aria-hidden="true" />
				<span>{tag.name}</span>
				{@render mark(state)}
			</button>
		{:else}
			<div class="tag-picker-empty">No custom tags</div>
		{/each}
	</div>
	<div class="account-menu-separator"></div>
	<button type="button" disabled={store.busy} onclick={onManageTags}>
		<Settings2 size={14} aria-hidden="true" /><span>Manage tags</span>
	</button>
</div>

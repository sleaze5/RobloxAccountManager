<script lang="ts">
	import CheckIcon from "phosphor-svelte/lib/CheckIcon"
	import CaretLeftIcon from "phosphor-svelte/lib/CaretLeftIcon"
	import CaretRightIcon from "phosphor-svelte/lib/CaretRightIcon"
	import HashIcon from "phosphor-svelte/lib/HashIcon"
	import FunnelIcon from "phosphor-svelte/lib/FunnelIcon"
	import CircleNotchIcon from "phosphor-svelte/lib/CircleNotchIcon"
	import PlusIcon from "phosphor-svelte/lib/PlusIcon"
	import ArrowsClockwiseIcon from "phosphor-svelte/lib/ArrowsClockwiseIcon"
	import MagnifyingGlassIcon from "phosphor-svelte/lib/MagnifyingGlassIcon"
	import SlidersIcon from "phosphor-svelte/lib/SlidersIcon"
	import StarIcon from "phosphor-svelte/lib/StarIcon"
	import UsersIcon from "phosphor-svelte/lib/UsersIcon"
	import XIcon from "phosphor-svelte/lib/XIcon"
	import type { WorkspaceState } from "../layout/workspace-state.svelte"
	import SidebarResizer from "../layout/SidebarResizer.svelte"
	import { menuIn, menuOut } from "../shared/presence"
	import AccountList from "./AccountList.svelte"
	import AddAccountMenu from "./AddAccountMenu.svelte"
	import type { AccountStore } from "./account-store.svelte"

	let {
		store,
		workspace,
		onAddAccount,
		onManageTags,
		onBrowser,
	}: {
		store: AccountStore
		workspace: WorkspaceState
		onAddAccount: () => void
		onManageTags: () => void
		onBrowser: (accountId: number | null) => void
	} = $props()
	let filterTrigger = $state<HTMLButtonElement>()

	function constrainTagFilter(node: HTMLElement) {
		const place = () => {
			const host = node.parentElement
			if (!host) return
			const styles = getComputedStyle(node),
				gap = Number.parseFloat(styles.getPropertyValue("--space-1")),
				margin = Number.parseFloat(styles.getPropertyValue("--space-2")),
				top = host.getBoundingClientRect().bottom + gap
			node.style.setProperty(
				"--tag-filter-height",
				`${Math.max(0, window.innerHeight - top - margin)}px`,
			)
		}
		place()
		window.addEventListener("resize", place)
		return { destroy: () => window.removeEventListener("resize", place) }
	}
	function pasteCookie(): void {
		workspace.closeAccountMenus()
		onAddAccount()
	}
	function openBrowser(): void {
		workspace.closeAccountMenus()
		onBrowser(null)
	}
</script>

<aside
	class:collapsed={workspace.sidebarCollapsed}
	class="account-explorer"
	aria-label="Account explorer">
	{#if workspace.sidebarCollapsed}
		<div class="collapsed-sidebar">
			<button
				type="button"
				aria-label="Expand account sidebar"
				data-tooltip="Expand the account sidebar"
				data-tooltip-side="right"
				onclick={() => (workspace.sidebarCollapsed = false)}>
				<CaretRightIcon size={18} aria-hidden="true" />
			</button>
			<UsersIcon size={18} aria-hidden="true" />
			<span>{store.accounts.length}</span>
		</div>
	{:else}
		<div class="explorer-controls">
			<div class="account-title">
				<h1>Accounts <span>({store.accounts.length})</span></h1>
				<div class="account-title-actions">
					<div class="add-account-host" data-account-menu>
						<button
							class:active={workspace.activeAccountMenu === "add-account"}
							type="button"
							aria-label="Add account"
							aria-expanded={workspace.activeAccountMenu ===
								"add-account"}
							data-tooltip="Add account"
							data-tooltip-side="bottom-end"
							disabled={store.busy}
							onclick={() => workspace.toggleAccountMenu("add-account")}>
							<PlusIcon size={16} aria-hidden="true" />
						</button>
					</div>
					<button
						type="button"
						aria-label={store.refreshing
							? "Refreshing accounts"
							: "Refresh accounts"}
						aria-busy={store.refreshing}
						data-tooltip="Refresh accounts"
						data-tooltip-side="bottom-end"
						disabled={!store.vault.unlocked || store.refreshing}
						onclick={() => void store.refreshAccountList()}>
						{#if store.refreshing}
							<CircleNotchIcon
								class="spinner"
								size={16}
								aria-hidden="true" />
						{:else}
							<ArrowsClockwiseIcon size={16} aria-hidden="true" />
						{/if}
					</button>
					<button
						type="button"
						aria-label="Collapse account sidebar"
						data-tooltip="Collapse the account sidebar"
						data-tooltip-side="bottom-end"
						onclick={() => (workspace.sidebarCollapsed = true)}>
						<CaretLeftIcon size={17} aria-hidden="true" />
					</button>
				</div>
				{#if workspace.activeAccountMenu === "add-account"}
					<AddAccountMenu
						disabled={store.busy}
						onPasteCookie={pasteCookie}
						onBrowser={openBrowser} />
				{/if}
			</div>
			<div class="account-filters">
				<div class="search-field">
					<MagnifyingGlassIcon size={17} aria-hidden="true" />
					<input
						value={store.query}
						oninput={(event) =>
							store.setQuery(
								(event.currentTarget as HTMLInputElement).value,
							)}
						type="text"
						placeholder="Search accounts"
						aria-label="Search accounts" />
					{#if store.query}
						<button
							class="search-clear"
							type="button"
							aria-label="Clear account search"
							onclick={() => store.setQuery("")}>
							<XIcon size={14} aria-hidden="true" />
						</button>
					{/if}
				</div>
				<div class="tag-filter-host" data-account-menu>
					<button
						class:active={workspace.activeAccountMenu === "tag-filter"}
						class:filtered={store.selectedTagIds.length > 0}
						class="tag-filter-trigger"
						bind:this={filterTrigger}
						type="button"
						aria-label={store.selectedTagIds.length === 0
							? "Filter accounts by tags"
							: `Filter accounts by tags, ${store.selectedTagIds.length} selected`}
						aria-expanded={workspace.activeAccountMenu === "tag-filter"}
						data-tooltip={store.selectedTagIds.length === 0
							? "Filter by tags"
							: `${store.selectedTagIds.length} tag filters`}
						data-tooltip-side="bottom-end"
						onclick={() => workspace.toggleAccountMenu("tag-filter")}>
						<FunnelIcon size={16} aria-hidden="true" />
						{#if store.selectedTagIds.length > 0}
							<span class="tag-filter-count"
								>{store.selectedTagIds.length}</span>
						{/if}
					</button>
					{#if workspace.activeAccountMenu === "tag-filter"}
						<div
							class="tag-filter-menu account-menu"
							use:constrainTagFilter
							role="group"
							aria-label="Filter accounts by tag"
							in:menuIn
							out:menuOut>
							<div class="tag-filter-heading">
								<strong>Filter by tags</strong>
								{#if store.selectedTagIds.length > 0}
									<button
										class="clear-tag-filters"
										type="button"
										aria-label="Clear filters"
										onclick={() => {
											store.clearSelectedTags()
											filterTrigger?.focus()
										}}>
										<XIcon size={14} aria-hidden="true" /><span
											>Clear</span>
									</button>
								{/if}
							</div>
							<button
								type="button"
								aria-pressed={store.favoriteTag !== null &&
									store.selectedTagIds.includes(store.favoriteTag.id)}
								disabled={!store.favoriteTag}
								onclick={() =>
									store.favoriteTag &&
									store.toggleSelectedTag(store.favoriteTag.id)}>
								<StarIcon
									size={16}
									aria-hidden="true"
									weight={store.favoriteTag &&
									store.selectedTagIds.includes(store.favoriteTag.id)
										? "fill"
										: "regular"} />
								<span>Favorites</span>
								{#if store.favoriteTag && store.selectedTagIds.includes(store.favoriteTag.id)}<CheckIcon
										size={15}
										aria-hidden="true"
										class="menu-check" />{/if}
							</button>
							<div class="account-menu-heading" aria-hidden="true">
								Tags
							</div>
							<div class="tag-filter-list">
								{#each store.customTags as tag (tag.id)}
									<button
										type="button"
										aria-pressed={store.selectedTagIds.includes(
											tag.id,
										)}
										onclick={() => store.toggleSelectedTag(tag.id)}>
										<HashIcon size={16} aria-hidden="true" />
										<span>{tag.name}</span>
										{#if store.selectedTagIds.includes(tag.id)}<CheckIcon
												size={15}
												aria-hidden="true"
												class="menu-check" />{/if}
									</button>
								{:else}
									<div class="tag-picker-empty">No custom tags</div>
								{/each}
							</div>
							<div class="account-menu-separator"></div>
							<button type="button" onclick={() => onManageTags()}>
								<SlidersIcon size={16} aria-hidden="true" /><span
									>Manage tags</span>
							</button>
						</div>
					{/if}
				</div>
			</div>
		</div>

		<div
			class="account-list-region"
			class:multiple-selected={store.multipleSelected}>
			<AccountList {store} {workspace} />
			{#if store.multipleSelected}
				<div class="account-selection-actions">
					<button
						class="control-button"
						type="button"
						in:menuIn
						out:menuOut
						onclick={() => {
							store.clearSelection()
							workspace.closeAccountMenus()
							document
								.querySelector<HTMLButtonElement>(".account-row-select")
								?.focus()
						}}>
						<XIcon size={14} aria-hidden="true" />
						<span>Unselect all</span>
					</button>
				</div>
			{/if}
		</div>
	{/if}
</aside>

<SidebarResizer
	bind:width={workspace.sidebarWidth}
	collapsed={workspace.sidebarCollapsed}
	name="account" />

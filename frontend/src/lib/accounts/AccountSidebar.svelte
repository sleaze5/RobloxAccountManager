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
				gap = Number.parseFloat(styles.getPropertyValue("--spacing")),
				margin = gap * 2,
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
	class="flex min-h-0 w-(--sidebar-width) shrink-0 flex-col bg-surface"
	aria-label="Account explorer">
	{#if workspace.sidebarCollapsed}
		<div class="flex h-full flex-col items-center gap-3 pt-2 text-fg-faint">
			<button
				class="btn-ghost"
				type="button"
				aria-label="Expand account sidebar"
				data-tooltip="Expand the account sidebar"
				data-tooltip-side="right"
				onclick={() => (workspace.sidebarCollapsed = false)}>
				<CaretRightIcon size={16} aria-hidden="true" />
			</button>
			<UsersIcon size={16} aria-hidden="true" />
			<span class="font-mono text-2xs">{store.accountTotal}</span>
		</div>
	{:else}
		<div class="flex shrink-0 flex-col gap-1 px-3 pt-1 pb-2">
			<div class="relative flex h-7 items-center">
				<h1>
					Accounts <span class="font-mono text-2xs font-normal text-fg-faint"
						>({store.accountTotal})</span>
				</h1>
				<div class="ml-auto flex items-center gap-px">
					<div class="relative" data-account-menu>
						<button
							class="btn-ghost"
							data-open={workspace.activeAccountMenu === "add-account" ||
								undefined}
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
						class="btn-ghost"
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
						class="btn-ghost"
						type="button"
						aria-label="Collapse account sidebar"
						data-tooltip="Collapse the account sidebar"
						data-tooltip-side="bottom-end"
						onclick={() => (workspace.sidebarCollapsed = true)}>
						<CaretLeftIcon size={16} aria-hidden="true" />
					</button>
				</div>
				{#if workspace.activeAccountMenu === "add-account"}
					<AddAccountMenu
						disabled={store.busy}
						onPasteCookie={pasteCookie}
						onBrowser={openBrowser} />
				{/if}
			</div>
			<div class="flex items-center gap-1">
				<div class="search-field">
					<MagnifyingGlassIcon size={16} aria-hidden="true" />
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
							type="button"
							aria-label="Clear account search"
							onclick={() => store.setQuery("")}>
							<XIcon size={14} aria-hidden="true" />
						</button>
					{/if}
				</div>
				<div class="relative shrink-0" data-account-menu>
					<button
						class={[
							"relative flex size-8 items-center justify-center rounded-md border bg-transparent text-2xs transition-colors duration-100 hover:border-line-focus hover:bg-hover hover:text-accent-strong data-open:border-line-focus data-open:bg-hover data-open:text-accent-strong",
							store.selectedTagIds.length > 0
								? "border-line-strong text-accent-strong"
								: "border-line text-fg-muted",
						]}
						data-open={workspace.activeAccountMenu === "tag-filter" ||
							undefined}
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
							<span
								class="absolute -top-1 -right-1 grid h-3.5 min-w-3.5 place-items-center rounded-sm border border-surface bg-accent-strong px-0.5 font-mono text-2xs font-bold text-canvas"
								>{store.selectedTagIds.length}</span>
						{/if}
					</button>
					{#if workspace.activeAccountMenu === "tag-filter"}
						<div
							class="absolute top-full right-0 z-10 mt-1 flex max-h-(--tag-filter-height) menu flex-col"
							use:constrainTagFilter
							role="group"
							aria-label="Filter accounts by tag"
							in:menuIn
							out:menuOut>
							<div
								class="flex h-8 shrink-0 items-center justify-between gap-2 px-2 py-1">
								<strong class="text-xs font-semibold text-fg"
									>Filter by tags</strong>
								{#if store.selectedTagIds.length > 0}
									<button
										class="flex h-6 items-center gap-1 rounded-sm border border-line-strong bg-raised px-1 text-2xs font-medium text-fg transition-colors duration-100 hover:bg-raised-hover hover:text-fg-bright"
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
								class="menu-item shrink-0"
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
										size={16}
										aria-hidden="true"
										class="ml-auto shrink-0 animate-pop" />{/if}
							</button>
							<div class="menu-heading shrink-0" aria-hidden="true">
								Tags
							</div>
							<div class="max-h-52 min-h-0 overflow-y-auto">
								{#each store.customTags as tag (tag.id)}
									<button
										class="menu-item"
										type="button"
										aria-pressed={store.selectedTagIds.includes(
											tag.id,
										)}
										onclick={() => store.toggleSelectedTag(tag.id)}>
										<HashIcon size={16} aria-hidden="true" />
										<span>{tag.name}</span>
										{#if store.selectedTagIds.includes(tag.id)}<CheckIcon
												size={16}
												aria-hidden="true"
												class="ml-auto shrink-0 animate-pop" />{/if}
									</button>
								{:else}
									<div class="p-2 text-xs text-fg-faint">
										No custom tags
									</div>
								{/each}
							</div>
							<div class="menu-separator shrink-0"></div>
							<button
								class="menu-item shrink-0"
								type="button"
								onclick={() => onManageTags()}>
								<SlidersIcon size={16} aria-hidden="true" /><span
									>Manage tags</span>
							</button>
						</div>
					{/if}
				</div>
			</div>
		</div>

		<div class="relative grid min-h-0 flex-1 grid-rows-1">
			<AccountList {store} {workspace} />
			{#if store.multipleSelected}
				<div
					class="pointer-events-none absolute inset-x-0 bottom-2 flex justify-center">
					<button
						class="pointer-events-auto btn bg-raised shadow-float hover:bg-raised-hover"
						type="button"
						in:menuIn
						out:menuOut
						onclick={() => {
							store.clearSelection()
							workspace.closeAccountMenus()
							document
								.querySelector<HTMLButtonElement>(
									"[data-account-select]",
								)
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

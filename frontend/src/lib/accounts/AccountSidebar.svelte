<script lang="ts">
	import checkIcon from "@phosphor-icons/core/regular/check.svg?raw"
	import funnelIcon from "@phosphor-icons/core/regular/funnel.svg?raw"
	import hashIcon from "@phosphor-icons/core/regular/hash.svg?raw"
	import magnifyingGlassIcon from "@phosphor-icons/core/regular/magnifying-glass.svg?raw"
	import plusIcon from "@phosphor-icons/core/regular/plus.svg?raw"
	import slidersIcon from "@phosphor-icons/core/regular/sliders.svg?raw"
	import starFillIcon from "@phosphor-icons/core/fill/star-fill.svg?raw"
	import starIcon from "@phosphor-icons/core/regular/star.svg?raw"
	import usersIcon from "@phosphor-icons/core/regular/users.svg?raw"
	import xIcon from "@phosphor-icons/core/regular/x.svg?raw"
	import Icon from "../shared/Icon.svelte"
	import RefreshIcon from "../shared/RefreshIcon.svelte"
	import type { WorkspaceState } from "../layout/workspace-state.svelte"
	import WorkbenchSidebar from "../layout/WorkbenchSidebar.svelte"
	import { menuIn, menuKeyboard, menuOut } from "../shared/presence"
	import AccountList from "./AccountList.svelte"
	import AddAccountMenu from "./AddAccountMenu.svelte"
	import type { AccountStore } from "./account-store.svelte"

	let {
		store,
		workspace,
		onAddAccount,
		onManageTags,
		onBrowser,
		onImportFile,
	}: {
		store: AccountStore
		workspace: WorkspaceState
		onAddAccount: () => void
		onManageTags: () => void
		onBrowser: (accountId: number | null) => void
		onImportFile: () => void
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
	function importFile(): void {
		workspace.closeAccountMenus()
		onImportFile()
	}
</script>

<WorkbenchSidebar
	label="Account explorer"
	name="account"
	icon={usersIcon}
	count={store.accountTotal}
	bind:collapsed={workspace.sidebarCollapsed}
	bind:width={workspace.sidebarWidth}>
	{#snippet title()}Accounts
		{#if store.vault.unlocked}<span
				class="font-mono text-2xs font-normal text-fg-faint"
				>({store.accountTotal})</span
			>{/if}{/snippet}
	{#snippet actions()}
		<div class="relative" data-account-menu>
			<button
				class="button-ghost"
				data-open={workspace.activeAccountMenu === "add-account" || undefined}
				type="button"
				aria-label="Add account"
				aria-expanded={workspace.activeAccountMenu === "add-account"}
				data-tooltip="Add account"
				data-tooltip-side="bottom-end"
				disabled={store.busy}
				onclick={() => workspace.toggleAccountMenu("add-account")}>
				<Icon svg={plusIcon} size={16} aria-hidden="true" />
			</button>
		</div>
		<button
			class="button-ghost"
			type="button"
			aria-label={store.refreshing ? "Refreshing accounts" : "Refresh accounts"}
			aria-busy={store.refreshing}
			data-tooltip="Refresh accounts"
			data-tooltip-side="bottom-end"
			disabled={!store.vault.unlocked || store.refreshing}
			onclick={() => void store.refreshAccountList()}>
			<RefreshIcon isRefreshing={store.refreshing} />
		</button>
	{/snippet}
	{#snippet menu()}
		{#if workspace.activeAccountMenu === "add-account"}
			<AddAccountMenu
				disabled={store.busy}
				onPasteCookie={pasteCookie}
				onBrowser={openBrowser}
				onImportFile={importFile} />
		{/if}
	{/snippet}
	{#snippet tools()}
		<div class="flex items-center gap-1">
			<div class="search-field">
				<Icon svg={magnifyingGlassIcon} size={16} aria-hidden="true" />
				<input
					value={store.query}
					oninput={(event) =>
						store.setQuery((event.currentTarget as HTMLInputElement).value)}
					type="text"
					placeholder="Search accounts"
					aria-label="Search accounts" />
				{#if store.query}
					<button
						type="button"
						aria-label="Clear account search"
						onclick={() => store.setQuery("")}>
						<Icon svg={xIcon} size={14} aria-hidden="true" />
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
					<Icon svg={funnelIcon} size={16} aria-hidden="true" />
					{#if store.selectedTagIds.length > 0}
						<span
							class="absolute -top-1 -right-1 grid h-3.5 min-w-3.5 animate-bloom place-items-center rounded-sm border border-surface bg-accent-strong px-0.5 font-mono text-2xs font-bold text-canvas"
							>{store.selectedTagIds.length}</span>
					{/if}
				</button>
				{#if workspace.activeAccountMenu === "tag-filter"}
					<div
						class="absolute top-full right-0 z-10 mt-1 flex max-h-(--tag-filter-height) menu flex-col"
						use:constrainTagFilter
						use:menuKeyboard
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
									<Icon
										svg={xIcon}
										size={14}
										aria-hidden="true" /><span>Clear</span>
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
							<Icon
								svg={store.favoriteTag &&
								store.selectedTagIds.includes(store.favoriteTag.id)
									? starFillIcon
									: starIcon}
								size={16}
								aria-hidden="true" />
							<span>Favorites</span>
							{#if store.favoriteTag && store.selectedTagIds.includes(store.favoriteTag.id)}<Icon
									svg={checkIcon}
									size={16}
									aria-hidden="true"
									class="ml-auto shrink-0 animate-bloom" />{/if}
						</button>
						<div class="menu-heading shrink-0" aria-hidden="true">Tags</div>
						<div class="max-h-52 min-h-0 overflow-y-auto">
							{#each store.customTags as tag (tag.id)}
								<button
									class="menu-item"
									type="button"
									aria-pressed={store.selectedTagIds.includes(tag.id)}
									onclick={() => store.toggleSelectedTag(tag.id)}>
									<Icon svg={hashIcon} size={16} aria-hidden="true" />
									<span>{tag.name}</span>
									{#if store.selectedTagIds.includes(tag.id)}<Icon
											svg={checkIcon}
											size={16}
											aria-hidden="true"
											class="ml-auto shrink-0 animate-bloom" />{/if}
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
							<Icon svg={slidersIcon} size={16} aria-hidden="true" /><span
								>Manage tags</span>
						</button>
					</div>
				{/if}
			</div>
		</div>
	{/snippet}
	<div class="relative grid min-h-0 flex-1 grid-rows-1">
		<AccountList {store} {workspace} />
	</div>
</WorkbenchSidebar>

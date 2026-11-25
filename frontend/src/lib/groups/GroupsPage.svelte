<script lang="ts">
	import { notifyErrors } from "../notifications/error-notifications.svelte"
	import LoadMore from "../shared/LoadMore.svelte"
	import arrowRightIcon from "@phosphor-icons/core/regular/arrow-right.svg?raw"
	import circleNotchIcon from "@phosphor-icons/core/regular/circle-notch.svg?raw"
	import magnifyingGlassIcon from "@phosphor-icons/core/regular/magnifying-glass.svg?raw"
	import sealCheckIcon from "@phosphor-icons/core/regular/seal-check.svg?raw"
	import starFillIcon from "@phosphor-icons/core/fill/star-fill.svg?raw"
	import usersThreeIcon from "@phosphor-icons/core/regular/users-three.svg?raw"
	import warningIcon from "@phosphor-icons/core/regular/warning.svg?raw"
	import xIcon from "@phosphor-icons/core/regular/x.svg?raw"
	import type { GroupSummary } from "../backend/bridge"
	import type { AccountStore } from "../accounts/account-store.svelte"
	import ConfirmationDialog from "../shared/ConfirmationDialog.svelte"
	import { isMultiSelectGesture } from "../shared/multi-selection.svelte"
	import GroupContextMenu from "./GroupContextMenu.svelte"
	import AccountSwitcher from "../layout/AccountSwitcher.svelte"
	import WorkbenchSidebar from "../layout/WorkbenchSidebar.svelte"
	import type { NotificationCenter } from "../notifications/notification-center.svelte"
	import Icon from "../shared/Icon.svelte"
	import { moveListFocus } from "../shared/move-shortcut"
	import RefreshIcon from "../shared/RefreshIcon.svelte"
	import { matchesSearch, searchTerms } from "../shared/search"
	import SkeletonList from "../shared/SkeletonList.svelte"
	import SkeletonMediaRow from "../shared/SkeletonMediaRow.svelte"
	import GroupGamesPage from "./GroupGamesPage.svelte"
	import GroupIcon from "./GroupIcon.svelte"
	import GroupItemsPage from "./GroupItemsPage.svelte"
	import GroupJoinRequestsPage from "./GroupJoinRequestsPage.svelte"
	import GroupMembersPage from "./GroupMembersPage.svelte"
	import GroupProfileView from "./GroupProfileView.svelte"
	import { minGroupSearchLength } from "./groups-store.svelte"
	import type { GroupsStore, GroupSubpage } from "./groups-store.svelte"
	import { openPageLabel, setProfilePages } from "../shared/profile-pages"
	import type { ProfilePages } from "../shared/profile-pages"
	import StatusTrail from "../layout/StatusTrail.svelte"
	import { tick } from "svelte"

	let {
		store,
		accounts,
		notifications,
		onOpenInBrowser,
		onShowCreations,
	}: {
		store: GroupsStore
		accounts: AccountStore
		notifications: NotificationCenter
		onOpenInBrowser: (accountId: number, pageUrl: string) => void
		onShowCreations: (groupName: string) => void
	} = $props()

	const compactNumbers = new Intl.NumberFormat("en-US", {
		notation: "compact",
		maximumFractionDigits: 1,
	})
	let searchInput = $state<HTMLInputElement | undefined>(undefined),
		detailPane = $state<HTMLElement | undefined>(undefined)

	const isSearchMode = $derived(store.mode === "search"),
		account = $derived(accounts.selectedAccount),
		terms = $derived(isSearchMode ? [] : searchTerms(store.query)),
		myGroups = $derived(store.myGroups?.groups ?? []),
		shownMyGroups = $derived(
			terms.length === 0
				? myGroups
				: myGroups.filter((userGroup) =>
						matchesSearch(
							terms,
							`${userGroup.group.name} ${userGroup.role.name}`,
						),
					),
		),
		rows = $derived<{ group: GroupSummary; detail: string }[]>(
			isSearchMode
				? store.searchResults.map((group) => ({
						group,
						detail: `${compactNumbers.format(group.memberCount)} ${group.memberCount === 1 ? "member" : "members"}`,
					}))
				: shownMyGroups.map((userGroup) => ({
						group: userGroup.group,
						detail: `${userGroup.role.name} · Rank ${userGroup.role.rank}`,
					})),
		),
		isListLoading = $derived(
			isSearchMode
				? store.isSearching && store.searchResults.length === 0
				: store.isLoadingMyGroups && !store.myGroups,
		),
		listError = $derived(isSearchMode ? store.searchError : store.myGroupsError),
		tabStopGroupId = $derived(
			(rows.find((row) => row.group.groupId === store.selectedGroupId) ?? rows[0])
				?.group.groupId,
		)

	$effect(() => {
		if (accounts.vault.unlocked) store.bindAccount(account.id)
	})

	// A new group or subpage starts at the top of the detail pane.
	$effect(() => {
		void store.selectedGroupId
		void store.subpage
		detailPane?.scrollTo({ top: 0 })
	})

	const groupSubpages: GroupSubpage[] = ["members", "games", "items", "joinRequests"]

	const profilePages: ProfilePages = {
		get label() {
			return `${store.profile?.details.group.name ?? "Group"} pages`
		},
		get tabs() {
			return [
				{ id: "members", label: "Members" },
				{ id: "games", label: "Games" },
				{ id: "items", label: "Items" },
				...(store.profile?.membership.permissions.canAcceptJoinRequests
					? [{ id: "joinRequests", label: "Join requests" }]
					: []),
			]
		},
		get currentPage() {
			return store.subpage ?? ""
		},
		openPage: (page) => {
			const subpage = groupSubpages.find((knownSubpage) => knownSubpage === page)
			if (subpage) store.subpage = subpage
		},
		openOverview: () => void closeSubpage(),
	}
	setProfilePages(profilePages)

	async function closeSubpage(): Promise<void> {
		const closedSubpage = store.subpage
		store.subpage = null
		await tick()
		detailPane
			?.querySelector<HTMLButtonElement>(
				`[data-subpage-button="${closedSubpage}"]`,
			)
			?.focus({ preventScroll: true })
	}

	let leavingGroupIds = $state<number[] | null>(null)

	// selectGroup opens the group, or marks it with Ctrl+click, as the account
	// list does.
	function selectGroup(event: MouseEvent | KeyboardEvent, groupId: number): void {
		if (isMultiSelectGesture(event)) {
			store.marked.toggle(groupId, store.selectedGroupId || undefined)
			return
		}
		store.marked.clear()
		store.showGroup(groupId)
	}

	async function leaveGroups(groupIds: number[]): Promise<void> {
		await store.leaveGroups(groupIds)
		leavingGroupIds = null
	}

	function leaveDescription(groupIds: number[]): string {
		if (groupIds.length > 1)
			return `This account will leave ${groupIds.length} groups and lose its roles in them. Joining again may need approval.`
		const name = myGroups.find(
			(userGroup) => userGroup.group.groupId === groupIds[0],
		)?.group.name
		return `This account will leave ${name ?? "the group"} and lose its role. Joining again may need approval.`
	}

	function submitSearch(): void {
		if (store.query.trim().length >= minGroupSearchLength) void store.search()
		else searchInput?.focus()
	}

	notifyErrors("Groups not loaded", () => listError)
</script>

<main class="flex min-h-0 flex-1 flex-col bg-canvas" aria-label="Groups">
	<div
		class="flex min-h-0 flex-1"
		style={`--sidebar-width: ${store.sidebarCollapsed ? 44 : store.sidebarWidth}px;`}>
		<WorkbenchSidebar
			label="Group list"
			name="groups"
			icon={usersThreeIcon}
			count={isSearchMode ? undefined : myGroups.length}
			bind:collapsed={store.sidebarCollapsed}
			bind:width={store.sidebarWidth}>
			{#snippet title()}Groups
				{#if !isSearchMode && store.myGroups}<span
						class="font-mono text-2xs font-normal text-fg-faint"
						>({myGroups.length})</span
					>{/if}{/snippet}
			{#snippet actions()}
				<button
					class="button-ghost"
					type="button"
					aria-label={store.isLoadingMyGroups
						? "Refreshing groups…"
						: "Refresh groups"}
					data-tooltip="Refresh groups"
					aria-busy={store.isLoadingMyGroups}
					disabled={store.isLoadingMyGroups || isSearchMode}
					onclick={() => void store.refreshMyGroups()}>
					<RefreshIcon isRefreshing={store.isLoadingMyGroups} />
				</button>
			{/snippet}
			{#snippet tools()}
				<form
					novalidate
					class="flex items-center gap-1"
					role="search"
					onsubmit={(event) => {
						event.preventDefault()
						if (isSearchMode) submitSearch()
					}}>
					<div class="search-field">
						<Icon
							svg={magnifyingGlassIcon}
							size={16}
							aria-hidden="true" /><input
							type="text"
							placeholder={isSearchMode
								? "Search Roblox groups"
								: "Filter my groups"}
							aria-label={isSearchMode
								? "Search Roblox groups by name, or by group ID with id:"
								: "Filter my groups by name or role"}
							maxlength={50}
							autocomplete="off"
							spellcheck="false"
							bind:this={searchInput}
							data-groups-search
							bind:value={store.query} />{#if store.query}<button
								type="button"
								aria-label="Clear search"
								onclick={() => {
									store.clearSearch()
									searchInput?.focus()
								}}
								><Icon
									svg={xIcon}
									size={14}
									aria-hidden="true" /></button
							>{/if}
					</div>
					{#if isSearchMode}
						<button
							class="button-icon shrink-0"
							type="submit"
							aria-label="Search Roblox groups"
							aria-busy={store.isSearching}
							data-tooltip="Search Roblox groups"
							data-tooltip-side="bottom-end">
							{#if store.isSearching}<Icon
									svg={circleNotchIcon}
									class="spinner"
									size={16}
									aria-hidden="true" />{:else}<Icon
									svg={arrowRightIcon}
									size={16}
									aria-hidden="true" />{/if}
						</button>
					{/if}
				</form>
				<div class="segmented" role="group" aria-label="Show groups">
					<button
						type="button"
						aria-pressed={!isSearchMode}
						onclick={() => store.show("mine")}>
						<Icon svg={usersThreeIcon} size={14} aria-hidden="true" />My
						groups
					</button>
					<button
						type="button"
						aria-pressed={isSearchMode}
						onclick={() => {
							store.show("search")
							searchInput?.focus()
						}}>
						<Icon
							svg={magnifyingGlassIcon}
							size={14}
							aria-hidden="true" />Search
					</button>
				</div>
			{/snippet}
			<div
				class="min-h-0 flex-1 scrollbar-stable overflow-y-auto"
				role="group"
				aria-label={isSearchMode ? "Group search results" : "My groups"}
				data-list
				aria-busy={isListLoading}
				data-load-more-root>
				{#if isListLoading}
					<SkeletonList
						label={isSearchMode ? "Searching groups…" : "Loading groups…"}>
						{#snippet row(index)}<SkeletonMediaRow {index} />{/snippet}
					</SkeletonList>
				{:else if listError && rows.length === 0}
					<div class="empty-state px-3 py-8">
						<Icon svg={warningIcon} size={28} aria-hidden="true" />
						<h2>Groups unavailable</h2>
						<button
							class="button"
							type="button"
							onclick={() =>
								void (isSearchMode
									? store.search()
									: store.refreshMyGroups())}>Try again</button>
					</div>
				{:else if rows.length === 0}
					<div class="empty-state px-3 py-8" role="status">
						<Icon
							svg={isSearchMode ? magnifyingGlassIcon : usersThreeIcon}
							size={28}
							aria-hidden="true" />
						{#if isSearchMode}
							<h2>
								{store.searchedText
									? "No matching groups"
									: "Search Roblox groups"}
							</h2>
							<p>
								{store.searchedText
									? "No group matches this search. Try another name or ID."
									: "Enter at least 2 characters of a group name, or a group ID like id:1234."}
							</p>
						{:else if terms.length > 0}
							<h2>No matching groups</h2>
							<p>Change the search to find a group.</p>
							<button
								class="button"
								type="button"
								onclick={() => {
									store.query = ""
									searchInput?.focus()
								}}>Clear search</button>
						{:else}
							<h2>No groups yet</h2>
							<p>
								@{account.username} is not in any group. Search to find one.
							</p>
							<button
								class="button"
								type="button"
								onclick={() => {
									store.show("search")
									searchInput?.focus()
								}}>Search groups</button>
						{/if}
					</div>
				{:else}
					{#each rows as row (row.group.groupId)}
						{@const isSelected =
							store.selectedGroupId === row.group.groupId}
						<button
							class="list-row relative flex h-12 w-full items-center gap-2 border-b border-line-subtle px-3 text-left"
							type="button"
							data-selected={isSelected ||
								store.marked.has(row.group.groupId) ||
								undefined}
							data-open={store.menu?.groups.some(
								(group) => group.groupId === row.group.groupId,
							) || undefined}
							aria-current={isSelected ? "true" : undefined}
							aria-haspopup="menu"
							data-list-item
							tabindex={row.group.groupId === tabStopGroupId ? 0 : -1}
							onclick={(event) => selectGroup(event, row.group.groupId)}
							oncontextmenu={(event) =>
								store.openMenu(
									event,
									row.group,
									rows.map((listedRow) => listedRow.group),
								)}
							onkeydown={(event) => {
								if (
									event.key === "ContextMenu" ||
									(event.shiftKey && event.key === "F10")
								)
									store.openMenu(
										event,
										row.group,
										rows.map((listedRow) => listedRow.group),
									)
								if (
									(event.key === " " || event.key === "Enter") &&
									isMultiSelectGesture(event)
								) {
									event.preventDefault()
									selectGroup(event, row.group.groupId)
								}
								moveListFocus(event)
							}}>
							<GroupIcon url={row.group.iconUrl} />
							<span class="flex min-w-0 flex-col leading-snug">
								<strong
									class="inline-flex min-w-0 items-center gap-1 text-xs font-semibold text-fg"
									><span class="truncate">{row.group.name}</span
									>{#if row.group.hasVerifiedBadge}<Icon
											svg={sealCheckIcon}
											class="shrink-0 text-fg-muted"
											size={12}
											aria-label="Verified group" />{/if}{#if store.myGroups?.primaryGroupId === row.group.groupId}<Icon
											svg={starFillIcon}
											class="shrink-0 animate-pop text-accent-strong"
											size={12}
											aria-label="Primary group" />{/if}</strong>
								<span class="truncate text-2xs text-fg-muted"
									>{row.detail}</span>
							</span>
						</button>
					{/each}
					{#if isSearchMode && store.searchCursor}
						<LoadMore
							pageKey={store.searchResults.length}
							label="Load more results"
							loadingLabel="Loading more results…"
							isLoadingMore={store.isSearching}
							hasFailed={!!store.searchError}
							onLoadMore={() => void store.loadMoreSearchResults()} />
					{/if}
				{/if}
			</div>
		</WorkbenchSidebar>
		<div
			class={[
				"@container min-w-0 flex-1 border-l border-line-subtle",
				store.subpage ? "overflow-hidden" : "overflow-x-hidden overflow-y-auto",
			]}
			bind:this={detailPane}>
			{#if !store.selectedGroupId}
				<div class="empty-state h-full" role="status">
					<Icon svg={usersThreeIcon} size={28} aria-hidden="true" />
					<h2>Select a group</h2>
					<p>
						Select a group from the list, or search Roblox groups by name or
						ID.
					</p>
				</div>
			{:else if store.subpage === "members"}
				{#key `${store.accountId}:${store.selectedGroupId}`}
					<GroupMembersPage {store} {notifications} />
				{/key}
			{:else if store.subpage === "games"}
				{#key `${store.accountId}:${store.selectedGroupId}`}
					<GroupGamesPage {store} />
				{/key}
			{:else if store.subpage === "items"}
				{#key `${store.accountId}:${store.selectedGroupId}`}
					<GroupItemsPage
						{store}
						{accounts}
						{notifications}
						{onShowCreations} />
				{/key}
			{:else if store.subpage === "joinRequests"}
				{#key `${store.accountId}:${store.selectedGroupId}`}
					<GroupJoinRequestsPage {store} {notifications} />
				{/key}
			{:else}
				{#key store.selectedGroupId}
					<div class="animate-enter">
						<GroupProfileView {store} {onOpenInBrowser} />
					</div>
				{/key}
			{/if}
		</div>
	</div>
	<footer
		class="flex h-6 shrink-0 items-center justify-between border-t border-line-subtle bg-surface pl-3 text-2xs text-fg-faint"
		aria-label="Group list">
		<div class="flex min-w-0 items-center gap-1">
			<AccountSwitcher store={accounts} />
			<StatusTrail
				parts={store.selectedGroupId && store.profile
					? [
							store.profile.details.group.name,
							openPageLabel(profilePages),
						].filter(Boolean)
					: []} />
		</div>
		{#if store.selectedGroupId}
			<span class="shrink-0 pr-3"
				>Group ID: <span class="font-mono">{store.selectedGroupId}</span></span>
		{/if}
	</footer>
</main>

{#if store.menu}
	<GroupContextMenu
		{store}
		menu={store.menu}
		onOpenInBrowser={(pageUrl) => onOpenInBrowser(account.id, pageUrl)}
		onLeave={(groupIds) => (leavingGroupIds = groupIds)} />
{/if}

{#if leavingGroupIds}
	{@const groupIds = leavingGroupIds}
	<ConfirmationDialog
		title={groupIds.length > 1
			? `Leave ${groupIds.length} groups?`
			: "Leave group?"}
		description={leaveDescription(groupIds)}
		confirmLabel={groupIds.length > 1
			? `Leave ${groupIds.length} groups`
			: "Leave group"}
		progressLabel="Leaving…"
		isWorking={store.isLeavingGroups}
		onConfirm={() => void leaveGroups(groupIds)}
		onCancel={() => (leavingGroupIds = null)} />
{/if}

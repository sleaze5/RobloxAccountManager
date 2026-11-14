<script lang="ts">
	import { onDestroy, tick, untrack } from "svelte"
	import arrowRightIcon from "@phosphor-icons/core/regular/arrow-right.svg?raw"
	import caretLeftIcon from "@phosphor-icons/core/regular/caret-left.svg?raw"
	import caretRightIcon from "@phosphor-icons/core/regular/caret-right.svg?raw"
	import circleNotchIcon from "@phosphor-icons/core/regular/circle-notch.svg?raw"
	import handshakeIcon from "@phosphor-icons/core/regular/handshake.svg?raw"
	import magnifyingGlassIcon from "@phosphor-icons/core/regular/magnifying-glass.svg?raw"
	import userPlusIcon from "@phosphor-icons/core/regular/user-plus.svg?raw"
	import usersIcon from "@phosphor-icons/core/regular/users.svg?raw"
	import warningIcon from "@phosphor-icons/core/regular/warning.svg?raw"
	import xIcon from "@phosphor-icons/core/regular/x.svg?raw"
	import listChecksIcon from "@phosphor-icons/core/regular/list-checks.svg?raw"
	import prohibitIcon from "@phosphor-icons/core/regular/prohibit.svg?raw"
	import shieldCheckFillIcon from "@phosphor-icons/core/fill/shield-check-fill.svg?raw"
	import shieldCheckIcon from "@phosphor-icons/core/regular/shield-check.svg?raw"
	import Icon from "../shared/Icon.svelte"
	import SkeletonList from "../shared/SkeletonList.svelte"
	import SkeletonMediaRow from "../shared/SkeletonMediaRow.svelte"
	import CheckboxMark from "../shared/CheckboxMark.svelte"
	import RefreshIcon from "../shared/RefreshIcon.svelte"
	import { PresenceType, SessionState, SocialList } from "../backend/bridge"
	import { moderationLabel } from "../accounts/moderation.svelte"
	import type { SocialUser } from "../backend/bridge"
	import type { AccountStore } from "../accounts/account-store.svelte"
	import { presenceDetail, presenceLabel } from "../accounts/account-model"
	import WorkbenchSidebar from "../layout/WorkbenchSidebar.svelte"
	import { appSettings } from "../settings/settings-store.svelte"
	import OperationError from "../shared/OperationError.svelte"
	import ProfileImage from "../shared/ProfileImage.svelte"
	import PresenceLine from "../accounts/PresenceLine.svelte"
	import Select from "../shared/Select.svelte"
	import { moveListFocus } from "../shared/move-shortcut"
	import { menuIn, menuOut } from "../shared/presence"
	import SocialAvatar from "./SocialAvatar.svelte"
	import SocialMenu from "./SocialMenu.svelte"
	import SocialProfile from "./SocialProfile.svelte"
	import InventoryPage from "../inventory/InventoryPage.svelte"
	import {
		bulkRemovalCopy,
		isBulkRemovalList,
		personName,
		playersView,
	} from "./social-store.svelte"
	import type { SocialListKind, SocialStore, SocialView } from "./social-store.svelte"
	import type { RelationshipActionId } from "./relationship"

	let {
		store,
		accounts,
		onFillLaunch,
		onSearchPlace,
		onSearchUniverse,
		onRelationshipAction,
		onEditNickname,
		onRemoveSelected,
	}: {
		store: SocialStore
		accounts: AccountStore
		onFillLaunch: (placeId: number, jobId: string) => void
		onSearchPlace: (placeId: number) => void
		onSearchUniverse: (universeId: number) => void
		onRelationshipAction: (person: SocialUser, action: RelationshipActionId) => void
		onEditNickname: (person: SocialUser, nickname: string) => void
		onRemoveSelected: () => void
	} = $props()

	const minimumSearchLength = 3,
		idSearch = /^id:\s*\d+$/i,
		listKinds: SocialListKind[] = [
			SocialList.SocialFriends,
			SocialList.SocialRequests,
			SocialList.SocialFollowers,
			SocialList.SocialFollowing,
		],
		views: Record<SocialView, { label: string; noun: string }> = {
			[SocialList.SocialFriends]: { label: "Friends", noun: "friends" },
			[SocialList.SocialFollowers]: { label: "Followers", noun: "followers" },
			[SocialList.SocialFollowing]: {
				label: "Following",
				noun: "followed users",
			},
			[SocialList.SocialRequests]: {
				label: "Requests",
				noun: "friend requests",
			},
			[playersView]: { label: "Player search", noun: "players" },
		}

	let searchInput = $state<HTMLInputElement | undefined>(undefined),
		detailPane = $state<HTMLDivElement | undefined>(undefined),
		searchError = $state("")
	const account = $derived(accounts.selectedAccount),
		accountOptions = $derived(
			accounts.accounts.map((savedAccount) => ({
				value: savedAccount.id,
				label: savedAccount.displayName,
				description: `@${savedAccount.username}`,
				image: savedAccount.avatarUrl,
			})),
		),
		subject = $derived(store.subject),
		owner = $derived(subject ? personName(subject) : `@${account.username}`),
		players = $derived(store.list === playersView),
		requestsOpen = $derived(store.list === SocialList.SocialRequests),
		current = $derived(views[store.list]),
		viewOptions = $derived([
			...listKinds
				.filter((list) => !subject || list !== SocialList.SocialRequests)
				.map((list) => ({
					value: list,
					label: views[list].label,
					group: "Connections",
				})),
			{ value: playersView, label: views[playersView].label, group: "Search" },
		]),
		filterText = $derived(store.query.trim()),
		total = $derived(store.page?.people?.length ?? 0),
		onlineCount = $derived(
			(store.page?.people ?? []).filter(
				(person) =>
					person.presence?.userPresenceType ===
						PresenceType.PresenceTypeOnline ||
					person.presence?.userPresenceType ===
						PresenceType.PresenceTypeInGame ||
					person.presence?.userPresenceType ===
						PresenceType.PresenceTypeInStudio,
			).length,
		),
		selected = $derived(store.selected),
		accountModerated = $derived(account.state === SessionState.StateModerated),
		bulkCopy = $derived(
			isBulkRemovalList(store.list) ? bulkRemovalCopy[store.list] : null,
		),
		visibleUserIds = $derived(store.people.map((person) => person.userId)),
		allVisibleSelected = $derived(
			visibleUserIds.length > 0 &&
				visibleUserIds.every((userId) =>
					store.selectedUserIds.includes(userId),
				),
		)

	// Roblox rejects the social requests of a moderated account, so the page
	// shows its status instead of loading lists that would fail.
	$effect(() => {
		const accountId = account.id,
			selfUserId = Number(account.userId) || 0
		if (!accounts.vault.unlocked) return
		if (accountModerated) {
			untrack(() => store.closeAccount())
			return
		}
		untrack(() => store.show(accountId, selfUserId))
	})

	$effect(() => {
		const presence = appSettings.presence.social
		if (!appSettings.initialized || !accounts.vault.unlocked || !presence.enabled) {
			return
		}
		const timer = window.setInterval(
			() => void store.refreshPresences(),
			presence.intervalSeconds * 1000,
		)
		return () => window.clearInterval(timer)
	})

	onDestroy(() => store.closeMenu())

	async function changeView(view: SocialView): Promise<void> {
		store.showList(view)
		if (view !== playersView) return
		await tick()
		searchInput?.focus()
	}

	function search(): void {
		if (filterText.length < minimumSearchLength && !idSearch.test(filterText)) {
			searchError = players
				? `Enter at least ${minimumSearchLength} characters, or id: and a user ID.`
				: ""
			searchInput?.focus()
			return
		}
		searchError = ""
		store.searchPlayers(filterText)
	}

	async function closeInventory(): Promise<void> {
		store.inventoryOpen = false
		await tick()
		detailPane?.scrollTo({ top: 0 })
		detailPane
			?.querySelector<HTMLButtonElement>("[data-inventory-button]")
			?.focus({ preventScroll: true })
	}

	function subtitle(person: SocialUser): string {
		return person.username ? `@${person.username}` : `ID ${person.userId}`
	}

	function rowLabel(person: SocialUser): string {
		const name = `${personName(person)}${person.username ? ` (@${person.username})` : ""}${person.trusted ? ", trusted friend" : ""}`
		if (person.request) return `${name}, friend request incoming`
		if (!person.presence) return name
		const detail = presenceDetail(person.presence)
		return `${name}, ${presenceLabel(person.presence)}${detail ? `: ${detail}` : ""}`
	}

	function emptyCopy(): { title: string; text: string } {
		switch (store.list) {
			case playersView:
				return { title: "No players found", text: "Try a different name." }
			case SocialList.SocialFollowers:
				return { title: "No followers", text: `Nobody follows ${owner} yet.` }
			case SocialList.SocialFollowing:
				return {
					title: "Not following anyone",
					text: `${owner} doesn't follow anyone yet.`,
				}
			case SocialList.SocialRequests:
				return {
					title: "No friend requests",
					text: `New friend requests to ${owner} appear here.`,
				}
			default:
				return subject
					? { title: "No friends", text: `${owner} has no friends yet.` }
					: {
							title: "No friends yet",
							text: `Friends that ${owner} adds on Roblox appear here.`,
						}
		}
	}

	function statusSummary(): string {
		const more = store.page?.nextCursor ? "+" : ""
		const online =
			store.list === SocialList.SocialFriends ? `, ${onlineCount} online` : ""
		return `${total}${more} ${current.noun}${online}`
	}
</script>

{#snippet personSummary(person: SocialUser, detail: string)}
	<span
		class={["flex w-full min-w-0 gap-2", detail ? "items-start" : "items-center"]}>
		<SocialAvatar
			imageUrl={person.imageUrl}
			presence={person.presence ?? undefined}
			className="size-9" />
		<span class="flex min-w-0 flex-col leading-snug">
			<strong
				class="flex min-w-0 items-center gap-1 text-sm font-semibold text-fg">
				<span class="truncate">{personName(person)}</span>
				{#if person.trusted}<Icon
						svg={shieldCheckFillIcon}
						class="shrink-0 animate-pop text-accent-strong"
						size={12}
						aria-label="Trusted friend" />{/if}
			</strong>
			<small
				class="truncate text-2xs text-fg-faint group-hover/row:text-fg-muted group-data-selected/row:text-fg-muted"
				>{subtitle(person)}</small>
			{#if detail && person.presence}<PresenceLine
					presence={person.presence} />{/if}
		</span>
	</span>
{/snippet}

<main class="flex min-h-0 flex-1 flex-col bg-canvas" aria-label="Social">
	<OperationError {store} />
	<div
		class="flex min-h-0 flex-1"
		style={`--sidebar-width: ${store.sidebarCollapsed ? 44 : store.sidebarWidth}px;`}>
		<WorkbenchSidebar
			label="People"
			name="social"
			icon={handshakeIcon}
			bind:collapsed={store.sidebarCollapsed}
			bind:width={store.sidebarWidth}>
			{#snippet title()}Social{/snippet}
			{#snippet actions()}
				{#if store.canRemoveInBulk && !accountModerated}
					<button
						class="button-ghost"
						type="button"
						aria-label={`Select ${current.noun}`}
						aria-pressed={store.selecting}
						data-open={store.selecting || undefined}
						data-tooltip={store.selecting
							? "Stop selecting"
							: `Select ${current.noun} to ${bulkCopy?.action.toLowerCase() ?? "remove"}`}
						data-tooltip-side="bottom-end"
						disabled={account.id === 0 ||
							!store.page ||
							store.removingSelected}
						onclick={() => store.setSelecting(!store.selecting)}>
						<Icon svg={listChecksIcon} size={16} aria-hidden="true" />
					</button>
				{/if}
				<button
					class="button-ghost"
					type="button"
					aria-label={store.loading ? "Refreshing list" : "Refresh list"}
					aria-busy={store.loading}
					data-tooltip="Refresh list"
					data-tooltip-side="bottom-end"
					disabled={account.id === 0 || store.loading || accountModerated}
					onclick={() => void store.refresh()}>
					<RefreshIcon isRefreshing={store.loading} />
				</button>
			{/snippet}
			{#snippet tools()}
				<Select
					label="Account used to browse"
					value={account.id}
					options={accountOptions}
					disabled={accountOptions.length === 0}
					className="w-full"
					onChange={(accountId) => accounts.selectAccount(accountId)} />
				{#if subject}
					{@const previous = store.trail.at(-2)}
					{@const backLabel = `Back to ${previous ? personName(previous) : `@${account.username}`}`}
					<nav
						class="flex h-7 min-w-0 animate-enter items-center gap-1 text-xs font-medium"
						aria-label="Browsing trail">
						<button
							class="button-ghost-sm"
							type="button"
							aria-label={backLabel}
							data-tooltip={backLabel}
							onclick={() => store.back()}
							><Icon
								svg={caretLeftIcon}
								data-nudge="left"
								size={14}
								aria-hidden="true" /></button>
						<button
							class="max-w-24 shrink-0 truncate rounded-sm text-fg-faint transition-colors duration-100 hover:text-fg"
							type="button"
							aria-label={`Return to @${account.username}`}
							onclick={() => store.back(0)}>@{account.username}</button>
						<Icon
							svg={caretRightIcon}
							class="shrink-0 text-fg-faint"
							size={12}
							aria-hidden="true" />
						{#if store.trail.length > 1}
							<span class="shrink-0 text-fg-faint" aria-hidden="true"
								>…</span>
							<Icon
								svg={caretRightIcon}
								class="shrink-0 text-fg-faint"
								size={12}
								aria-hidden="true" />
						{/if}
						<span
							class="relative grid size-5 shrink-0 place-items-center overflow-hidden rounded-sm border border-line-strong bg-raised"
							aria-hidden="true"
							><ProfileImage url={subject.imageUrl} /></span>
						<span class="min-w-0 truncate text-fg" aria-current="page"
							>{personName(subject)}</span>
					</nav>
				{/if}
				<Select
					label="List"
					value={store.list}
					options={viewOptions}
					disabled={account.id === 0 || accountModerated}
					className="w-full"
					onChange={(view) => void changeView(view)} />
				{#if store.showsTrustedFilter && store.page && !accountModerated}
					<div
						class="segmented animate-enter"
						role="group"
						aria-label="Show friends">
						<button
							type="button"
							aria-pressed={!store.trustedOnly}
							onclick={() => store.setTrustedOnly(false)}>
							All <span>{total}</span>
						</button>
						<button
							type="button"
							aria-pressed={store.trustedOnly}
							disabled={!store.page.people?.length ||
								store.page.unavailable?.includes("trusted")}
							onclick={() => store.setTrustedOnly(true)}>
							<Icon svg={shieldCheckIcon} size={14} aria-hidden="true" />
							Trusted <span>{store.trustedCount}</span>
						</button>
					</div>
				{/if}
				<form
					novalidate
					role="search"
					onsubmit={(event) => {
						event.preventDefault()
						search()
					}}>
					<div class="search-field">
						<Icon
							svg={magnifyingGlassIcon}
							size={16}
							aria-hidden="true" /><input
							type="text"
							placeholder={players
								? "Search players or id:123"
								: `Filter ${current.noun}`}
							aria-label={players
								? "Search Roblox players by username, display name, or id: and a user ID"
								: `Filter ${current.noun}, or press Enter to search Roblox players`}
							aria-invalid={searchError ? "true" : undefined}
							maxlength={50}
							disabled={account.id === 0 || accountModerated}
							bind:this={searchInput}
							bind:value={store.query}
							oninput={() =>
								(searchError = "")} />{#if store.query}<button
								type="button"
								aria-label="Clear search"
								onclick={() => {
									store.query = ""
									searchError = ""
									searchInput?.focus()
								}}
								><Icon
									svg={xIcon}
									size={14}
									aria-hidden="true" /></button
							>{/if}
					</div>
					{#if searchError}
						<p class="mt-1 text-xs text-danger-muted" role="alert">
							{searchError}
						</p>
					{/if}
				</form>
			{/snippet}
			{#if store.loadError && store.page}
				<p class="px-3 pb-2 text-xs text-danger-muted" role="alert">
					{store.loadError}
				</p>
			{:else if store.page?.unavailable?.includes("presence")}
				<p class="px-3 pb-2 text-xs text-danger-muted" role="alert">
					Online status could not be loaded. Refresh to retry.
				</p>
			{:else if store.page?.unavailable?.includes("trusted")}
				<p class="px-3 pb-2 text-xs text-danger-muted" role="alert">
					Trusted friends could not be loaded. Refresh to retry.
				</p>
			{/if}
			<div class="relative grid min-h-0 flex-1 grid-rows-1">
				<div
					class={[
						"min-h-0 scrollbar-stable overflow-y-auto",
						((players && store.playerQuery) || store.selecting) && "pb-12",
					]}
					aria-label={current.label}
					aria-busy={store.loading || store.loadingMore}>
					{#if !store.page && store.loading}
						<SkeletonList
							label={players
								? "Searching Roblox"
								: `Loading ${current.noun}`}>
							{#snippet row(index)}<SkeletonMediaRow {index} />{/snippet}
						</SkeletonList>
					{/if}
					{#each store.people as person (person.userId)}
						{@const active = store.selectedUserId === person.userId}
						{@const detail = person.presence
							? presenceDetail(person.presence)
							: ""}
						{#if store.selecting}
							{@const checked = store.selectedUserIds.includes(
								person.userId,
							)}
							<label
								class={[
									"group/row list-row relative flex w-full cursor-pointer items-center gap-2 border-b border-line-subtle px-3",
									detail ? "h-16" : "h-12",
								]}
								data-selected={checked || undefined}>
								<span
									class={[
										"checkbox shrink-0",
										detail && "mt-2.5 self-start",
									]}>
									<input
										type="checkbox"
										data-list-item
										aria-label={`Select ${rowLabel(person)}`}
										{checked}
										disabled={store.removingSelected}
										onkeydown={moveListFocus}
										onchange={() =>
											store.toggleSelected(person.userId)} />
									<CheckboxMark />
								</span>
								{@render personSummary(person, detail)}
							</label>
						{:else}
							<button
								class={[
									"group/row list-row relative flex w-full flex-col justify-center border-b border-line-subtle px-3 text-left",
									detail ? "h-16" : "h-12",
								]}
								data-selected={active || undefined}
								data-open={store.menu?.person.userId ===
									person.userId || undefined}
								type="button"
								aria-current={active ? "true" : undefined}
								aria-haspopup="menu"
								data-list-item
								aria-label={rowLabel(person)}
								onclick={() => store.select(person)}
								oncontextmenu={(event) => store.openMenu(event, person)}
								onkeydown={(event) => {
									if (
										event.key === "ContextMenu" ||
										(event.shiftKey && event.key === "F10")
									)
										store.openMenu(event, person)
									moveListFocus(event)
								}}>
								{@render personSummary(person, detail)}
							</button>
						{/if}
					{/each}
					{#if !players && filterText.length >= minimumSearchLength}
						<button
							class="list-row flex h-10 w-full items-center gap-2 border-b border-line-subtle px-3 text-left text-xs text-fg-muted"
							type="button"
							onclick={search}>
							<Icon
								svg={magnifyingGlassIcon}
								class="shrink-0"
								size={16}
								aria-hidden="true" />
							<span class="min-w-0 flex-1 truncate"
								>Search players for <strong class="font-medium text-fg"
									>"{filterText}"</strong
								></span>
							<Icon
								svg={arrowRightIcon}
								class="shrink-0"
								data-nudge="right"
								size={14}
								aria-hidden="true" />
						</button>
					{/if}
					{#if store.page?.nextCursor && (players || !store.query)}
						<div class="flex min-h-12 items-center justify-center">
							<button
								class="button"
								type="button"
								aria-busy={store.loadingMore}
								disabled={store.loadingMore}
								onclick={() => void store.load(true)}
								>{store.loadingMore ? "Loading…" : "Load more"}</button>
						</div>
					{/if}
				</div>
				{#if store.selecting && bulkCopy}
					{@const selectedCount = store.selectedUserIds.length}
					<div
						class="absolute inset-x-0 bottom-0 flex h-11 animate-enter items-center gap-2 border-t border-line bg-overlay px-3"
						role="toolbar"
						aria-label="Selected people">
						<span class="checkbox shrink-0">
							<input
								type="checkbox"
								aria-label={allVisibleSelected
									? "Clear selection"
									: "Select all shown"}
								checked={allVisibleSelected}
								indeterminate={selectedCount > 0 && !allVisibleSelected}
								disabled={visibleUserIds.length === 0 ||
									store.removingSelected}
								onchange={() => store.toggleAllVisible()} />
							<CheckboxMark
								isIndeterminate={selectedCount > 0 &&
									!allVisibleSelected} />
						</span>
						<span
							class="min-w-0 flex-1 truncate text-xs text-fg-muted"
							aria-live="polite"
							><strong class="font-mono font-medium text-fg"
								>{selectedCount}</strong>
							selected</span>
						<button
							class="button"
							type="button"
							disabled={store.removingSelected}
							onclick={() => store.setSelecting(false)}>Cancel</button>
						<button
							class="button-danger"
							type="button"
							disabled={selectedCount === 0 || store.removingSelected}
							onclick={onRemoveSelected}>{bulkCopy.action}</button>
					</div>
				{/if}
				{#if players && store.playerQuery}
					<div
						class="pointer-events-none absolute inset-x-0 bottom-2 flex justify-center">
						<button
							class="pointer-events-auto button bg-raised shadow-float hover:bg-raised-hover"
							type="button"
							in:menuIn
							out:menuOut
							onclick={() => {
								store.clearPlayers()
								searchError = ""
								searchInput?.focus()
							}}>
							<Icon svg={xIcon} size={14} aria-hidden="true" />
							<span>Clear search</span>
						</button>
					</div>
				{/if}
			</div>
		</WorkbenchSidebar>
		<div
			class={[
				"@container min-w-0 flex-1",
				selected && store.inventoryOpen
					? "overflow-hidden"
					: "overflow-x-hidden overflow-y-auto",
			]}
			bind:this={detailPane}>
			{#if selected && store.inventoryOpen}
				{#key `${account.id}:${selected.userId}`}
					<InventoryPage
						{accounts}
						accountId={account.id}
						userId={selected.userId}
						username={selected.username || String(selected.userId)}
						bind:sidebarWidth={store.inventorySidebarWidth}
						{onSearchPlace}
						onBack={() => void closeInventory()} />
				{/key}
			{:else if selected}
				{#key selected.userId}
					<div class="animate-enter">
						<SocialProfile
							{store}
							{accounts}
							person={selected}
							{onFillLaunch}
							{onSearchPlace}
							{onSearchUniverse}
							onEditNickname={(nickname) =>
								onEditNickname(selected, nickname)}
							{onRelationshipAction}
							onOpenInventory={() => (store.inventoryOpen = true)} />
					</div>
				{/key}
			{:else}
				{@const empty = emptyCopy()}
				<div class="empty-state h-full" role="status">
					{#if accountModerated}<Icon
							svg={prohibitIcon}
							class="text-danger-muted"
							size={28}
							aria-hidden="true" />
						<h2>
							@{account.username} is {moderationLabel(
								account.moderation,
							).toLowerCase()}
						</h2>
						<p>
							Roblox blocks friends and player search for this account.
							Choose another account above.
						</p>
					{:else if account.id === 0 && accounts.loadingAccounts}<Icon
							svg={circleNotchIcon}
							class="spinner"
							size={28}
							aria-hidden="true" />
						<h2>Loading accounts…</h2>
					{:else if account.id === 0}<Icon
							svg={usersIcon}
							size={28}
							aria-hidden="true" />
						{#if accountOptions.length}
							<h2>Select an account</h2>
							<p>
								Choose the account to browse friends and players with.
							</p>
						{:else}
							<h2>No accounts yet</h2>
							<p>
								Add an account in Accounts to browse friends and
								players.
							</p>
						{/if}
					{:else if players && !store.playerQuery}<Icon
							svg={magnifyingGlassIcon}
							size={28}
							aria-hidden="true" />
						<h2>Search Roblox players</h2>
						<p>Enter a username or display name, then press Enter.</p>
					{:else if !store.page && store.loading}<Icon
							svg={circleNotchIcon}
							class="spinner"
							size={28}
							aria-hidden="true" />
						<h2>
							{players ? "Searching Roblox…" : `Loading ${current.noun}…`}
						</h2>
					{:else if !store.page && store.loadError}<Icon
							svg={warningIcon}
							size={28}
							aria-hidden="true" />
						<h2>
							{players
								? "Player search is unavailable"
								: "This list is unavailable"}
						</h2>
						<p>{store.loadError}</p>
						<button
							class="button"
							type="button"
							onclick={() => void store.load()}>Try again</button>
					{:else if total === 0}<Icon
							svg={players
								? magnifyingGlassIcon
								: requestsOpen
									? userPlusIcon
									: handshakeIcon}
							size={28}
							aria-hidden="true" />
						<h2>{empty.title}</h2>
						<p>{empty.text}</p>
					{:else if store.people.length === 0}<Icon
							svg={magnifyingGlassIcon}
							size={28}
							aria-hidden="true" />
						<h2>No matches</h2>
						<p>Press Enter to search all of Roblox instead.</p>
					{:else}<Icon
							svg={requestsOpen ? userPlusIcon : handshakeIcon}
							size={28}
							aria-hidden="true" />
						<h2>
							{requestsOpen
								? "Select a friend request"
								: "Select someone"}
						</h2>
						<p>
							{requestsOpen
								? "Choose a request to see who sent it, then accept or decline it."
								: "Choose someone to see their profile. Select their friend or follower counts to browse those lists."}
						</p>
					{/if}
				</div>
			{/if}
		</div>
	</div>
	<footer
		class="flex h-6 shrink-0 items-center justify-between border-t border-line-subtle bg-surface pl-3 text-2xs text-fg-faint"
		aria-label="Social status">
		{#if store.page}
			<span class="ml-auto truncate pr-3" aria-live="polite"
				>{statusSummary()}</span>
		{/if}
	</footer>
</main>

{#if store.menu}
	<SocialMenu
		{store}
		menu={store.menu}
		{onFillLaunch}
		{onSearchPlace}
		{onRelationshipAction} />
{/if}

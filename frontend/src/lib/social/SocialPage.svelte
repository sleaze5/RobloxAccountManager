<script lang="ts">
	import { onDestroy, tick, untrack } from "svelte"
	import arrowRightIcon from "@phosphor-icons/core/regular/arrow-right.svg?raw"
	import arrowsClockwiseIcon from "@phosphor-icons/core/regular/arrows-clockwise.svg?raw"
	import caretLeftIcon from "@phosphor-icons/core/regular/caret-left.svg?raw"
	import caretRightIcon from "@phosphor-icons/core/regular/caret-right.svg?raw"
	import circleNotchIcon from "@phosphor-icons/core/regular/circle-notch.svg?raw"
	import handshakeIcon from "@phosphor-icons/core/regular/handshake.svg?raw"
	import magnifyingGlassIcon from "@phosphor-icons/core/regular/magnifying-glass.svg?raw"
	import userPlusIcon from "@phosphor-icons/core/regular/user-plus.svg?raw"
	import usersIcon from "@phosphor-icons/core/regular/users.svg?raw"
	import warningIcon from "@phosphor-icons/core/regular/warning.svg?raw"
	import xIcon from "@phosphor-icons/core/regular/x.svg?raw"
	import Icon from "../shared/Icon.svelte"
	import { PresenceType, SocialList } from "../backend/bridge"
	import type { SocialUser } from "../backend/bridge"
	import type { AccountStore } from "../accounts/account-store.svelte"
	import {
		presenceDetail,
		presenceLabel,
		presenceLine,
	} from "../accounts/account-model"
	import SidebarResizer from "../layout/SidebarResizer.svelte"
	import { appSettings } from "../settings/settings-store.svelte"
	import OperationError from "../shared/OperationError.svelte"
	import ProfileImage from "../shared/ProfileImage.svelte"
	import Select from "../shared/Select.svelte"
	import { moveListFocus } from "../shared/move-shortcut"
	import { menuIn, menuOut } from "../shared/presence"
	import SocialAvatar from "./SocialAvatar.svelte"
	import SocialMenu from "./SocialMenu.svelte"
	import SocialProfile from "./SocialProfile.svelte"
	import { personName, playersView } from "./social-store.svelte"
	import type { SocialListKind, SocialStore, SocialView } from "./social-store.svelte"

	let {
		store,
		accounts,
		onFillLaunch,
		onSearchPlace,
		onUnfriend,
		onRemoveTrusted,
		onEditNickname,
	}: {
		store: SocialStore
		accounts: AccountStore
		onFillLaunch: (placeId: number, jobId: string) => void
		onSearchPlace: (placeId: number) => void
		onUnfriend: (person: SocialUser) => void
		onRemoveTrusted: (person: SocialUser) => void
		onEditNickname: (person: SocialUser, nickname: string) => void
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
		searchError = $state("")
	const account = $derived(accounts.selectedAccount),
		accountOptions = $derived(
			accounts.accounts.map((item) => ({
				value: item.id,
				label: item.displayName,
				description: `@${item.username}`,
				image: item.avatarUrl,
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
		selected = $derived(store.selected)

	$effect(() => {
		const accountId = account.id,
			selfUserId = Number(account.userId) || 0
		if (!accounts.vault.unlocked) return
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

	function subtitle(person: SocialUser): string {
		return person.username ? `@${person.username}` : `ID ${person.userId}`
	}

	function rowLabel(person: SocialUser): string {
		const name = `${personName(person)}${person.username ? ` (@${person.username})` : ""}`
		if (person.request) return `${name}, friend request`
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

<main class="flex min-h-0 flex-1 flex-col bg-canvas" aria-label="Social">
	<OperationError {store} />
	<div
		class="flex min-h-0 flex-1"
		style={`--sidebar-width: ${store.sidebarCollapsed ? 44 : store.sidebarWidth}px;`}>
		<aside
			class="flex min-h-0 w-(--sidebar-width) shrink-0 flex-col bg-surface"
			aria-label="People">
			{#if store.sidebarCollapsed}
				<div class="flex h-full flex-col items-center gap-3 pt-2 text-fg-faint">
					<button
						class="btn-ghost"
						type="button"
						aria-label="Expand social sidebar"
						data-tooltip="Expand the social sidebar"
						data-tooltip-side="right"
						onclick={() => (store.sidebarCollapsed = false)}
						><Icon
							svg={caretRightIcon}
							data-nudge="right"
							size={16}
							aria-hidden="true" /></button>
					<Icon svg={handshakeIcon} size={16} aria-hidden="true" />
				</div>
			{:else}
				<div class="flex shrink-0 flex-col gap-1 px-3 pt-1 pb-2">
					<div class="relative flex h-7 items-center">
						<h1>Social</h1>
						<div class="ml-auto flex items-center gap-px">
							<button
								class="btn-ghost"
								type="button"
								aria-label={store.loading
									? "Refreshing list"
									: "Refresh list"}
								aria-busy={store.loading}
								data-tooltip="Refresh list"
								data-tooltip-side="bottom-end"
								disabled={account.id === 0 || store.loading}
								onclick={() => void store.refresh()}>
								{#if store.loading}
									<Icon
										svg={circleNotchIcon}
										class="spinner"
										size={16}
										aria-hidden="true" />
								{:else}
									<Icon
										svg={arrowsClockwiseIcon}
										data-turn
										size={16}
										aria-hidden="true" />
								{/if}
							</button>
							<button
								class="btn-ghost"
								type="button"
								aria-label="Collapse social sidebar"
								data-tooltip="Collapse the social sidebar"
								data-tooltip-side="bottom-end"
								onclick={() => (store.sidebarCollapsed = true)}
								><Icon
									svg={caretLeftIcon}
									data-nudge="left"
									size={16}
									aria-hidden="true" /></button>
						</div>
					</div>
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
								class="btn-ghost-sm"
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
								onclick={() => store.back(0)}
								>@{account.username}</button>
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
						disabled={account.id === 0}
						className="w-full"
						onChange={(view) => void changeView(view)} />
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
								disabled={account.id === 0}
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
				</div>
				{#if store.loadError && store.page}
					<p class="px-3 pb-2 text-xs text-danger-muted" role="alert">
						{store.loadError}
					</p>
				{:else if store.page?.unavailable?.includes("presence")}
					<p class="px-3 pb-2 text-xs text-danger-muted" role="alert">
						Online status could not be loaded. Refresh to retry.
					</p>
				{/if}
				<div class="relative grid min-h-0 flex-1 grid-rows-1">
					<div
						class={[
							"min-h-0 scrollbar-stable overflow-y-auto",
							players && store.playerQuery && "pb-12",
						]}
						aria-label={current.label}
						aria-busy={store.loading || store.loadingMore}>
						{#each store.people as person (person.userId)}
							{@const active = store.selectedUserId === person.userId}
							{@const presence = presenceLine(person.presence)}
							<button
								class={[
									"group/row list-row relative flex w-full flex-col justify-center gap-1 border-b border-line-subtle px-3 text-left",
									presence ? "h-16" : "h-12",
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
								<span class="flex w-full min-w-0 items-center gap-2">
									<SocialAvatar
										imageUrl={person.imageUrl}
										presence={person.presence ?? undefined}
										className="size-9" />
									<span class="flex min-w-0 flex-col leading-snug">
										<strong
											class="truncate text-sm font-semibold text-fg"
											>{personName(person)}</strong>
										<small
											class="truncate text-2xs text-fg-faint group-hover/row:text-fg-muted group-data-selected/row:text-fg-muted"
											>{subtitle(person)}</small>
									</span>
								</span>
								{#if presence}
									<small
										class="block w-full truncate text-2xs text-fg-muted"
										>{presence}</small>
								{/if}
							</button>
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
									>Search players for <strong
										class="font-medium text-fg"
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
									class="btn"
									type="button"
									aria-busy={store.loadingMore}
									disabled={store.loadingMore}
									onclick={() => void store.load(true)}
									>{store.loadingMore
										? "Loading…"
										: "Load more"}</button>
							</div>
						{/if}
					</div>
					{#if players && store.playerQuery}
						<div
							class="pointer-events-none absolute inset-x-0 bottom-2 flex justify-center">
							<button
								class="pointer-events-auto btn bg-raised shadow-float hover:bg-raised-hover"
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
			{/if}
		</aside>
		<SidebarResizer
			bind:width={store.sidebarWidth}
			collapsed={store.sidebarCollapsed}
			name="social" />
		<div class="@container min-w-0 flex-1 overflow-x-hidden overflow-y-auto">
			{#if selected}
				{#key selected.userId}
					<div class="animate-enter">
						<SocialProfile
							{store}
							{accounts}
							person={selected}
							{onFillLaunch}
							{onSearchPlace}
							onEditNickname={(nickname) =>
								onEditNickname(selected, nickname)} />
					</div>
				{/key}
			{:else}
				{@const empty = emptyCopy()}
				<div class="empty-state h-full" role="status">
					{#if account.id === 0 && accounts.loadingAccounts}<Icon
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
							class="btn"
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
		{onUnfriend}
		{onRemoveTrusted} />
{/if}

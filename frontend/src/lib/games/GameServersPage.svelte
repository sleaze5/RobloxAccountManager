<script lang="ts">
	import ProfilePageHeader from "../shared/ProfilePageHeader.svelte"
	import arrowSquareOutIcon from "@phosphor-icons/core/regular/arrow-square-out.svg?raw"
	import hashIcon from "@phosphor-icons/core/regular/hash.svg?raw"
	import linkIcon from "@phosphor-icons/core/regular/link.svg?raw"
	import listPlusIcon from "@phosphor-icons/core/regular/list-plus.svg?raw"
	import slidersHorizontalIcon from "@phosphor-icons/core/regular/sliders-horizontal.svg?raw"
	import trashIcon from "@phosphor-icons/core/regular/trash.svg?raw"
	import Icon from "../shared/Icon.svelte"
	import RefreshIcon from "../shared/RefreshIcon.svelte"
	import { untrack } from "svelte"
	import type { AccountStore } from "../accounts/account-store.svelte"
	import type { NotificationCenter } from "../notifications/notification-center.svelte"
	import type { SavedPrivateServer } from "../backend/bridge"
	import AddPrivateServerLinkDialog from "../private-servers/AddPrivateServerLinkDialog.svelte"
	import PrivateServerDialog from "../private-servers/PrivateServerDialog.svelte"
	import RemovePrivateServerLinkDialog from "../private-servers/RemovePrivateServerLinkDialog.svelte"
	import { appSettings } from "../settings/settings-store.svelte"
	import { errorMessage } from "../shared/errors"
	import AnchoredActionMenu, {
		type MenuAction,
	} from "../shared/AnchoredActionMenu.svelte"
	import FriendsInGameList from "./FriendsInGameList.svelte"
	import PrivateServerList from "./PrivateServerList.svelte"
	import RBXServersList from "./RBXServersList.svelte"
	import RecordedServerList from "./RecordedServerList.svelte"
	import RobloxServerList from "./RobloxServerList.svelte"
	import type { ServerSource } from "./game-servers-state.svelte"
	import type { GamesStore } from "./games-store.svelte"
	import { friendServerKey } from "./server-row-menu"

	let {
		store,
		accounts,
		notifications,
		onFillLaunch,
		onFillLaunchLink,
	}: {
		store: GamesStore
		accounts: AccountStore
		notifications: NotificationCenter
		onFillLaunch: (placeId: number, serverId: string) => void
		onFillLaunchLink: (link: string) => void
	} = $props()

	const allSources: { id: ServerSource; label: string; tooltip: string }[] = [
		{
			id: "private",
			label: "Private servers",
			tooltip: "Private servers the account owns or can join",
		},
		{
			id: "roblox",
			label: "Roblox",
			tooltip: "Servers running now, with players, ping, and FPS",
		},
		{
			id: "rovalra",
			label: "RoValra",
			tooltip: "Servers seen by RoValra users, by region and age",
		},
		{
			id: "rbxservers",
			label: "RBXServers",
			tooltip: "Private servers shared publicly on RBXServers",
		},
	]

	const place = untrack(() => store.selected!),
		placeServers = untrack(() =>
			store.serversFor(place, accounts.selectedAccount.id),
		),
		robloxServers = placeServers.roblox,
		friendsInGame = placeServers.friendsInGame,
		recordedServers = placeServers.recorded,
		privateServers = placeServers.private,
		listedServers = placeServers.listed,
		savedServers = placeServers.saved,
		stats = placeServers.stats,
		menuID = $props.id()
	let activeMenu = $state<{
			key: string
			trigger: HTMLElement
			open: boolean
		} | null>(null),
		managing = $state<{
			accountId: number
			username: string
			privateServerId: number
			name: string
		} | null>(null),
		addingLink = $state(false),
		removingLink = $state<SavedPrivateServer | null>(null)
	const sources = $derived(
			allSources.filter(
				(option) =>
					(option.id !== "rovalra" || appSettings.roValraEnabled) &&
					(option.id !== "rbxservers" || appSettings.rbxServersEnabled),
			),
		),
		source = $derived<ServerSource>(
			sources.some((option) => option.id === store.serverSource)
				? store.serverSource
				: "roblox",
		),
		active = $derived(
			{
				roblox: robloxServers,
				private: privateServers,
				rovalra: recordedServers,
				rbxservers: listedServers,
			}[source],
		),
		openKey = $derived(activeMenu?.open ? activeMenu.key : undefined),
		menuActions = $derived(activeMenu ? actionsFor(activeMenu.key) : [])

	// The store keeps the server lists after the page closes, so that it
	// reopens on the same source, options, and page. Opening a list shows its
	// kept page and requests it again only when that page is no longer fresh.
	// The account effect runs first, so that the private list never loads the
	// servers of the account that was selected when the page last closed.
	$effect(() => {
		const accountId = accounts.selectedAccount.id
		untrack(() => {
			privateServers.showAccount(accountId, active === privateServers)
			friendsInGame.showAccount(accountId, active === robloxServers)
		})
	})

	$effect(() => {
		const current = active
		untrack(() => {
			if (current === privateServers && !savedServers.servers) {
				void savedServers.load()
			}
			if (current === robloxServers) void friendsInGame.load()
			if (current === privateServers && accounts.selectedAccount.id === 0) return
			void current.load()
		})
	})

	$effect(() => {
		void active.listVersion
		untrack(closeMenu)
	})

	$effect(() => {
		void appSettings.roValraEnabled
		untrack(() => {
			if (active === robloxServers && robloxServers.page)
				void robloxServers.load()
		})
	})

	// Only the Roblox and RoValra lists show the stats.
	$effect(() => {
		const current = active
		if (appSettings.roValraEnabled) {
			untrack(() => {
				if (
					(current === robloxServers || current === recordedServers) &&
					!stats.isFresh &&
					!stats.loading
				) {
					void stats.load()
				}
			})
		}
	})

	function openMenu(key: string, trigger: HTMLElement, toggle: boolean): void {
		if (toggle && activeMenu?.open && activeMenu.trigger === trigger) closeMenu()
		else activeMenu = { key, trigger, open: true }
	}

	function closeMenu(): void {
		if (activeMenu) activeMenu = { ...activeMenu, open: false }
	}

	function select(action: () => void): () => void {
		return () => {
			closeMenu()
			action()
		}
	}

	function fillLaunchLinkActions(link: string): MenuAction[] {
		return link
			? [
					{
						label: "Fill launch options",
						icon: listPlusIcon,
						onSelect: select(() => onFillLaunchLink(link)),
					},
				]
			: []
	}

	function copyLaunchLinkActions(link: string): MenuAction[] {
		return link
			? [
					{
						label: "Launch link",
						icon: linkIcon,
						section: "Copy",
						accessibleLabel: "Copy launch link",
						onSelect: select(() => void store.copy(link, "Launch link")),
					},
				]
			: []
	}

	async function withListedLink(
		serverId: string,
		use: (link: string) => void | Promise<void>,
	): Promise<void> {
		store.error = ""
		try {
			const link = await listedServers.launchLink(serverId)
			if (link) await use(link)
		} catch (error) {
			store.error = errorMessage(
				error,
				"Could not load the launch link. Try again.",
			)
		}
	}

	function actionsFor(key: string): MenuAction[] {
		if (source === "rbxservers") {
			const server = listedServers.page?.servers?.find(
				(listed) => listed.id === key,
			)
			return server
				? [
						{
							label: "Fill launch options",
							icon: listPlusIcon,
							onSelect: select(
								() => void withListedLink(server.id, onFillLaunchLink),
							),
						},
						{
							label: "Open on RBXServers",
							icon: arrowSquareOutIcon,
							onSelect: select(() => void store.openLink(server.url)),
						},
						{
							label: "Launch link",
							icon: linkIcon,
							section: "Copy",
							accessibleLabel: "Copy launch link",
							onSelect: select(
								() =>
									void withListedLink(server.id, (link) =>
										store.copy(link, "Launch link"),
									),
							),
						},
					]
				: []
		}
		if (source === "private" && key.startsWith("saved-")) {
			const server = savedServers.servers?.find(
				(saved) => `saved-${saved.id}` === key,
			)
			if (!server) return []
			return [
				...fillLaunchLinkActions(server.launchLink),
				...copyLaunchLinkActions(server.launchLink),
				{
					label: "Remove link",
					icon: trashIcon,
					tone: "danger",
					onSelect: select(() => (removingLink = server)),
				},
			]
		}
		if (source === "private") {
			const server = privateServers.page?.servers?.find(
					(owned) => String(owned.privateServerId) === key,
				),
				account = accounts.selectedAccount
			if (!server) return []
			return [
				...fillLaunchLinkActions(server.launchLink),
				...(server.owned
					? [
							{
								label: "Manage server",
								icon: slidersHorizontalIcon,
								onSelect: select(
									() =>
										(managing = {
											accountId: account.id,
											username: account.username,
											privateServerId: server.privateServerId,
											name: server.name,
										}),
								),
							},
						]
					: []),
				...copyLaunchLinkActions(server.launchLink),
			]
		}
		return robloxServerActions(key)
	}

	// A friend may play another place of the universe, so its row joins the
	// place of its own server.
	function robloxServerActions(key: string): MenuAction[] {
		const friendServer = (friendsInGame.friends?.places ?? [])
				.flatMap((friendPlace) =>
					(friendPlace.servers ?? []).map((server) => ({
						placeId: friendPlace.placeId,
						serverId: server.serverId,
					})),
				)
				.find(
					(server) =>
						friendServerKey(server.placeId, server.serverId) === key,
				),
			placeId = friendServer?.placeId ?? place.placeId,
			serverId = friendServer?.serverId ?? key
		return [
			{
				label: "Fill launch options",
				icon: listPlusIcon,
				onSelect: select(() => onFillLaunch(placeId, serverId)),
			},
			{
				label: "Server ID",
				icon: hashIcon,
				section: "Copy",
				accessibleLabel: "Copy server ID",
				onSelect: select(() => void store.copy(serverId, "Server ID")),
			},
		]
	}

	function refresh(): void {
		if (active === privateServers) void savedServers.load()
		if (active === robloxServers) void friendsInGame.refresh()
		if (active === privateServers && !accounts.selectedAccount.id) return
		active.refresh()
		if (appSettings.roValraEnabled) void stats.load()
	}
</script>

<section class="settings-page min-w-0" aria-labelledby="game-servers-title">
	<ProfilePageHeader
		titleId="game-servers-title"
		title="Servers"
		subtitle={store.displayName(place)}>
		<div class="segmented" role="radiogroup" aria-label="Server list">
			{#each sources as option (option.id)}
				<label data-tooltip={option.tooltip} data-tooltip-side="bottom-end">
					<input
						type="radio"
						name={`${menuID}-source`}
						value={option.id}
						checked={source === option.id}
						onchange={() => (store.serverSource = option.id)} />
					<span>{option.label}</span>
				</label>
			{/each}
		</div>
		<button
			class="button-ghost"
			type="button"
			aria-label={active.loading ? "Refreshing servers…" : "Refresh servers"}
			data-tooltip="Refresh servers"
			data-tooltip-side="bottom-end"
			aria-busy={active.loading}
			disabled={active.loading}
			onclick={refresh}>
			<RefreshIcon isRefreshing={active.loading} />
		</button>
	</ProfilePageHeader>

	{#if source === "rovalra"}
		<RecordedServerList
			servers={recordedServers}
			{stats}
			activeServerId={openKey}
			{menuID}
			onMenu={openMenu} />
	{:else if source === "private"}
		<PrivateServerList
			servers={privateServers}
			saved={savedServers}
			{accounts}
			activeKey={openKey}
			{menuID}
			onMenu={openMenu}
			onAddLink={() => (addingLink = true)} />
	{:else if source === "rbxservers"}
		<RBXServersList
			servers={listedServers}
			activeKey={openKey}
			{menuID}
			onMenu={openMenu} />
	{:else}
		{#if accounts.selectedAccount.id !== 0}
			<FriendsInGameList
				{friendsInGame}
				placeId={place.placeId}
				activeKey={openKey}
				{menuID}
				onMenu={openMenu} />
		{/if}
		<RobloxServerList
			servers={robloxServers}
			records={appSettings.roValraEnabled}
			newestVersion={stats.stats?.newestVersion ?? 0}
			activeServerId={openKey}
			{menuID}
			onMenu={openMenu} />
	{/if}
</section>

{#if activeMenu?.open && menuActions.length}
	<AnchoredActionMenu
		id={menuID}
		label="Server actions"
		anchor={activeMenu.trigger}
		actions={menuActions}
		onClose={closeMenu} />
{/if}

{#if addingLink}
	<AddPrivateServerLinkDialog
		saved={savedServers}
		gameName={store.displayName(place)}
		accountId={accounts.selectedAccount.id}
		username={accounts.selectedAccount.username}
		onClose={() => (addingLink = false)} />
{/if}

{#if removingLink}
	<RemovePrivateServerLinkDialog
		saved={savedServers}
		server={removingLink}
		onClose={() => (removingLink = null)} />
{/if}

{#if managing}
	<PrivateServerDialog
		accountId={managing.accountId}
		username={managing.username}
		privateServerId={managing.privateServerId}
		name={managing.name}
		gameName={store.displayName(place)}
		{notifications}
		onChanged={() => privateServers.refresh()}
		onClose={() => (managing = null)} />
{/if}

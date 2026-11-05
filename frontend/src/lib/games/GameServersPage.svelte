<script lang="ts">
	import arrowSquareOutIcon from "@phosphor-icons/core/regular/arrow-square-out.svg?raw"
	import arrowsClockwiseIcon from "@phosphor-icons/core/regular/arrows-clockwise.svg?raw"
	import caretLeftIcon from "@phosphor-icons/core/regular/caret-left.svg?raw"
	import circleNotchIcon from "@phosphor-icons/core/regular/circle-notch.svg?raw"
	import hashIcon from "@phosphor-icons/core/regular/hash.svg?raw"
	import linkIcon from "@phosphor-icons/core/regular/link.svg?raw"
	import listPlusIcon from "@phosphor-icons/core/regular/list-plus.svg?raw"
	import slidersHorizontalIcon from "@phosphor-icons/core/regular/sliders-horizontal.svg?raw"
	import Icon from "../shared/Icon.svelte"
	import { onMount, untrack } from "svelte"
	import type { AccountStore } from "../accounts/account-store.svelte"
	import type { NotificationCenter } from "../notifications/notification-center.svelte"
	import PrivateServerDialog from "../private-servers/PrivateServerDialog.svelte"
	import { appSettings } from "../settings/settings-store.svelte"
	import GameServerMenu, { type ServerMenuAction } from "./GameServerMenu.svelte"
	import PrivateServerList from "./PrivateServerList.svelte"
	import RBXServersList from "./RBXServersList.svelte"
	import RecordedServerList from "./RecordedServerList.svelte"
	import RobloxServerList from "./RobloxServerList.svelte"
	import {
		GameServers,
		ListedServers,
		PrivateServers,
		RecordedServers,
		type ServerSource,
		ServerStatsLoader,
	} from "./game-servers-state.svelte"
	import type { GamesStore } from "./games-store.svelte"

	let {
		store,
		accounts,
		notifications,
		onBack,
		onFillLaunch,
		onFillLaunchLink,
	}: {
		store: GamesStore
		accounts: AccountStore
		notifications: NotificationCenter
		onBack: () => void
		onFillLaunch: (placeId: number, jobId: string) => void
		onFillLaunchLink: (link: string) => void
	} = $props()

	const allSources: { id: ServerSource; label: string; tooltip: string }[] = [
		{
			id: "roblox",
			label: "Roblox",
			tooltip: "Servers running now, with players, ping, and FPS",
		},
		{
			id: "private",
			label: "Private servers",
			tooltip: "Private servers the account owns or can join",
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
		robloxServers = untrack(
			() =>
				new GameServers(
					place.placeId,
					store.serverPages,
					() => appSettings.roValraEnabled,
				),
		),
		recordedServers = untrack(
			() => new RecordedServers(place.placeId, store.recordPages),
		),
		privateServers = untrack(
			() =>
				new PrivateServers(
					place.placeId,
					store.privatePages,
					() => accounts.selectedAccount.id,
				),
		),
		listedServers = untrack(
			() => new ListedServers(place.placeId, store.listedPages),
		),
		stats = untrack(() => new ServerStatsLoader(place.placeId)),
		menuID = $props.id()
	let heading = $state<HTMLHeadingElement | undefined>(undefined),
		activeMenu = $state<{ key: string; trigger: HTMLElement } | null>(null),
		managing = $state<{
			accountId: number
			username: string
			serverId: number
			name: string
		} | null>(null)
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
		menuActions = $derived(activeMenu ? actionsFor(activeMenu.key) : [])

	onMount(() => {
		heading?.focus({ preventScroll: true })
		return () => {
			robloxServers.dispose()
			recordedServers.dispose()
			privateServers.dispose()
			listedServers.dispose()
			stats.dispose()
		}
	})

	$effect(() => {
		const current = active,
			accountId = accounts.selectedAccount.id
		untrack(() => {
			if (current === privateServers && accountId === 0) return
			if (!current.page && !current.loading) void current.load()
		})
	})

	$effect(() => {
		const accountId = accounts.selectedAccount.id
		untrack(() => {
			if (accountId && (privateServers.page || privateServers.error)) {
				privateServers.page = null
				privateServers.first()
			}
		})
	})

	$effect(() => {
		void active.page
		untrack(() => (activeMenu = null))
	})

	$effect(() => {
		void appSettings.roValraEnabled
		untrack(() => {
			if (robloxServers.page) void robloxServers.load()
		})
	})

	$effect(() => {
		if (appSettings.roValraEnabled) {
			untrack(() => {
				if (!stats.stats && !stats.loading) void stats.load()
			})
		}
	})

	function openMenu(key: string, trigger: HTMLElement, toggle: boolean): void {
		activeMenu = toggle && activeMenu?.trigger === trigger ? null : { key, trigger }
	}

	function select(action: () => void): () => void {
		return () => {
			activeMenu = null
			action()
		}
	}

	function linkActions(link: string): ServerMenuAction[] {
		return link
			? [
					{
						label: "Fill launch options",
						icon: listPlusIcon,
						onSelect: select(() => onFillLaunchLink(link)),
					},
					{
						label: "Copy launch link",
						icon: linkIcon,
						onSelect: select(() => void store.copy(link, "Launch link")),
					},
				]
			: []
	}

	function actionsFor(key: string): ServerMenuAction[] {
		if (source === "rbxservers") {
			const server = listedServers.page?.servers?.find((item) => item.id === key)
			return server
				? [
						{
							label: "Open on RBXServers",
							icon: arrowSquareOutIcon,
							onSelect: select(() => void store.openLink(server.url)),
						},
						{
							label: "Copy RBXServers link",
							icon: linkIcon,
							onSelect: select(
								() => void store.copy(server.url, "RBXServers link"),
							),
						},
					]
				: []
		}
		if (source === "private") {
			const server = privateServers.page?.servers?.find(
					(item) => String(item.serverId) === key,
				),
				account = accounts.selectedAccount
			if (!server) return []
			return [
				...linkActions(server.launchLink),
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
											serverId: server.serverId,
											name: server.name,
										}),
								),
							},
						]
					: []),
			]
		}
		return [
			{
				label: "Fill launch options",
				icon: listPlusIcon,
				onSelect: select(() => onFillLaunch(place.placeId, key)),
			},
			{
				label: "Copy job ID",
				icon: hashIcon,
				onSelect: select(() => void store.copy(key, "Job ID")),
			},
		]
	}

	function refresh(): void {
		if (active === privateServers && !accounts.selectedAccount.id) return
		active.refresh()
		if (appSettings.roValraEnabled) void stats.load()
	}
</script>

<section class="settings-page min-w-0" aria-labelledby="game-servers-title">
	<header class="settings-header">
		<button
			class="btn-ghost"
			type="button"
			aria-label="Back to game details"
			data-tooltip="Back to game details"
			onclick={onBack}>
			<Icon svg={caretLeftIcon} data-nudge="left" size={16} aria-hidden="true" />
		</button>
		<h1 id="game-servers-title" tabindex="-1" bind:this={heading}>Servers</h1>
		<span class="mr-auto min-w-0 text-xs wrap-anywhere text-fg-muted"
			>{store.displayName(place)}</span>
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
			class="btn-ghost"
			type="button"
			aria-label={active.loading ? "Refreshing servers" : "Refresh servers"}
			data-tooltip="Refresh servers"
			data-tooltip-side="bottom-end"
			aria-busy={active.loading}
			disabled={active.loading}
			onclick={refresh}>
			{#if active.loading}
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
	</header>

	{#if source === "rovalra"}
		<RecordedServerList
			servers={recordedServers}
			{stats}
			activeJobId={activeMenu?.key}
			{menuID}
			onMenu={openMenu} />
	{:else if source === "private"}
		<PrivateServerList
			servers={privateServers}
			{accounts}
			activeKey={activeMenu?.key}
			{menuID}
			onMenu={openMenu}
			onAccount={(accountId) => accounts.selectAccount(accountId)} />
	{:else if source === "rbxservers"}
		<RBXServersList
			servers={listedServers}
			activeKey={activeMenu?.key}
			{menuID}
			onMenu={openMenu} />
	{:else}
		<RobloxServerList
			servers={robloxServers}
			records={appSettings.roValraEnabled}
			newestVersion={stats.stats?.newestVersion ?? 0}
			activeJobId={activeMenu?.key}
			{menuID}
			onMenu={openMenu} />
	{/if}
</section>

{#if activeMenu && menuActions.length}
	<GameServerMenu
		id={menuID}
		anchor={activeMenu.trigger}
		actions={menuActions}
		onClose={() => (activeMenu = null)} />
{/if}

{#if managing}
	<PrivateServerDialog
		accountId={managing.accountId}
		username={managing.username}
		serverId={managing.serverId}
		name={managing.name}
		{notifications}
		onChanged={() => privateServers.refresh()}
		onClose={() => (managing = null)} />
{/if}

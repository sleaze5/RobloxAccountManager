<script lang="ts">
	import caretLeftIcon from "@phosphor-icons/core/regular/caret-left.svg?raw"
	import gameControllerIcon from "@phosphor-icons/core/regular/game-controller.svg?raw"
	import infoIcon from "@phosphor-icons/core/regular/info.svg?raw"
	import keyIcon from "@phosphor-icons/core/regular/key.svg?raw"
	import magnifyingGlassIcon from "@phosphor-icons/core/regular/magnifying-glass.svg?raw"
	import monitorIcon from "@phosphor-icons/core/regular/monitor.svg?raw"
	import plugIcon from "@phosphor-icons/core/regular/plug.svg?raw"
	import slidersHorizontalIcon from "@phosphor-icons/core/regular/sliders-horizontal.svg?raw"
	import xIcon from "@phosphor-icons/core/regular/x.svg?raw"
	import Icon from "../shared/Icon.svelte"
	import type { AccountStore } from "../accounts/account-store.svelte"
	import type { BrowserStore } from "../browser/browser-store.svelte"
	import type { NotificationCenter } from "../notifications/notification-center.svelte"
	import AboutSettings from "./AboutSettings.svelte"
	import BrowserSettings from "./BrowserSettings.svelte"
	import GeneralSettings from "./GeneralSettings.svelte"
	import IntegrationsSettings from "./IntegrationsSettings.svelte"
	import RobloxSettings from "./RobloxSettings.svelte"
	import { appSettings } from "./settings-store.svelte"
	import StorageSettings from "./StorageSettings.svelte"
	import TransferSettings from "./TransferSettings.svelte"
	import UpdateSettings from "./UpdateSettings.svelte"
	import UserInterfaceSettings from "./UserInterfaceSettings.svelte"
	import VaultSettings from "./VaultSettings.svelte"
	import SidebarResizer from "../layout/SidebarResizer.svelte"
	import type { WorkspaceState } from "../layout/workspace-state.svelte"

	const categories = [
		{
			icon: keyIcon,
			id: "vault",
			label: "Vault",
			search: "vault security vault status unlocked automatic unlock without master password when the app starts enable disable on off operating system user account lock vault test master password last tested not tested change master password rotate encryption key update password test reminder choose how often asks confirm storage location data folder path copy portable standard move data backups settings restart export import encrypted file password hint merge replace favorite places",
		},
		{
			icon: slidersHorizontalIcon,
			id: "general",
			label: "General",
			search: "general presence updates automatic enabled disabled interval seconds minutes online offline in game accounts bar all stored accounts batches active profile selected account while its profile is open logging all log levels enable disable every level trace detailed requests internal operations debug diagnostic information development info normal application activity completed operations warning recoverable problems need attention error failed operations critical problems",
		},
		{
			icon: monitorIcon,
			id: "user-interface",
			label: "User interface",
			search: "user interface motion animations animation reduce reduced full system operating system effects timestamps timestamp date time display format hover tooltip relative preview tokens token reference dollar reset defaults save",
		},
		{
			icon: gameControllerIcon,
			id: "roblox",
			label: "Roblox",
			search: "roblox multi instance multi-instance launch multiple games same time windows processes close ready status enabled disabled account manager open linux client sober mocktail automatic desktop default join",
		},
		{
			icon: plugIcon,
			id: "integrations",
			label: "Integrations",
			search: "integrations rovalra api server browser servers information features enabled disabled on off browser chrome testing runtime version download redownload remove disk usage installed invalid manage",
		},
		{
			icon: infoIcon,
			id: "about",
			label: "About",
			search: "about updates update check latest download install restart new release version build time timestamp unix platform architecture arch os go wails revision commit launch id startup stages timings duration ready storage data folder loaded components schema format version settings vault database sqlite encryption key file automatic unlock browser runtime manifest log created loaded recovered",
		},
	] as const

	type Category = (typeof categories)[number]

	let {
			store,
			browser,
			notifications,
			workspace,
			backLabel,
			onBack,
			onLock,
		}: {
			store: AccountStore
			browser: BrowserStore
			notifications: NotificationCenter
			workspace: WorkspaceState
			backLabel: string
			onBack: () => void
			onLock: () => void
		} = $props(),
		activeCategory = $state<Category>(categories[0]),
		query = $state(""),
		contentElement = $state<HTMLElement | undefined>(undefined),
		normalizedQuery = $derived(query.trim().toLowerCase()),
		searchTerms = $derived(normalizedQuery.split(/\s+/).filter(Boolean)),
		visibleCategories = $derived(
			categories.filter((category) =>
				searchTerms.every((term) => category.search.includes(term)),
			),
		)

	$effect(() => {
		if (
			visibleCategories.length > 0 &&
			!visibleCategories.some((category) => category.id === activeCategory.id)
		) {
			activeCategory = visibleCategories[0]
		}
	})

	$effect(() => {
		void activeCategory.id
		contentElement?.scrollTo({ top: 0 })
	})
</script>

<section class="settings-page" aria-labelledby="settings-title">
	<header class="settings-header">
		<button
			class="btn-ghost"
			type="button"
			aria-label={backLabel}
			data-tooltip={backLabel}
			onclick={onBack}>
			<Icon svg={caretLeftIcon} data-nudge="left" size={16} aria-hidden="true" />
		</button>
		<h1 id="settings-title">Settings</h1>
	</header>

	<div
		class="settings-layout"
		style={`--sidebar-width: ${workspace.settingsSidebarWidth}px;`}>
		<aside class="settings-sidebar">
			<div class="m-3 search-field">
				<Icon svg={magnifyingGlassIcon} size={16} aria-hidden="true" />
				<input
					bind:value={query}
					type="search"
					aria-label="Search settings"
					placeholder="Search settings" />
				{#if query}
					<button
						type="button"
						aria-label="Clear settings search"
						onclick={() => (query = "")}>
						<Icon svg={xIcon} size={14} aria-hidden="true" />
					</button>
				{/if}
			</div>
			<nav class="settings-nav" aria-label="Settings categories">
				{#each visibleCategories as category (category.id)}
					<button
						class="nav-item"
						type="button"
						aria-current={activeCategory.id === category.id
							? "page"
							: undefined}
						onclick={() => (activeCategory = category)}>
						<Icon svg={category.icon} size={16} aria-hidden="true" />
						<span>{category.label}</span>
					</button>
				{/each}
			</nav>
			{#if visibleCategories.length === 0}<p class="px-3 text-xs text-fg-faint">
					No settings found.
				</p>{/if}
		</aside>
		<SidebarResizer bind:width={workspace.settingsSidebarWidth} name="settings" />

		<main
			bind:this={contentElement}
			class="settings-content"
			aria-label="Settings content">
			{#key activeCategory.id}
				{#if visibleCategories.length > 0}
					<h2>{activeCategory.label}</h2>
					{#if activeCategory.id === "vault"}
						<VaultSettings {store} {query} {onLock} />
						<StorageSettings {store} {notifications} {query} />
						<TransferSettings {store} {notifications} {query} />
					{:else if activeCategory.id === "roblox"}
						<RobloxSettings />
					{:else if activeCategory.id === "integrations"}
						<BrowserSettings {browser} {query} />
						<IntegrationsSettings store={appSettings} {query} />
					{:else if activeCategory.id === "about"}
						<UpdateSettings />
						<AboutSettings />
					{:else if activeCategory.id === "user-interface"}
						<UserInterfaceSettings store={appSettings} {query} />
					{:else}
						<GeneralSettings store={appSettings} {query} />
					{/if}
				{:else}
					<p class="m-auto text-xs text-fg-faint">
						No settings match “{query}”.
					</p>
				{/if}
			{/key}
		</main>
	</div>
</section>

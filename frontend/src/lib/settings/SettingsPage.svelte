<script lang="ts">
	import ChevronLeft from "@lucide/svelte/icons/chevron-left"
	import Gamepad2 from "@lucide/svelte/icons/gamepad-2"
	import Info from "@lucide/svelte/icons/info"
	import KeyRound from "@lucide/svelte/icons/key-round"
	import MonitorCog from "@lucide/svelte/icons/monitor-cog"
	import Plug from "@lucide/svelte/icons/plug"
	import Search from "@lucide/svelte/icons/search"
	import SlidersHorizontal from "@lucide/svelte/icons/sliders-horizontal"
	import X from "@lucide/svelte/icons/x"
	import type { AccountStore } from "../accounts/account-store.svelte"
	import type { BrowserStore } from "../browser/browser-store.svelte"
	import AboutSettings from "./AboutSettings.svelte"
	import BrowserSettings from "./BrowserSettings.svelte"
	import GeneralSettings from "./GeneralSettings.svelte"
	import IntegrationsSettings from "./IntegrationsSettings.svelte"
	import RobloxSettings from "./RobloxSettings.svelte"
	import { appSettings } from "./settings-store.svelte"
	import UpdateSettings from "./UpdateSettings.svelte"
	import UserInterfaceSettings from "./UserInterfaceSettings.svelte"
	import VaultSettings from "./VaultSettings.svelte"

	const categories = [
		{
			icon: KeyRound,
			id: "vault",
			label: "Vault",
			search: "vault security vault status unlocked automatic unlock without master password when the app starts enable disable on off operating system user account lock vault test master password last tested not tested change master password rotate encryption key update password test reminder choose how often asks confirm",
		},
		{
			icon: SlidersHorizontal,
			id: "general",
			label: "General",
			search: "general presence updates automatic enabled disabled interval seconds minutes online offline in game accounts bar all stored accounts batches active profile selected account while its profile is open logging all log levels enable disable every level trace detailed requests internal operations debug diagnostic information development info normal application activity completed operations warning recoverable problems need attention error failed operations critical problems",
		},
		{
			icon: MonitorCog,
			id: "user-interface",
			label: "User Interface",
			search: "user interface motion animations animation reduce reduced full system operating system effects timestamps timestamp date time display format hover tooltip relative preview moment reset defaults save",
		},
		{
			icon: Gamepad2,
			id: "roblox",
			label: "Roblox",
			search: "roblox multi instance multi-instance launch multiple games same time windows processes close ready status enabled disabled account manager open",
		},
		{
			icon: Plug,
			id: "integrations",
			label: "Integrations",
			search: "integrations rovalra api server browser servers information features enabled disabled on off browser chrome testing runtime version download redownload remove disk usage installed invalid manage",
		},
		{
			icon: Info,
			id: "about",
			label: "About",
			search: "about updates update check latest download install restart new release version build time timestamp unix platform architecture arch os go wails revision commit launch id startup stages timings duration ready loaded components schema format version settings vault database sqlite sqlcipher key file automatic unlock browser runtime manifest log created loaded recovered",
		},
	] as const

	type Category = (typeof categories)[number]

	let {
			store,
			browser,
			onBack,
			onLock,
		}: {
			store: AccountStore
			browser: BrowserStore
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
			type="button"
			aria-label="Back to accounts"
			data-tooltip="Back to accounts"
			onclick={onBack}>
			<ChevronLeft size={16} aria-hidden="true" />
		</button>
		<h1 id="settings-title">Settings</h1>
	</header>

	<div class="settings-layout">
		<aside class="settings-sidebar">
			<div class="search-field">
				<Search size={15} aria-hidden="true" />
				<input
					bind:value={query}
					type="search"
					aria-label="Search settings"
					placeholder="Search settings" />
				{#if query}
					<button
						class="search-clear"
						type="button"
						aria-label="Clear settings search"
						onclick={() => (query = "")}>
						<X size={12} aria-hidden="true" />
					</button>
				{/if}
			</div>
			<nav aria-label="Settings categories">
				{#each visibleCategories as category (category.id)}
					<button
						type="button"
						aria-current={activeCategory.id === category.id
							? "page"
							: undefined}
						onclick={() => (activeCategory = category)}>
						<category.icon size={15} aria-hidden="true" />
						<span>{category.label}</span>
					</button>
				{/each}
			</nav>
			{#if visibleCategories.length === 0}<p class="settings-search-empty">
					No settings found.
				</p>{/if}
		</aside>

		<main
			bind:this={contentElement}
			class="settings-content"
			aria-label="Settings content">
			{#key `${activeCategory.id}:${normalizedQuery}`}
				{#if visibleCategories.length > 0}
					<h2>{activeCategory.label}</h2>
					{#if activeCategory.id === "vault"}
						<VaultSettings {store} {query} {onLock} />
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
					<p class="settings-content-empty">No settings match “{query}”.</p>
				{/if}
			{/key}
		</main>
	</div>
</section>

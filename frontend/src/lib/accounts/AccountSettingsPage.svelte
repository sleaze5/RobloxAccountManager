<script lang="ts">
	import BadgeDollarSign from "@lucide/svelte/icons/badge-dollar-sign"
	import ChevronLeft from "@lucide/svelte/icons/chevron-left"
	import ChevronRight from "@lucide/svelte/icons/chevron-right"
	import Eye from "@lucide/svelte/icons/eye"
	import LoaderCircle from "@lucide/svelte/icons/loader-circle"
	import LockKeyhole from "@lucide/svelte/icons/lock-keyhole"
	import MessageCircle from "@lucide/svelte/icons/message-circle"
	import RefreshCw from "@lucide/svelte/icons/refresh-cw"
	import Search from "@lucide/svelte/icons/search"
	import ShieldCheck from "@lucide/svelte/icons/shield-check"
	import SlidersHorizontal from "@lucide/svelte/icons/sliders-horizontal"
	import UserRound from "@lucide/svelte/icons/user-round"
	import X from "@lucide/svelte/icons/x"
	import { onMount, tick, untrack } from "svelte"
	import type { WorkspaceState } from "../layout/workspace-state.svelte"
	import { AccountSettingKey as Key } from "../backend/bridge"
	import type { AccountSettingOption, AccountSettingView } from "../backend/bridge"
	import type { NotificationCenter } from "../notifications/notification-center.svelte"
	import type { AccountStore } from "./account-store.svelte"
	import AccountInfoPanel, { accountInfoFields } from "./AccountInfoPanel.svelte"
	import type { AccountInfoFieldId } from "./AccountInfoPanel.svelte"
	import { AccountInfoState } from "./account-info-state.svelte"
	import AccountSettingRow from "./AccountSettingRow.svelte"
	import type { SettingRadioOption } from "./AccountSettingRow.svelte"
	import { AccountSettingsState } from "./account-settings-state.svelte"

	interface FieldDef {
		key: Key
		label: string
		description: string
		type: "toggle" | "radio"
		section: string
	}

	interface SectionDef {
		id: string
		title: string
		fields: FieldDef[]
	}

	interface CategoryDef {
		id: string
		label: string
		icon: typeof ShieldCheck
		search: string
		sections: SectionDef[]
	}

	const privacyPlaceholders: SettingRadioOption[] = [
			{ label: "Everyone", value: "AllUsers" },
			{
				label: "Friends, followers & people I follow",
				value: "FriendsFollowingAndFollowers",
			},
			{ label: "Friends & people I follow", value: "FriendsAndFollowing" },
			{ label: "Friends", value: "Friends" },
			{ label: "No one", value: "NoOne" },
		],
		visibilityPlaceholders: SettingRadioOption[] = [
			...privacyPlaceholders.slice(0, -1),
			{ label: "Trusted friends", value: "TrustedFriends" },
			privacyPlaceholders[privacyPlaceholders.length - 1],
		],
		friendsPlaceholders: SettingRadioOption[] = [
			{ label: "All friends", value: "AllConnections" },
			{ label: "Trusted friends", value: "TrustedConnectionsOnly" },
			{ label: "No one", value: "NoOne" },
		]

	function radioPlaceholders(key: Key): SettingRadioOption[] {
		switch (key) {
			case Key.SettingContentMaturity:
				return [
					{ label: "Minimal", value: "AllAges" },
					{ label: "Mild", value: "NinePlus" },
					{ label: "Moderate", value: "ThirteenPlus" },
					{ label: "Restricted", value: "SeventeenPlus" },
				]
			case Key.SettingPartyChat:
			case Key.SettingPartyVoice:
				return friendsPlaceholders
			case Key.SettingOnlineStatus:
				return visibilityPlaceholders
			case Key.SettingExperienceJoins:
				return visibilityPlaceholders.map((option) => ({
					label: option.label,
					value:
						(
							{
								AllUsers: "All",
								FriendsFollowingAndFollowers: "Followers",
								FriendsAndFollowing: "Following",
							} as Record<string, string>
						)[option.value] ?? option.value,
				}))
			case Key.SettingTradeQuality:
				return ["None", "Low", "Medium", "High"].map((value) => ({
					label: value,
					value,
				}))
			default:
				return privacyPlaceholders
		}
	}

	const categories: CategoryDef[] = [
		{
			id: "content-maturity",
			label: "Content maturity",
			icon: ShieldCheck,
			search: "content maturity minimal mild moderate restricted sensitive issues",
			sections: [
				{
					id: "maturity-section",
					title: "Content maturity",
					fields: [
						{
							key: Key.SettingContentMaturity,
							label: "Content maturity",
							description:
								"Choose the level of mature content and experiences allowed on this account.",
							type: "radio",
							section: "Content maturity",
						},
						{
							key: Key.SettingSensitiveIssues,
							label: "Sensitive issues",
							description:
								"Allow experiences that depict sensitive themes and topics.",
							type: "toggle",
							section: "Content maturity",
						},
					],
				},
			],
		},
		{
			id: "communication",
			label: "Communication",
			icon: MessageCircle,
			search: "communication experience direct chat party friends voice data product improvements camera input gameplay coordination quick words",
			sections: [
				{
					id: "experience-chat-section",
					title: "Experience chat",
					fields: [
						{
							key: Key.SettingExperienceChat,
							label: "Experience chat",
							description: "Enable text chat within Roblox experiences.",
							type: "toggle",
							section: "Experience chat",
						},
						{
							key: Key.SettingDirectExperienceChat,
							label: "Direct chat",
							description:
								"Allow direct messaging with other players inside experiences.",
							type: "toggle",
							section: "Experience chat",
						},
					],
				},
				{
					id: "party-friends-section",
					title: "Chat and party with friends",
					fields: [
						{
							key: Key.SettingParty,
							label: "Chat and party with friends",
							description:
								"Join parties and connect with friends across Roblox.",
							type: "toggle",
							section: "Chat and party with friends",
						},
						{
							key: Key.SettingPartyChat,
							label: "Friends chat",
							description: "Choose who can chat with you in parties.",
							type: "radio",
							section: "Chat and party with friends",
						},
						{
							key: Key.SettingPartyVoice,
							label: "Voice chat with friends",
							description: "Choose who can speak with you in parties.",
							type: "radio",
							section: "Chat and party with friends",
						},
						{
							key: Key.SettingVoiceOptIn,
							label: "Voice chat",
							description:
								"Enable spatial voice chat in supported experiences.",
							type: "toggle",
							section: "Chat and party with friends",
						},
					],
				},
				{
					id: "voice-data-section",
					title: "Voice data usage",
					fields: [
						{
							key: Key.SettingVoiceDataUsage,
							label: "Voice data for product improvements",
							description:
								"Allow Roblox to use voice recordings to improve products and services.",
							type: "toggle",
							section: "Voice data usage",
						},
					],
				},
				{
					id: "camera-input-section",
					title: "Camera input",
					fields: [
						{
							key: Key.SettingCamera,
							label: "Camera input",
							description:
								"Use camera tracking to animate your avatar with facial expressions.",
							type: "toggle",
							section: "Camera input",
						},
					],
				},
				{
					id: "gameplay-coordination-section",
					title: "Gameplay coordination",
					fields: [
						{
							key: Key.SettingQuickWords,
							label: "Quick words",
							description:
								"Use suggested preset phrases for quick chat during gameplay.",
							type: "toggle",
							section: "Gameplay coordination",
						},
					],
				},
			],
		},
		{
			id: "visibility-private-servers",
			label: "Visibility & private servers",
			icon: Eye,
			search: "visibility private servers online status current game activity updates social links who can add friend suggestions contacts phone",
			sections: [
				{
					id: "visibility-section",
					title: "Visibility",
					fields: [
						{
							key: Key.SettingOnlineStatus,
							label: "Online status",
							description: "Choose who can see when you are online.",
							type: "radio",
							section: "Visibility",
						},
						{
							key: Key.SettingExperienceJoins,
							label: "Show current game",
							description:
								"Choose who can see the game you are playing and join you.",
							type: "radio",
							section: "Visibility",
						},
						{
							key: Key.SettingActivityUpdates,
							label: "Share activity updates",
							description:
								"Notify friends about your in-game activity and milestones.",
							type: "toggle",
							section: "Visibility",
						},
						{
							key: Key.SettingSocialNetworks,
							label: "Social links",
							description:
								"Choose who can view links to your social networks on your profile.",
							type: "radio",
							section: "Visibility",
						},
					],
				},
				{
					id: "private-servers-section",
					title: "Private servers",
					fields: [
						{
							key: Key.SettingPrivateServerAdditions,
							label: "Who can add me to private servers",
							description:
								"Choose who can add you to their private servers.",
							type: "radio",
							section: "Private servers",
						},
					],
				},
				{
					id: "friends-contacts-section",
					title: "Friends & contacts",
					fields: [
						{
							key: Key.SettingFriendSuggestions,
							label: "Friend suggestions",
							description:
								"Receive friend suggestions based on mutual connections.",
							type: "toggle",
							section: "Friends & contacts",
						},
						{
							key: Key.SettingContactPermission,
							label: "Contact syncing",
							description:
								"Allow Roblox to sync contacts from your devices.",
							type: "toggle",
							section: "Friends & contacts",
						},
						{
							key: Key.SettingPhoneDiscovery,
							label: "Phone number discovery",
							description:
								"Allow people who have your phone number to find your account.",
							type: "toggle",
							section: "Friends & contacts",
						},
					],
				},
			],
		},
		{
			id: "trading-inventory",
			label: "Trading & inventory",
			icon: BadgeDollarSign,
			search: "trading inventory visibility trade audience quality filter",
			sections: [
				{
					id: "inventory-trading-section",
					title: "Trading & inventory",
					fields: [
						{
							key: Key.SettingInventoryVisibility,
							label: "Inventory visibility",
							description: "Choose who can view items in your inventory.",
							type: "radio",
							section: "Trading & inventory",
						},
						{
							key: Key.SettingTradeAudience,
							label: "Trade audience",
							description: "Choose who can send you trade requests.",
							type: "radio",
							section: "Trading & inventory",
						},
						{
							key: Key.SettingTradeQuality,
							label: "Trade quality filter",
							description:
								"Filter incoming trade requests by item value.",
							type: "radio",
							section: "Trading & inventory",
						},
					],
				},
			],
		},
		{
			id: "ads-preferences",
			label: "Ads preferences",
			icon: SlidersHorizontal,
			search: "ads preferences personalize advertising data selling sharing",
			sections: [
				{
					id: "ads-section",
					title: "Ads preferences",
					fields: [
						{
							key: Key.SettingPersonalizedAdvertising,
							label: "Personalize your ads",
							description:
								"Receive personalized advertising based on your activity and interests.",
							type: "toggle",
							section: "Ads preferences",
						},
						{
							key: Key.SettingDataSharing,
							label: "Data selling and sharing",
							description:
								"Allow sharing personal information with advertising partners.",
							type: "toggle",
							section: "Ads preferences",
						},
					],
				},
			],
		},
	]

	const privacyLabel = "Privacy & content restrictions",
		pages = [
			{
				id: "info",
				label: "Account info",
				icon: UserRound,
				search: "account info",
			},
			{
				id: "privacy",
				label: privacyLabel,
				icon: LockKeyhole,
				search: "privacy content restrictions",
			},
		] as const

	type PageId = (typeof pages)[number]["id"]

	let {
			store,
			workspace,
			notifications,
			onBack,
		}: {
			store: AccountStore
			workspace: WorkspaceState
			notifications: NotificationCenter
			onBack: () => void
		} = $props(),
		restoringScroll = $state(true),
		heading = $state<HTMLHeadingElement | undefined>(undefined),
		contentElement = $state<HTMLElement | undefined>(undefined),
		settingsState = $state<AccountSettingsState | null>(null),
		infoState = $state<AccountInfoState | null>(null),
		normalizedQuery = $derived(workspace.accountSettingsQuery.trim().toLowerCase()),
		searchTerms = $derived(normalizedQuery.split(/\s+/).filter(Boolean)),
		visibleCategories = $derived(
			categories.filter(
				(cat) =>
					filterFields(
						cat,
						cat.sections.flatMap((sec) => sec.fields),
					).length > 0,
			),
		),
		visibleInfoFields = $derived(
			new Set<AccountInfoFieldId>(
				accountInfoFields
					.filter(
						(field) =>
							searchTerms.length === 0 ||
							matches(`${pages[0].search} ${field.search}`),
					)
					.map((field) => field.id),
			),
		),
		visiblePages = $derived(
			pages.filter((page) =>
				page.id === "info"
					? visibleInfoFields.size > 0
					: visibleCategories.length > 0,
			),
		),
		activePage = $derived(
			visiblePages.find((page) => page.id === workspace.accountSettingsPage) ??
				visiblePages[0],
		),
		activeCategory = $derived(
			visibleCategories.find(
				(category) => category.id === workspace.accountSettingsCategory,
			) ?? null,
		),
		sectionErrors = $derived(settingsState?.snapshot?.sectionErrors ?? []),
		busy = $derived(
			!settingsState || settingsState.loading || settingsState.saving !== null,
		),
		refreshing = $derived(
			activePage?.id === "info" ? (infoState?.loading ?? false) : busy,
		)

	$effect(() => {
		if (activePage && activePage.id !== workspace.accountSettingsPage) {
			workspace.accountSettingsPage = activePage.id
		}
		if (workspace.accountSettingsCategory && !activeCategory) {
			workspace.accountSettingsCategory = null
		}
	})

	$effect(() => {
		const page = activePage?.id,
			info = infoState,
			settings = settingsState
		untrack(() => {
			if (page === "info" && info && !info.snapshot) void info.refresh()
			if (page === "privacy" && settings && !settings.snapshot) {
				void settings.refresh()
			}
		})
	})

	$effect(() => {
		void activePage?.id
		void activeCategory?.id
		void normalizedQuery
		const element = contentElement,
			ready =
				activePage?.id === "info"
					? !!infoState?.snapshot
					: !!settingsState?.snapshot,
			scrollTop = untrack(() => workspace.accountSettingsScrollTop),
			anchor = untrack(() => workspace.accountSettingsScrollAnchor)
		let cancelled = false
		restoringScroll = true
		void tick().then(() => {
			if (cancelled || !element || !ready) return undefined
			element.scrollTop = scrollTop
			const row =
				anchor &&
				[
					...element.querySelectorAll<HTMLElement>("[data-account-setting]"),
				].find((item) => item.dataset.accountSetting === anchor.key)
			if (row && anchor) {
				element.scrollTop +=
					row.getBoundingClientRect().top -
					element.getBoundingClientRect().top -
					anchor.offset
			}
			restoringScroll = false
			return undefined
		})
		return () => {
			cancelled = true
		}
	})

	function matches(text: string): boolean {
		const lower = text.toLowerCase()
		return searchTerms.every((term) => lower.includes(term))
	}

	function selectPage(page: PageId): void {
		if (
			workspace.accountSettingsPage === page &&
			!workspace.accountSettingsCategory
		) {
			return
		}
		resetScroll()
		workspace.accountSettingsPage = page
		workspace.accountSettingsCategory = null
	}

	async function openCategory(category: CategoryDef): Promise<void> {
		resetScroll()
		workspace.accountSettingsCategory = category.id
		await tick()
		contentElement?.querySelector<HTMLElement>(".settings-subpage-back")?.focus()
	}

	async function closeCategory(): Promise<void> {
		const previous = workspace.accountSettingsCategory
		resetScroll()
		workspace.accountSettingsCategory = null
		await tick()
		contentElement
			?.querySelector<HTMLElement>(`[data-settings-subpage="${previous}"]`)
			?.focus()
	}

	function refresh(): void {
		if (activePage?.id === "info") void infoState?.refresh()
		else void settingsState?.refresh()
	}

	function resetScroll(): void {
		workspace.accountSettingsScrollTop = 0
		workspace.accountSettingsScrollAnchor = null
	}

	function rememberScroll(): void {
		if (restoringScroll || !contentElement) return
		workspace.accountSettingsScrollTop = contentElement.scrollTop
		const top = contentElement.getBoundingClientRect().top,
			row = [
				...contentElement.querySelectorAll<HTMLElement>(
					"[data-account-setting]",
				),
			].find((item) => item.getBoundingClientRect().bottom > top)
		workspace.accountSettingsScrollAnchor =
			row?.dataset.accountSetting && contentElement.scrollTop > 0
				? {
						key: row.dataset.accountSetting,
						offset: row.getBoundingClientRect().top - top,
					}
				: null
	}

	function getSettingView(key: Key): AccountSettingView | undefined {
		return settingsState?.snapshot?.settings?.find((s) => s.key === key)
	}

	function computeRadioOptions(field: FieldDef): SettingRadioOption[] {
		const view = getSettingView(field.key)
		if (!view?.options?.length) {
			return radioPlaceholders(field.key).map((option) => ({
				label: option.label,
				value: option.value,
				disabled: true,
				reason:
					view?.unavailableReason ||
					"This setting is unavailable for this account.",
			}))
		}
		return (view?.options ?? []).flatMap((option) =>
			option.stringValue == null
				? []
				: [
						{
							value: option.stringValue,
							label: option.label,
							disabled: !view?.editable || !option.enabled,
							reason: option.reason || view?.unavailableReason || "",
						},
					],
		)
	}

	function computeFieldDisabled(field: FieldDef): boolean {
		const view = getSettingView(field.key)
		if (busy || settingsState?.refreshRequired || !view?.editable) return true
		return (
			field.type === "toggle" &&
			!getToggleOption(field.key, !isToggleChecked(view))?.enabled
		)
	}

	function isToggleChecked(view: AccountSettingView): boolean {
		return view.boolValue ?? view.stringValue === getToggleOnString(view.key)
	}

	function getToggleOption(
		key: Key,
		checked: boolean,
	): AccountSettingOption | undefined {
		const view = getSettingView(key)
		if (view?.boolValue != null) {
			return view.options?.find((option) => option.boolValue === checked)
		}
		const value = checked ? getToggleOnString(key) : getToggleOffString(key)
		return view?.options?.find((option) => option.stringValue === value)
	}
	function getToggleOnString(key: Key): string {
		switch (key) {
			case Key.SettingExperienceChat:
			case Key.SettingDirectExperienceChat:
				return "AllUsers"
			case Key.SettingParty:
				return "AllConnections"
			case Key.SettingActivityUpdates:
				return "Yes"
			case Key.SettingPhoneDiscovery:
				return "Discoverable"
			default:
				return "Enabled"
		}
	}

	function getToggleOffString(key: Key): string {
		switch (key) {
			case Key.SettingExperienceChat:
			case Key.SettingDirectExperienceChat:
			case Key.SettingParty:
				return "NoOne"
			case Key.SettingActivityUpdates:
				return "No"
			case Key.SettingPhoneDiscovery:
				return "NotDiscoverable"
			default:
				return "Disabled"
		}
	}

	async function handleToggle(key: Key, checked: boolean): Promise<void> {
		const view = getSettingView(key),
			option = getToggleOption(key, checked)
		if (!view?.editable || !option?.enabled || !settingsState) return
		await settingsState.save(view, option)
	}

	async function handleRadioSelect(key: Key, value: string): Promise<void> {
		const view = getSettingView(key),
			option = view?.options?.find((item) => item.stringValue === value)
		if (!view?.editable || !option?.enabled || !settingsState) return
		await settingsState.save(view, option)
	}

	function filterFields(category: CategoryDef, fields: FieldDef[]): FieldDef[] {
		if (
			searchTerms.length === 0 ||
			matches(`${privacyLabel} ${category.label} ${category.search}`)
		) {
			return fields
		}
		return fields.filter((f) => matches(`${f.label} ${f.description} ${f.section}`))
	}

	onMount(() => {
		const settings = new AccountSettingsState(
			store.selectedAccount.id,
			store,
			notifications,
		)
		const info = new AccountInfoState(store.selectedAccount.id, store)
		settingsState = settings
		infoState = info
		heading?.closest(".workspace-scroll")?.scrollTo({ top: 0 })
		heading?.focus({ preventScroll: true })
		return () => {
			settings.dispose()
			info.dispose()
		}
	})
</script>

<section class="settings-page" aria-labelledby="account-settings-title">
	<header class="settings-header">
		<button
			type="button"
			aria-label="Back to profile"
			data-tooltip="Back to profile"
			onclick={onBack}>
			<ChevronLeft size={16} aria-hidden="true" />
		</button>
		<h1 id="account-settings-title" tabindex="-1" bind:this={heading}>
			Account settings
		</h1>
		<span class="account-settings-identity">@{store.selectedAccount.username}</span>
		<button
			type="button"
			aria-label="Refresh settings"
			data-tooltip="Refresh settings"
			disabled={refreshing}
			onclick={refresh}>
			{#if refreshing}
				<LoaderCircle class="spinner" size={15} aria-hidden="true" />
			{:else}
				<RefreshCw size={15} aria-hidden="true" />
			{/if}
		</button>
	</header>

	<div class="settings-layout account-settings-layout">
		<aside class="settings-sidebar">
			<div class="search-field">
				<Search size={15} aria-hidden="true" />
				<input
					bind:value={workspace.accountSettingsQuery}
					oninput={resetScroll}
					type="search"
					aria-label="Search settings"
					placeholder="Search settings" />
				{#if workspace.accountSettingsQuery}
					<button
						class="search-clear"
						type="button"
						aria-label="Clear settings search"
						onclick={() => {
							resetScroll()
							workspace.accountSettingsQuery = ""
						}}>
						<X size={12} aria-hidden="true" />
					</button>
				{/if}
			</div>
			<nav aria-label="Account settings pages">
				{#each visiblePages as page (page.id)}
					<button
						type="button"
						aria-current={activePage?.id === page.id ? "page" : undefined}
						data-tooltip={page.label}
						data-tooltip-side="right"
						onclick={() => selectPage(page.id)}>
						<page.icon size={15} aria-hidden="true" />
						<span>{page.label}</span>
					</button>
				{/each}
			</nav>
			{#if visiblePages.length === 0}
				<p class="settings-search-empty">No settings found.</p>
			{/if}
		</aside>

		<main
			bind:this={contentElement}
			onscroll={rememberScroll}
			class="settings-content"
			aria-label="Account settings content">
			{#key `${activePage?.id}:${activeCategory?.id}:${normalizedQuery}`}
				{#if !activePage}
					<p class="settings-content-empty">
						No settings match “{workspace.accountSettingsQuery}”.
					</p>
				{:else if activePage.id === "info"}
					<h2>Account info</h2>
					{#if infoState}
						<AccountInfoPanel
							info={infoState}
							username={store.selectedAccount.username}
							visible={visibleInfoFields} />
					{/if}
				{:else}
					<h2>{privacyLabel}</h2>
					{#if settingsState?.error}
						<div class="settings-inline-error" role="alert">
							{settingsState.error}
						</div>
					{/if}
					{#each sectionErrors as error}
						<div class="settings-inline-error" role="alert">
							{error.message}
						</div>
					{/each}

					{#if !activeCategory}
						<nav class="settings-subpages" aria-label={privacyLabel}>
							{#each visibleCategories as cat (cat.id)}
								<button
									type="button"
									data-settings-subpage={cat.id}
									onclick={() => void openCategory(cat)}>
									<cat.icon size={15} aria-hidden="true" />
									<span>{cat.label}</span>
									<ChevronRight size={15} aria-hidden="true" />
								</button>
							{/each}
						</nav>
					{:else}
						<div class="settings-subpage-heading">
							<button
								class="icon-action settings-subpage-back"
								type="button"
								aria-label={`Back to ${privacyLabel}`}
								data-tooltip={`Back to ${privacyLabel}`}
								onclick={() => void closeCategory()}>
								<ChevronLeft size={16} aria-hidden="true" />
							</button>
							<h3>{activeCategory.label}</h3>
						</div>
						{#if !settingsState || (settingsState.loading && !settingsState.snapshot)}
							<div class="account-settings-loading" role="status">
								<LoaderCircle
									class="spinner"
									size={14}
									aria-hidden="true" />
								Loading settings
							</div>
						{:else}
							{#each activeCategory.sections as section (section.id)}
								{@const matchingFields = filterFields(
									activeCategory,
									section.fields,
								)}
								{#if matchingFields.length > 0}
									<section
										class="settings-section"
										aria-labelledby={section.title !==
										activeCategory.label
											? `section-title-${section.id}`
											: undefined}>
										{#if section.title !== activeCategory.label}
											<h3 id={`section-title-${section.id}`}>
												{section.title}
											</h3>
										{/if}
										<div class="settings-rows">
											{#each matchingFields as field (field.key)}
												{@const view = getSettingView(
													field.key,
												)}
												{@const fieldDisabled =
													computeFieldDisabled(field)}
												{@const fieldOptions =
													computeRadioOptions(field)}
												{@const isSaving =
													settingsState.saving === field.key}
												{@const fieldError =
													settingsState.feedback?.key ===
														field.key &&
													!settingsState.feedback.success
														? settingsState.feedback.message
														: ""}
												<AccountSettingRow
													id={field.key}
													label={field.label}
													description={field.description}
													type={field.type}
													{view}
													disabled={fieldDisabled}
													saving={isSaving}
													radioOptions={fieldOptions}
													error={fieldError}
													onToggle={(checked) =>
														void handleToggle(
															field.key,
															checked,
														)}
													onRadioSelect={(value) =>
														void handleRadioSelect(
															field.key,
															value,
														)} />
											{/each}
										</div>
									</section>
								{/if}
							{/each}
						{/if}
					{/if}
				{/if}
			{/key}
		</main>
	</div>
</section>

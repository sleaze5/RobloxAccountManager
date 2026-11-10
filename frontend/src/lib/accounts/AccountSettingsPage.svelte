<script lang="ts">
	import { matchesSearch, searchTerms } from "../shared/search"
	import circleNotchIcon from "@phosphor-icons/core/regular/circle-notch.svg?raw"
	import devicesIcon from "@phosphor-icons/core/regular/devices.svg?raw"
	import hardDrivesIcon from "@phosphor-icons/core/regular/hard-drives.svg?raw"
	import lockKeyIcon from "@phosphor-icons/core/regular/lock-key.svg?raw"
	import magnifyingGlassIcon from "@phosphor-icons/core/regular/magnifying-glass.svg?raw"
	import userCircleIcon from "@phosphor-icons/core/regular/user-circle.svg?raw"
	import xIcon from "@phosphor-icons/core/regular/x.svg?raw"
	import Icon from "../shared/Icon.svelte"
	import AccountPageHeader from "./AccountPageHeader.svelte"
	import RefreshIcon from "../shared/RefreshIcon.svelte"
	import { onMount, tick, untrack } from "svelte"
	import SidebarResizer from "../layout/SidebarResizer.svelte"
	import type { WorkspaceState } from "../layout/workspace-state.svelte"
	import { AccountSettingKey as Key } from "../backend/bridge"
	import type { AccountSettingOption, AccountSettingView } from "../backend/bridge"
	import AccountEditDialog from "../dialogs/AccountEditDialog.svelte"
	import SavedPasswordDialog from "../dialogs/SavedPasswordDialog.svelte"
	import SignOutSessionsDialog from "../dialogs/SignOutSessionsDialog.svelte"
	import type { AccountEditKind } from "../dialogs/AccountEditDialog.svelte"
	import type { NotificationCenter } from "../notifications/notification-center.svelte"
	import PrivateServerDialog from "../private-servers/PrivateServerDialog.svelte"
	import type { OwnedPrivateServer } from "../backend/bridge"
	import type { AccountStore } from "./account-store.svelte"
	import AccountInfoPanel, { accountInfoFields } from "./AccountInfoPanel.svelte"
	import AccountSessionsList from "./AccountSessionsList.svelte"
	import { AccountSessionsState } from "./account-sessions-state.svelte"
	import type { AccountInfoFieldId } from "./AccountInfoPanel.svelte"
	import { AccountInfoState } from "./account-info-state.svelte"
	import AccountSettingRow from "./AccountSettingRow.svelte"
	import type { SettingRadioOption } from "./AccountSettingRow.svelte"
	import { AccountSettingsState } from "./account-settings-state.svelte"
	import BlockedUsersList from "./BlockedUsersList.svelte"
	import { BlockedUsersState } from "./blocked-users-state.svelte"
	import OwnedPrivateServersList from "./OwnedPrivateServersList.svelte"
	import { OwnedPrivateServersState } from "./owned-private-servers-state.svelte"

	interface FieldDef {
		key: Key
		label: string
		description: string
		type: "toggle" | "radio"
	}

	interface SectionDef {
		id: string
		title: string
		fields: FieldDef[]
	}

	interface CategoryDef {
		id: string
		label: string
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
						},
						{
							key: Key.SettingSensitiveIssues,
							label: "Sensitive issues",
							description:
								"Allow experiences that depict sensitive themes and topics.",
							type: "toggle",
						},
					],
				},
			],
		},
		{
			id: "communication",
			label: "Communication",
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
						},
						{
							key: Key.SettingDirectExperienceChat,
							label: "Direct chat",
							description:
								"Allow direct messaging with other players inside experiences.",
							type: "toggle",
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
						},
						{
							key: Key.SettingPartyChat,
							label: "Friends chat",
							description: "Choose who can chat with you in parties.",
							type: "radio",
						},
						{
							key: Key.SettingPartyVoice,
							label: "Voice chat with friends",
							description: "Choose who can speak with you in parties.",
							type: "radio",
						},
						{
							key: Key.SettingVoiceOptIn,
							label: "Voice chat",
							description:
								"Enable spatial voice chat in supported experiences.",
							type: "toggle",
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
						},
					],
				},
			],
		},
		{
			id: "visibility-private-servers",
			label: "Visibility & private servers",
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
						},
						{
							key: Key.SettingExperienceJoins,
							label: "Show current game",
							description:
								"Choose who can see the game you are playing and join you.",
							type: "radio",
						},
						{
							key: Key.SettingActivityUpdates,
							label: "Share activity updates",
							description:
								"Notify friends about your in-game activity and milestones.",
							type: "toggle",
						},
						{
							key: Key.SettingSocialNetworks,
							label: "Social links",
							description:
								"Choose who can view links to your social networks on your profile.",
							type: "radio",
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
						},
						{
							key: Key.SettingContactPermission,
							label: "Contact syncing",
							description:
								"Allow Roblox to sync contacts from your devices.",
							type: "toggle",
						},
						{
							key: Key.SettingPhoneDiscovery,
							label: "Phone number discovery",
							description:
								"Allow people who have your phone number to find your account.",
							type: "toggle",
						},
					],
				},
			],
		},
		{
			id: "trading-inventory",
			label: "Trading & inventory",
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
						},
						{
							key: Key.SettingTradeAudience,
							label: "Trade audience",
							description: "Choose who can send you trade requests.",
							type: "radio",
						},
						{
							key: Key.SettingTradeQuality,
							label: "Trade quality filter",
							description:
								"Filter incoming trade requests by item value.",
							type: "radio",
						},
					],
				},
			],
		},
		{
			id: "ads-preferences",
			label: "Ads preferences",
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
						},
						{
							key: Key.SettingDataSharing,
							label: "Data selling and sharing",
							description:
								"Allow sharing personal information with advertising partners.",
							type: "toggle",
						},
					],
				},
			],
		},
	]

	const blockedUsersPage = {
			id: "blocked-users",
			label: "Blocked users",
			search: "blocked users block unblock",
		},
		privacyLabel = "Privacy & content restrictions",
		pages = [
			{
				id: "info",
				label: "Account info",
				icon: userCircleIcon,
				search: "account info",
			},
			{
				id: "privacy",
				label: "Privacy & content",
				icon: lockKeyIcon,
				search: "privacy content restrictions",
			},
			{
				id: "servers",
				label: "Private servers",
				icon: hardDrivesIcon,
				search: "private servers vip owned active renew subscription join link allowed users",
			},
			{
				id: "sessions",
				label: "Sessions",
				icon: devicesIcon,
				search: "sessions signed in devices browsers apps location ip address last active sign out log out security",
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
		contentElement = $state<HTMLElement | undefined>(undefined),
		settingsState = $state<AccountSettingsState | null>(null),
		infoState = $state<AccountInfoState | null>(null),
		blockedState = $state<BlockedUsersState | null>(null),
		ownedState = $state<OwnedPrivateServersState | null>(null),
		sessionsState = $state<AccountSessionsState | null>(null),
		signOutOpen = $state(false),
		managingServer = $state<OwnedPrivateServer | null>(null),
		editing = $state<AccountEditKind | null>(null),
		savedPasswordOpen = $state(false),
		normalizedQuery = $derived(workspace.accountSettingsQuery.trim().toLowerCase()),
		terms = $derived(searchTerms(normalizedQuery)),
		visibleCategories = $derived(
			categories.filter((category) =>
				category.sections.some(
					(section) => filterFields(category, section).length > 0,
				),
			),
		),
		visibleInfoFields = $derived(
			new Set<AccountInfoFieldId>(
				accountInfoFields
					.filter(
						(field) =>
							terms.length === 0 ||
							matches(`${pages[0].search} ${field.search}`),
					)
					.map((field) => field.id),
			),
		),
		showBlockedUsers = $derived(
			terms.length === 0 ||
				matches(
					`${privacyLabel} ${blockedUsersPage.label} ${blockedUsersPage.search}`,
				),
		),
		visibleSubpages = $derived([
			...visibleCategories.map((category) => ({
				id: category.id,
				label: category.label,
			})),
			...(showBlockedUsers ? [blockedUsersPage] : []),
		]),
		visiblePages = $derived(
			pages.filter((page) =>
				page.id === "info"
					? visibleInfoFields.size > 0
					: page.id === "privacy"
						? visibleSubpages.length > 0
						: terms.length === 0 || matches(`${page.label} ${page.search}`),
			),
		),
		activePage = $derived(
			visiblePages.find((page) => page.id === workspace.accountSettingsPage) ??
				visiblePages[0],
		),
		activeSubpage = $derived(
			activePage?.id === "privacy"
				? (visibleSubpages.find(
						(subpage) => subpage.id === workspace.accountSettingsCategory,
					) ?? visibleSubpages[0])
				: undefined,
		),
		activeCategory = $derived(
			visibleCategories.find((category) => category.id === activeSubpage?.id) ??
				null,
		),
		blockedUsersOpen = $derived(activeSubpage?.id === blockedUsersPage.id),
		sectionErrors = $derived(settingsState?.snapshot?.sectionErrors ?? []),
		busy = $derived(
			!settingsState || settingsState.loading || settingsState.saving !== null,
		),
		refreshing = $derived(
			activePage?.id === "info"
				? (infoState?.loading ?? false)
				: activePage?.id === "servers"
					? (ownedState?.loading ?? false)
					: activePage?.id === "sessions"
						? (sessionsState?.loading ?? false) ||
							(sessionsState?.signingOut ?? false)
						: blockedUsersOpen
							? (blockedState?.loading ?? false)
							: busy,
		)

	$effect(() => {
		if (activePage && activePage.id !== workspace.accountSettingsPage) {
			workspace.accountSettingsPage = activePage.id
		}
		if (activeSubpage && activeSubpage.id !== workspace.accountSettingsCategory) {
			workspace.accountSettingsCategory = activeSubpage.id
		}
	})

	$effect(() => {
		const page = activePage?.id,
			info = infoState,
			settings = settingsState,
			owned = ownedState,
			sessions = sessionsState,
			blocked = blockedUsersOpen ? blockedState : null
		untrack(() => {
			if (page === "info" && info && !info.snapshot) void info.refresh()
			if (page === "servers" && owned && !owned.servers) void owned.refresh()
			if (page === "sessions" && sessions && !sessions.sessions) {
				void sessions.refresh()
			}
			if (page === "privacy" && settings && !settings.snapshot) {
				void settings.refresh()
			}
			if (blocked && !blocked.users) void blocked.refresh()
		})
	})

	$effect(() => {
		void activePage?.id
		void activeSubpage?.id
		void normalizedQuery
		const element = contentElement,
			ready =
				activePage?.id === "info"
					? !!infoState?.snapshot
					: activePage?.id === "servers"
						? !!ownedState?.servers
						: activePage?.id === "sessions"
							? !!sessionsState?.sessions
							: blockedUsersOpen
								? !!blockedState?.users
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
				].find(
					(settingElement) =>
						settingElement.dataset.accountSetting === anchor.key,
				)
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
		return matchesSearch(terms, text)
	}

	function selectPage(page: PageId, subpage?: string): void {
		if (
			workspace.accountSettingsPage === page &&
			(!subpage || workspace.accountSettingsCategory === subpage)
		) {
			return
		}
		resetScroll()
		workspace.accountSettingsPage = page
		if (subpage) workspace.accountSettingsCategory = subpage
	}

	function refresh(): void {
		if (activePage?.id === "info") void infoState?.refresh()
		else if (activePage?.id === "servers") void ownedState?.refresh()
		else if (activePage?.id === "sessions") void sessionsState?.refresh()
		else if (blockedUsersOpen) void blockedState?.refresh()
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
			].find(
				(settingElement) => settingElement.getBoundingClientRect().bottom > top,
			)
		workspace.accountSettingsScrollAnchor =
			row?.dataset.accountSetting && contentElement.scrollTop > 0
				? {
						key: row.dataset.accountSetting,
						offset: row.getBoundingClientRect().top - top,
					}
				: null
	}

	function getSettingView(key: Key): AccountSettingView | undefined {
		return settingsState?.snapshot?.settings?.find((setting) => setting.key === key)
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
			option = view?.options?.find(
				(listedOption) => listedOption.stringValue === value,
			)
		if (!view?.editable || !option?.enabled || !settingsState) return
		await settingsState.save(view, option)
	}

	function filterFields(category: CategoryDef, section: SectionDef): FieldDef[] {
		if (
			terms.length === 0 ||
			matches(`${privacyLabel} ${category.label} ${category.search}`)
		) {
			return section.fields
		}
		return section.fields.filter((field) =>
			matches(`${field.label} ${field.description} ${section.title}`),
		)
	}

	onMount(() => {
		const settings = new AccountSettingsState(
			store.selectedAccount.id,
			store,
			notifications,
		)
		const info = new AccountInfoState(store.selectedAccount.id, store),
			blocked = new BlockedUsersState(store.selectedAccount.id, store),
			owned = new OwnedPrivateServersState(store.selectedAccount.id, store),
			sessions = new AccountSessionsState(
				store.selectedAccount.id,
				store,
				notifications,
			)
		settingsState = settings
		infoState = info
		blockedState = blocked
		ownedState = owned
		sessionsState = sessions
		return () => {
			settings.dispose()
			info.dispose()
			blocked.dispose()
			owned.dispose()
			sessions.dispose()
		}
	})
</script>

<section class="settings-page" aria-labelledby="account-settings-title">
	<AccountPageHeader
		titleId="account-settings-title"
		title="Account settings"
		username={store.selectedAccount.username}
		{onBack}>
		<button
			class="button-ghost"
			type="button"
			aria-label="Refresh settings"
			data-tooltip="Refresh settings"
			disabled={refreshing}
			onclick={refresh}>
			<RefreshIcon isRefreshing={refreshing} />
		</button>
	</AccountPageHeader>

	<div
		class="settings-layout"
		style={`--sidebar-width: ${workspace.accountSettingsSidebarWidth}px;`}>
		<aside class="settings-sidebar">
			<div class="m-3 search-field">
				<Icon svg={magnifyingGlassIcon} size={16} aria-hidden="true" />
				<input
					bind:value={workspace.accountSettingsQuery}
					oninput={resetScroll}
					type="search"
					aria-label="Search settings"
					placeholder="Search settings" />
				{#if workspace.accountSettingsQuery}
					<button
						type="button"
						aria-label="Clear settings search"
						onclick={() => {
							resetScroll()
							workspace.accountSettingsQuery = ""
						}}>
						<Icon svg={xIcon} size={14} aria-hidden="true" />
					</button>
				{/if}
			</div>
			<nav class="settings-nav" aria-label="Account settings pages">
				{#each visiblePages as page (page.id)}
					<button
						class="nav-item"
						type="button"
						aria-current={activePage?.id === page.id && !activeSubpage
							? "page"
							: undefined}
						data-current-branch={activePage?.id === page.id && activeSubpage
							? true
							: undefined}
						data-tooltip={page.label}
						data-tooltip-side="right"
						onclick={() => selectPage(page.id)}>
						<Icon svg={page.icon} size={16} aria-hidden="true" />
						<span>{page.label}</span>
					</button>
					{#if page.id === "privacy"}
						<div
							class="ml-4 flex flex-col gap-1 border-l border-line-subtle pl-1.5"
							role="group"
							aria-label={page.label}>
							{#each visibleSubpages as subpage (subpage.id)}
								<button
									class="nav-item"
									type="button"
									aria-current={activeSubpage?.id === subpage.id
										? "page"
										: undefined}
									data-tooltip={subpage.label}
									data-tooltip-side="right"
									onclick={() => selectPage("privacy", subpage.id)}>
									<span>{subpage.label}</span>
								</button>
							{/each}
						</div>
					{/if}
				{/each}
			</nav>
			{#if visiblePages.length === 0}
				<p class="px-3 text-xs text-fg-faint">No settings found.</p>
			{/if}
		</aside>
		<SidebarResizer
			bind:width={workspace.accountSettingsSidebarWidth}
			name="account settings"
			minimumWidth={192} />

		<main
			bind:this={contentElement}
			onscroll={rememberScroll}
			class="settings-content"
			aria-label="Account settings content">
			{#key `${activePage?.id}:${activeSubpage?.id}:${normalizedQuery}`}
				{#if !activePage}
					<p class="m-auto text-xs text-fg-faint">
						No settings match “{workspace.accountSettingsQuery}”.
					</p>
				{:else if activePage.id === "info"}
					<h2>Account info</h2>
					{#if infoState}
						<AccountInfoPanel
							info={infoState}
							account={store.selectedAccount}
							visible={visibleInfoFields}
							onSavedPassword={() => (savedPasswordOpen = true)}
							onEdit={(kind) => (editing = kind)} />
					{/if}
				{:else if activePage.id === "servers"}
					<h2>Private servers</h2>
					{#if ownedState}
						<OwnedPrivateServersList
							owned={ownedState}
							username={store.selectedAccount.username}
							onManage={(server) => (managingServer = server)} />
					{/if}
				{:else if activePage.id === "sessions"}
					<h2>Sessions</h2>
					{#if sessionsState}
						<AccountSessionsList
							sessions={sessionsState}
							username={store.selectedAccount.username}
							onSignOut={() => (signOutOpen = true)} />
					{/if}
				{:else}
					<h2>{activeSubpage?.label}</h2>
					{#if !blockedUsersOpen}
						{#if settingsState?.error}
							<div class="notice-error" role="alert">
								{settingsState.error}
							</div>
						{/if}
						{#each sectionErrors as error}
							<div class="notice-error" role="alert">
								{error.message}
							</div>
						{/each}
					{/if}
					{#if !activeCategory}
						{#if blockedState}
							<BlockedUsersList blocked={blockedState} />
						{/if}
					{:else if !settingsState || (settingsState.loading && !settingsState.snapshot)}
						<div
							class="flex items-center gap-2 text-xs text-fg-muted"
							role="status">
							<Icon
								svg={circleNotchIcon}
								class="spinner"
								size={16}
								aria-hidden="true" />
							Loading settings…
						</div>
					{:else}
						{#each activeCategory.sections as section (section.id)}
							{@const matchingFields = filterFields(
								activeCategory,
								section,
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
											{@const view = getSettingView(field.key)}
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
			{/key}
		</main>
	</div>
</section>

{#if editing}
	<AccountEditDialog
		account={store.selectedAccount}
		kind={editing}
		{notifications}
		onSaved={() => void infoState?.refresh()}
		onClose={() => (editing = null)} />
{/if}

{#if managingServer && store.vault.unlocked}
	<PrivateServerDialog
		accountId={store.selectedAccount.id}
		username={store.selectedAccount.username}
		serverId={managingServer.serverId}
		name={managingServer.name}
		gameName={managingServer.gameName}
		{notifications}
		onChanged={(settings) => ownedState?.update(settings)}
		onClose={() => (managingServer = null)} />
{/if}

{#if signOutOpen && sessionsState && store.vault.unlocked}
	<SignOutSessionsDialog
		account={store.selectedAccount}
		sessions={sessionsState}
		onClose={() => (signOutOpen = false)} />
{/if}

{#if savedPasswordOpen && store.vault.unlocked}
	{#key store.selectedAccount.id}
		<SavedPasswordDialog
			account={store.selectedAccount}
			{notifications}
			onClose={() => (savedPasswordOpen = false)} />
	{/key}
{/if}

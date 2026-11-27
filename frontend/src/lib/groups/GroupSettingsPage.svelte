<script lang="ts">
	import { notifyErrors } from "../notifications/error-notifications.svelte"
	import eyeIcon from "@phosphor-icons/core/regular/eye.svg?raw"
	import magnifyingGlassIcon from "@phosphor-icons/core/regular/magnifying-glass.svg?raw"
	import signInIcon from "@phosphor-icons/core/regular/sign-in.svg?raw"
	import usersThreeIcon from "@phosphor-icons/core/regular/users-three.svg?raw"
	import warningIcon from "@phosphor-icons/core/regular/warning.svg?raw"
	import xIcon from "@phosphor-icons/core/regular/x.svg?raw"
	import {
		GroupSlowMode,
		GroupSource,
		GroupVerificationLevel,
	} from "../backend/bridge"
	import type { GroupSettings, GroupSettingsChange } from "../backend/bridge"
	import SidebarResizer from "../layout/SidebarResizer.svelte"
	import { accountSubpageSidebarMinimumWidth } from "../layout/sidebar-widths"
	import { getNotificationCenter } from "../notifications/notification-center.svelte"
	import ConfirmationDialog from "../shared/ConfirmationDialog.svelte"
	import Icon from "../shared/Icon.svelte"
	import NoMatchingSettings from "../shared/NoMatchingSettings.svelte"
	import ProfilePageHeader from "../shared/ProfilePageHeader.svelte"
	import RefreshIcon from "../shared/RefreshIcon.svelte"
	import { matchesSearch, searchTerms } from "../shared/search"
	import SettingChangeList from "../shared/SettingChangeList.svelte"
	import SettingRow from "../shared/SettingRow.svelte"
	import type { SettingRadioOption } from "../shared/SettingRow.svelte"
	import type { GroupsStore } from "./groups-store.svelte"

	let { store }: { store: GroupsStore } = $props()

	type ToggleKey =
		| "isApprovalRequired"
		| "areFundsVisible"
		| "areGamesVisible"
		| "isMemberListVisible"
		| "areEnemiesAllowed"
	type RadioKey = "verificationLevel" | "minimumAccountAgeDays" | "slowMode"

	type FieldDef =
		| { key: ToggleKey; type: "toggle"; label: string; description: string }
		| {
				key: RadioKey
				type: "radio"
				label: string
				description: string
				options: SettingRadioOption[]
		  }

	interface PageDef {
		id: string
		label: string
		icon: string
		search: string
		fields: FieldDef[]
	}

	const onOff = (value: boolean) => (value ? "On" : "Off"),
		verificationOptions: SettingRadioOption[] = [
			{ value: GroupVerificationLevel.GroupVerificationNone, label: "None" },
			{
				value: GroupVerificationLevel.GroupVerificationLow,
				label: "Low: a verified phone, email, or ID",
			},
			{
				value: GroupVerificationLevel.GroupVerificationMedium,
				label: "Medium: a verified ID, or a verified phone and email",
			},
			{
				value: GroupVerificationLevel.GroupVerificationHigh,
				label: "High: a verified ID",
			},
		],
		accountAgeOptions: SettingRadioOption[] = [0, 1, 3, 7, 30, 90].map((days) => ({
			value: String(days),
			label: days === 0 ? "Any age" : `${days} ${days === 1 ? "day" : "days"}`,
		})),
		slowModeOptions: SettingRadioOption[] = [
			{ value: GroupSlowMode.GroupSlowModeOff, label: "Off" },
			{
				value: GroupSlowMode.GroupSlowModeSlow,
				label: "Slow: 1 post every 2 minutes, 2 comments a minute",
			},
			{
				value: GroupSlowMode.GroupSlowModeSlower,
				label: "Slower: 1 post every 5 minutes, 1 comment a minute",
			},
			{
				value: GroupSlowMode.GroupSlowModeSlowest,
				label: "Slowest: 1 post every 10 minutes, 1 comment every 2 minutes",
			},
		],
		pages: PageDef[] = [
			{
				id: "join",
				label: "Join requirements",
				icon: signInIcon,
				search: "join requests approval manual verification phone email id account age days entry",
				fields: [
					{
						key: "isApprovalRequired",
						type: "toggle",
						label: "Require join requests",
						description:
							"Users send a join request. A member with the permission accepts or declines it.",
					},
					{
						key: "verificationLevel",
						type: "radio",
						label: "Verification",
						description:
							"Select the verification that a user needs to join. Current members keep their place.",
						options: verificationOptions,
					},
					{
						key: "minimumAccountAgeDays",
						type: "radio",
						label: "Minimum account age",
						description:
							"Select how old an account must be to join. Current members keep their place.",
						options: accountAgeOptions,
					},
				],
			},
			{
				id: "visibility",
				label: "Visibility",
				icon: eyeIcon,
				search: "public show hide funds robux games members member list",
				fields: [
					{
						key: "areFundsVisible",
						type: "toggle",
						label: "Show funds",
						description: "Anyone can see the Robux of the group.",
					},
					{
						key: "areGamesVisible",
						type: "toggle",
						label: "Show games",
						description:
							"Anyone can see the games of the group on its page.",
					},
					{
						key: "isMemberListVisible",
						type: "toggle",
						label: "Show members",
						description: "Anyone can see the member list.",
					},
				],
			},
			{
				id: "community",
				label: "Community",
				icon: usersThreeIcon,
				search: "enemies allies relationships forum slow mode posts comments rate limit",
				fields: [
					{
						key: "areEnemiesAllowed",
						type: "toggle",
						label: "Allow enemies",
						description: "Other groups can declare this group an enemy.",
					},
					{
						key: "slowMode",
						type: "radio",
						label: "Forum slow mode",
						description:
							"Select how often members can post in the forums. Members with the Bypass Slow Mode permission have no limit.",
						options: slowModeOptions,
					},
				],
			},
		]

	interface PendingChange {
		label: string
		previous: string
		next: string
		change: GroupSettingsChange
	}

	const emptyChange: GroupSettingsChange = {
		isApprovalRequired: null,
		areEnemiesAllowed: null,
		areFundsVisible: null,
		areGamesVisible: null,
		isMemberListVisible: null,
		verificationLevel: null,
		minimumAccountAgeDays: null,
		slowMode: null,
	}

	const notifications = getNotificationCenter()

	let query = $state(""),
		selectedPageId = $state("join"),
		sidebarWidth = $state(accountSubpageSidebarMinimumWidth),
		pendingChange = $state<PendingChange | null>(null),
		searchInput = $state<HTMLInputElement>(),
		contentElement = $state<HTMLElement>()

	const profile = $derived(store.profile),
		group = $derived(profile?.details.group),
		settings = $derived<GroupSettings | null>(profile?.settings ?? null),
		areSettingsUnavailable = $derived(
			profile?.unavailable?.includes(GroupSource.GroupSourceSettings) ?? false,
		),
		terms = $derived(searchTerms(query)),
		visiblePages = $derived(
			pages.filter(
				(page) =>
					terms.length === 0 ||
					matchingFields(page).length > 0 ||
					matchesSearch(terms, `${page.label} ${page.search}`),
			),
		),
		activePage = $derived(
			visiblePages.find((page) => page.id === selectedPageId) ?? visiblePages[0],
		),
		isSaving = $derived(store.pendingAction === "settings")

	function matchingFields(page: PageDef): FieldDef[] {
		if (terms.length === 0 || matchesSearch(terms, `${page.label} ${page.search}`))
			return page.fields
		return page.fields.filter((field) =>
			matchesSearch(terms, `${field.label} ${field.description}`),
		)
	}

	function radioValue(field: FieldDef): string {
		if (!settings || field.type !== "radio") return ""
		const value = settings[field.key]
		return value === null || value === undefined ? "" : String(value)
	}

	function optionLabel(field: FieldDef, value: string): string {
		return field.type === "radio"
			? (field.options
					.find((option) => option.value === value)
					?.label.split(":")[0] ?? "Unknown")
			: value
	}

	function requestToggle(field: FieldDef, isOn: boolean): void {
		if (!settings || field.type !== "toggle") return
		pendingChange = {
			label: field.label,
			previous: onOff(!isOn),
			next: onOff(isOn),
			change: { ...emptyChange, [field.key]: isOn },
		}
	}

	function requestRadio(field: FieldDef, value: string): void {
		if (!settings || field.type !== "radio") return
		const changedValue =
			field.key === "minimumAccountAgeDays" ? Number(value) : value
		pendingChange = {
			label: field.label,
			previous: optionLabel(field, radioValue(field)),
			next: optionLabel(field, value),
			change: { ...emptyChange, [field.key]: changedValue },
		}
	}

	async function confirmChange(): Promise<void> {
		const pending = pendingChange
		if (!pending || !(await store.saveSettings(pending.change))) return
		pendingChange = null
		notifications.show({
			id: `group-settings-${store.selectedGroupId}`,
			title: "Group setting saved",
			tone: "success",
			message: `${pending.label}: ${pending.next}`,
		})
	}

	function selectPage(pageId: string): void {
		selectedPageId = pageId
		contentElement?.scrollTo({ top: 0 })
	}

	function clearSearch(): void {
		query = ""
		searchInput?.focus()
	}
</script>

{#if group && profile}
	<section class="settings-page min-w-0" aria-labelledby="group-settings-title">
		<ProfilePageHeader
			titleId="group-settings-title"
			title="Settings"
			subtitle={group.name}>
			<button
				class="button-ghost"
				type="button"
				aria-label={store.isLoadingProfile
					? "Refreshing settings…"
					: "Refresh settings"}
				data-tooltip="Refresh settings"
				data-tooltip-side="bottom-end"
				aria-busy={store.isLoadingProfile}
				disabled={store.isLoadingProfile}
				onclick={() => void store.refreshProfile()}>
				<RefreshIcon isRefreshing={store.isLoadingProfile} />
			</button>
		</ProfilePageHeader>

		<div class="settings-layout" style:--sidebar-width={`${sidebarWidth}px`}>
			<div class="settings-sidebar">
				<div class="m-3 search-field">
					<Icon svg={magnifyingGlassIcon} size={16} aria-hidden="true" />
					<input
						bind:value={query}
						bind:this={searchInput}
						type="search"
						aria-label="Search settings"
						placeholder="Search settings" />
					{#if query}
						<button
							type="button"
							aria-label="Clear search"
							onclick={clearSearch}>
							<Icon svg={xIcon} size={14} aria-hidden="true" />
						</button>
					{/if}
				</div>
				<nav class="settings-nav" aria-label="Group settings pages">
					{#each visiblePages as page (page.id)}
						<button
							class="nav-item"
							type="button"
							aria-current={activePage?.id === page.id
								? "page"
								: undefined}
							data-tooltip={page.label}
							data-tooltip-side="right"
							onclick={() => selectPage(page.id)}>
							<Icon svg={page.icon} size={16} aria-hidden="true" />
							<span>{page.label}</span>
						</button>
					{/each}
				</nav>
			</div>
			<SidebarResizer
				bind:width={sidebarWidth}
				name="group settings"
				minimumWidth={accountSubpageSidebarMinimumWidth} />

			<section
				bind:this={contentElement}
				class="settings-content"
				aria-label="Group settings content">
				{#key `${activePage?.id}:${query}`}
					{#if !settings && areSettingsUnavailable}
						<div class="empty-state h-full">
							<Icon svg={warningIcon} size={28} aria-hidden="true" />
							<h2>Group settings unavailable</h2>
							<button
								class="button"
								type="button"
								disabled={store.isLoadingProfile}
								onclick={() => void store.refreshProfile()}
								>Try again</button>
						</div>
					{:else if !activePage}
						<NoMatchingSettings onClearSearch={clearSearch} />
					{:else}
						<h2>{activePage.label}</h2>
						<section class="settings-section" aria-label={activePage.label}>
							<div class="settings-rows">
								{#each matchingFields(activePage) as field (field.key)}
									{@const isUnavailable =
										!!settings &&
										(settings[field.key] === null ||
											settings[field.key] === undefined)}
									<SettingRow
										id={`group-${field.key}`}
										label={field.label}
										description={field.description}
										type={field.type}
										isChecked={field.type === "toggle" &&
											settings?.[field.key] === true}
										radioValue={radioValue(field)}
										radioOptions={field.type === "radio"
											? field.options
											: []}
										notes={isUnavailable
											? [
													"Roblox does not share this setting for this group.",
												]
											: []}
										disabled={!settings ||
											isUnavailable ||
											isSaving}
										isLoading={!settings}
										saving={isSaving &&
											pendingChange?.label === field.label}
										onToggle={(isOn) => requestToggle(field, isOn)}
										onRadioSelect={(value) =>
											requestRadio(field, value)} />
								{/each}
							</div>
						</section>
					{/if}
				{/key}
			</section>
		</div>
	</section>
{/if}

{#if pendingChange && group}
	<ConfirmationDialog
		title="Save group setting on Roblox?"
		description={`Roblox applies this change to ${group.name}.`}
		confirmLabel="Save on Roblox"
		progressLabel="Saving…"
		tone="primary"
		account={store.account}
		isWorking={isSaving}
		onConfirm={() => void confirmChange()}
		onCancel={() => (pendingChange = null)}>
		<SettingChangeList
			changes={[
				{
					label: pendingChange.label,
					previous: pendingChange.previous,
					next: pendingChange.next,
				},
			]} />
	</ConfirmationDialog>
{/if}

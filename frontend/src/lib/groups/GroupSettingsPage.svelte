<script lang="ts">
	import { notifyErrors } from "../notifications/error-notifications.svelte"
	import eyeIcon from "@phosphor-icons/core/regular/eye.svg?raw"
	import magnifyingGlassIcon from "@phosphor-icons/core/regular/magnifying-glass.svg?raw"
	import signInIcon from "@phosphor-icons/core/regular/sign-in.svg?raw"
	import usersThreeIcon from "@phosphor-icons/core/regular/users-three.svg?raw"
	import warningIcon from "@phosphor-icons/core/regular/warning.svg?raw"
	import xIcon from "@phosphor-icons/core/regular/x.svg?raw"
	import {
		GroupMinimumAccountAge,
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
	type RadioKey = "verificationLevel" | "minimumAccountAge" | "slowMode"

	// SettingOption is an option of a setting. A confirmation names it by its
	// name alone, and the list shows its detail too.
	type SettingOption = SettingRadioOption & { name: string }

	type SettingField =
		| { key: ToggleKey; type: "toggle"; label: string; description: string }
		| {
				key: RadioKey
				type: "radio"
				label: string
				description: string
				options: SettingOption[]
		  }

	interface SettingCategory {
		id: string
		label: string
		icon: string
		search: string
		fields: SettingField[]
	}

	const onOff = (value: boolean) => (value ? "On" : "Off"),
		settingOption = (
			value: string | number,
			name: string,
			detail = "",
		): SettingOption => ({
			value: String(value),
			name,
			label: detail ? `${name}: ${detail}` : name,
		}),
		verificationOptions = [
			settingOption(GroupVerificationLevel.GroupVerificationNone, "None"),
			settingOption(
				GroupVerificationLevel.GroupVerificationLow,
				"Low",
				"a verified phone, email, or ID",
			),
			settingOption(
				GroupVerificationLevel.GroupVerificationMedium,
				"Medium",
				"a verified ID, or a verified phone and email",
			),
			settingOption(
				GroupVerificationLevel.GroupVerificationHigh,
				"High",
				"a verified ID",
			),
		],
		accountAgeOptions = [
			GroupMinimumAccountAge.GroupMinimumAccountAgeAny,
			GroupMinimumAccountAge.GroupMinimumAccountAgeOneDay,
			GroupMinimumAccountAge.GroupMinimumAccountAgeThreeDays,
			GroupMinimumAccountAge.GroupMinimumAccountAgeSevenDays,
			GroupMinimumAccountAge.GroupMinimumAccountAgeThirtyDays,
			GroupMinimumAccountAge.GroupMinimumAccountAgeNinetyDays,
		].map((days) =>
			settingOption(
				days,
				days === GroupMinimumAccountAge.GroupMinimumAccountAgeAny
					? "Any age"
					: `${days} ${days === 1 ? "day" : "days"}`,
			),
		),
		slowModeOptions = [
			settingOption(GroupSlowMode.GroupSlowModeOff, "Off"),
			settingOption(
				GroupSlowMode.GroupSlowModeSlow,
				"Slow",
				"1 post every 2 minutes, 2 comments a minute",
			),
			settingOption(
				GroupSlowMode.GroupSlowModeSlower,
				"Slower",
				"1 post every 5 minutes, 1 comment a minute",
			),
			settingOption(
				GroupSlowMode.GroupSlowModeSlowest,
				"Slowest",
				"1 post every 10 minutes, 1 comment every 2 minutes",
			),
		],
		categories: SettingCategory[] = [
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
						key: "minimumAccountAge",
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
		key: ToggleKey | RadioKey
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
		minimumAccountAge: null,
		slowMode: null,
	}

	const notifications = getNotificationCenter()

	let query = $state(""),
		selectedCategoryId = $state("join"),
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
		visibleCategories = $derived(
			categories.filter(
				(category) =>
					terms.length === 0 ||
					matchingFields(category).length > 0 ||
					matchesSearch(terms, `${category.label} ${category.search}`),
			),
		),
		activeCategory = $derived(
			visibleCategories.find((category) => category.id === selectedCategoryId) ??
				visibleCategories[0],
		),
		isSaving = $derived(store.pendingAction === "settings")

	function matchingFields(category: SettingCategory): SettingField[] {
		if (
			terms.length === 0 ||
			matchesSearch(terms, `${category.label} ${category.search}`)
		)
			return category.fields
		return category.fields.filter((field) =>
			matchesSearch(terms, `${field.label} ${field.description}`),
		)
	}

	function radioValue(field: SettingField): string {
		if (!settings || field.type !== "radio") return ""
		const value = settings[field.key]
		return value === null || value === undefined ? "" : String(value)
	}

	function optionName(field: SettingField, value: string): string {
		return field.type === "radio"
			? (field.options.find((option) => option.value === value)?.name ??
					"Unknown")
			: value
	}

	function requestToggle(field: SettingField, isOn: boolean): void {
		if (!settings || field.type !== "toggle") return
		pendingChange = {
			key: field.key,
			label: field.label,
			previous: onOff(!isOn),
			next: onOff(isOn),
			change: { ...emptyChange, [field.key]: isOn },
		}
	}

	function requestRadio(field: SettingField, value: string): void {
		if (!settings || field.type !== "radio") return
		const changedValue = field.key === "minimumAccountAge" ? Number(value) : value
		pendingChange = {
			key: field.key,
			label: field.label,
			previous: optionName(field, radioValue(field)),
			next: optionName(field, value),
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

	function selectCategory(categoryId: string): void {
		selectedCategoryId = categoryId
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
					{#each visibleCategories as category (category.id)}
						<button
							class="nav-item"
							type="button"
							aria-current={activeCategory?.id === category.id
								? "page"
								: undefined}
							data-tooltip={category.label}
							data-tooltip-side="right"
							onclick={() => selectCategory(category.id)}>
							<Icon svg={category.icon} size={16} aria-hidden="true" />
							<span>{category.label}</span>
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
				{#key `${activeCategory?.id}:${query}`}
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
					{:else if !activeCategory}
						<NoMatchingSettings onClearSearch={clearSearch} />
					{:else}
						<h2>{activeCategory.label}</h2>
						<section
							class="settings-section"
							aria-label={activeCategory.label}>
							<div class="settings-rows">
								{#each matchingFields(activeCategory) as field (field.key)}
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
											pendingChange?.key === field.key}
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

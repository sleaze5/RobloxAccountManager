<script lang="ts">
	import { notifyErrors } from "../notifications/error-notifications.svelte"
	import ProfileShowcase from "./ProfileShowcase.svelte"
	import dotsThreeIcon from "@phosphor-icons/core/regular/dots-three.svg?raw"
	import hashIcon from "@phosphor-icons/core/regular/hash.svg?raw"
	import sealCheckIcon from "@phosphor-icons/core/regular/seal-check.svg?raw"
	import starFillIcon from "@phosphor-icons/core/fill/star-fill.svg?raw"
	import tagIcon from "@phosphor-icons/core/regular/tag.svg?raw"
	import warningCircleIcon from "@phosphor-icons/core/regular/warning-circle.svg?raw"
	import Icon from "../shared/Icon.svelte"
	import RefreshIcon from "../shared/RefreshIcon.svelte"
	import Skeleton from "../shared/Skeleton.svelte"
	import { tick, untrack } from "svelte"
	import arrowCounterClockwiseIcon from "@phosphor-icons/core/regular/arrow-counter-clockwise.svg?raw"
	import circleNotchIcon from "@phosphor-icons/core/regular/circle-notch.svg?raw"
	import {
		AccountProfileSection,
		SessionState,
		SocialList,
		UserJoinability,
	} from "../backend/bridge"
	import type { GamePlace, RobloxNotificationUser } from "../backend/bridge"
	import AccountModerationPanel from "./AccountModerationPanel.svelte"
	import { banEndsAtMs, canReactivate, hasReactivation } from "./moderation.svelte"
	import { CountdownClock } from "../shared/countdown-clock.svelte"
	import type { SocialListKind } from "../social/social-store.svelte"
	import AvatarPage from "../avatar/AvatarPage.svelte"
	import ChatsPage from "../chats/ChatsPage.svelte"
	import InventoryPage from "../inventory/InventoryPage.svelte"
	import RobloxNotificationsPage from "../roblox-notifications/RobloxNotificationsPage.svelte"
	import FriendConnections from "../social/FriendConnections.svelte"
	import { relationshipStats } from "../social/relationship-stats"
	import ProfileStats from "../shared/ProfileStats.svelte"
	import { getPageVisibility } from "../layout/page-visibility"
	import type { WorkspaceState } from "../layout/workspace-state.svelte"
	import type { NotificationCenter } from "../notifications/notification-center.svelte"
	import { appSettings } from "../settings/settings-store.svelte"
	import { floatMenu, menuIn, menuKeyboard, menuOut } from "../shared/transitions"
	import ImageMenuTarget from "../layout/ImageMenuTarget.svelte"
	import AvatarPortraitFrame from "../avatar/AvatarPortraitFrame.svelte"
	import { AvatarPortraitLoad } from "../avatar/avatar-portrait.svelte"
	import ProfileTabs from "../shared/ProfileTabs.svelte"
	import OnlineStatusJoinButton from "./OnlineStatusJoinButton.svelte"
	import { onlineStatusJoinFor } from "./online-status-join"
	import { setProfilePages } from "../shared/profile-pages"
	import { accountPageLabels, accountPages } from "./account-pages"
	import type { AccountPage } from "./account-pages"
	import cookieIcon from "@phosphor-icons/core/regular/cookie.svg?raw"
	import AccountAvatar from "./AccountAvatar.svelte"
	import AccountMembershipBadges from "./AccountMembershipBadges.svelte"
	import AccountOnlineStatus from "./AccountOnlineStatus.svelte"
	import AccountProfileBalance from "./AccountProfileBalance.svelte"
	import AccountProfileDescription from "./AccountProfileDescription.svelte"
	import AccountProfileDetails from "./AccountProfileDetails.svelte"
	import AccountSettingsPage from "./AccountSettingsPage.svelte"
	import type { AccountStore } from "./account-store.svelte"
	import { getCustomTags, sessionAttentionLabel } from "./account-model"
	import TagPicker from "./TagPicker.svelte"

	let {
		store,
		workspace,
		onManageTags,
		onJoinServer,
		onFollowUser,
		onReplaceCookie,
		onEditDescription,
		onShowSocial,
		onShowCreations,
		onOpenBrowser,
		notifications,
	}: {
		store: AccountStore
		workspace: WorkspaceState
		onManageTags: (accountId: number) => void
		onJoinServer: (placeId: number, serverId: string) => void
		onFollowUser: (user: RobloxNotificationUser) => void
		onReplaceCookie: () => void
		onEditDescription: (description: string) => void
		onShowSocial: (list: SocialListKind) => void
		onShowCreations: (username: string) => void
		onOpenBrowser: (accountId: number) => void
		notifications: NotificationCenter
	} = $props()

	const pageVisibility = getPageVisibility()
	let showcaseRefreshCount = $state(0),
		profileSection = $state<HTMLElement>()
	const avatarPortrait = new AvatarPortraitLoad()
	const selectedAccount = $derived(store.selectedAccount),
		selectedCustomTags = $derived(getCustomTags(selectedAccount)),
		profileAccountId = $derived(selectedAccount.id),
		hasAccount = $derived(profileAccountId !== 0),
		robloxUserId = $derived(Number(selectedAccount.userId)),
		moderated = $derived(selectedAccount.state === SessionState.StateModerated),
		sessionAttention = $derived(sessionAttentionLabel(selectedAccount.state)),
		profileVisible = $derived(
			store.vault.unlocked &&
				hasAccount &&
				!moderated &&
				workspace.accountPage === "profile",
		),
		moderationClock = new CountdownClock(),
		canBeReactivated = $derived(
			moderated && hasReactivation(selectedAccount.moderation),
		),
		reactivatable = $derived(
			canReactivate(selectedAccount.moderation, moderationClock.nowMs),
		),
		reactivating = $derived(store.reactivatingAccountId === selectedAccount.id),
		checkingModeration = $derived(store.checkingAccountId === selectedAccount.id),
		// Join is the main action only when nothing about the account needs
		// attention first.
		canJoinFromHeader = $derived(
			hasAccount && store.vault.unlocked && !moderated && !sessionAttention,
		),
		hasJoinAction = $derived(
			canJoinFromHeader &&
				onlineStatusJoinFor(
					selectedAccount.onlineStatus,
					UserJoinability.UserJoinabilityUnknown,
					false,
				).hasVisibleServer,
		),
		pageUnavailableReason = $derived(
			!store.vault.unlocked
				? "Unlock the vault first"
				: moderated
					? "Unavailable while moderated"
					: "",
		)

	let tagsButton = $state<HTMLButtonElement | undefined>(undefined)

	// A refresh of the account data also reads the portrait again.
	$effect(() => {
		void showcaseRefreshCount
		return avatarPortrait.load(profileVisible ? profileAccountId : 0, robloxUserId)
	})
	const profileState = $derived(
			profileVisible ? store.getAccountProfile(profileAccountId) : null,
		),
		profile = $derived(
			profileState?.sections.hasAnySection ? profileState.snapshot : null,
		),
		isBalanceLoading = $derived(
			!!profileState &&
				!profileState.sections.hasSection(
					AccountProfileSection.ProfileSectionRobux,
				) &&
				!profileState.sections.isSectionUnavailable(
					AccountProfileSection.ProfileSectionRobux,
				) &&
				!profileState.error,
		),
		unreadNotifications = $derived(
			moderated ? 0 : (profile?.unreadNotificationCount ?? 0),
		),
		hasAbout = $derived(
			!!profileState &&
				(profileState.sections.hasSection(
					AccountProfileSection.ProfileSectionAbout,
				) ||
					profileState.sections.isSectionUnavailable(
						AccountProfileSection.ProfileSectionAbout,
					)),
		),
		refreshing = $derived(
			moderated ? checkingModeration : (profileState?.loading ?? false),
		)

	$effect(() => {
		if (!store.vault.unlocked || !hasAccount || moderated) {
			workspace.accountPage = "profile"
		}
	})

	$effect(() => moderationClock.track(banEndsAtMs(selectedAccount.moderation)))

	$effect(() => {
		const state = profileState
		if (!state || !pageVisibility.isShown) return
		untrack(() => state.enter())
	})

	$effect(() => {
		const onlineStatus = appSettings.onlineStatus.profile,
			accountId = profileAccountId
		if (
			!appSettings.initialized ||
			!profileVisible ||
			!pageVisibility.isShown ||
			!onlineStatus.enabled
		) {
			return
		}
		untrack(() => void store.refreshOnlineStatuses(accountId))
		const timer = window.setInterval(
			() => void store.refreshOnlineStatuses(accountId),
			onlineStatus.intervalSeconds * 1000,
		)
		return () => window.clearInterval(timer)
	})

	function openPage(page: AccountPage): void {
		workspace.closeAccountMenus()
		workspace.accountPage = page
	}

	setProfilePages({
		get label() {
			return `${selectedAccount.displayName} pages`
		},
		get tabs() {
			return accountPages.map((page) => ({
				id: page,
				label: accountPageLabels[page],
				unreadCount: page === "notifications" ? unreadNotifications : 0,
				unavailableReason: pageUnavailableReason,
			}))
		},
		get currentPage() {
			return workspace.accountPage === "profile" ? "" : workspace.accountPage
		},
		openPage: (page) => {
			const accountPage = accountPages.find((knownPage) => knownPage === page)
			if (accountPage) openPage(accountPage)
		},
		openOverview: () => void backToProfile(),
	})

	async function backToProfile(): Promise<void> {
		const closedPage = workspace.accountPage
		workspace.accountPage = "profile"
		await tick()
		const opener = profileSection?.querySelector<HTMLButtonElement>(
			`[data-subpage-button="${closedPage}"]`,
		)
		opener?.closest("[data-workspace-scroll]")?.scrollTo({ top: 0 })
		opener?.focus({ preventScroll: true })
	}

	notifyErrors("Profile not loaded", () =>
		profileVisible ? (profileState?.error ?? "") : "",
	)
</script>

{#if workspace.accountPage === "settings" && hasAccount && store.vault.unlocked}
	<AccountSettingsPage {store} {workspace} {notifications} />
{:else if workspace.accountPage === "chats" && hasAccount && store.vault.unlocked}
	<ChatsPage {store} {workspace} />
{:else if workspace.accountPage === "avatar" && hasAccount && store.vault.unlocked}
	<AvatarPage
		accounts={store}
		{notifications}
		accountId={selectedAccount.id}
		userId={Number(selectedAccount.userId)}
		username={selectedAccount.username} />
{:else if workspace.accountPage === "inventory" && hasAccount && store.vault.unlocked}
	<InventoryPage
		accounts={store}
		{notifications}
		accountId={selectedAccount.id}
		userId={Number(selectedAccount.userId)}
		username={selectedAccount.username}
		bind:sidebarWidth={workspace.inventorySidebarWidth} />
{:else if workspace.accountPage === "notifications" && hasAccount && store.vault.unlocked}
	<RobloxNotificationsPage
		{store}
		onOpenChat={(userId) => {
			workspace.chatUserIdToOpen = userId
			openPage("chats")
		}}
		{onFollowUser} />
{:else}
	<section
		bind:this={profileSection}
		class="flex flex-col gap-5 p-4 pb-5"
		aria-labelledby="account-name">
		<div class="flex items-start gap-4">
			{#if hasAccount}
				<AvatarPortraitFrame
					portraitLoad={avatarPortrait}
					headshotUrl={selectedAccount.avatarUrl}
					label={`@${selectedAccount.username}'s avatar`}
					onOpenAvatar={() => {
						if (!pageUnavailableReason) openPage("avatar")
					}} />
			{:else}
				<ImageMenuTarget
					imageUrl={selectedAccount.avatarUrl}
					label={`@${selectedAccount.username}'s profile image`}>
					<AccountAvatar
						account={selectedAccount}
						className="size-20 text-lg" />
				</ImageMenuTarget>
			{/if}
			<div class="flex min-w-0 flex-1 flex-col gap-1 pt-1">
				<h2
					id="account-name"
					class="flex min-w-0 items-center gap-1 text-lg leading-tight font-semibold tracking-tight text-fg">
					<span class="truncate select-text"
						>{selectedAccount.displayName}</span>
					{#if selectedAccount.favorite}<Icon
							svg={starFillIcon}
							class="shrink-0 animate-pop text-accent-strong"
							size={16}
							aria-hidden="true" />{/if}
					{#if profile?.verifiedBadge}<Icon
							svg={sealCheckIcon}
							class="shrink-0 text-accent-strong"
							size={16}
							aria-label="Verified" />{/if}
				</h2>
				{#if hasAccount}
					<div
						class="flex min-w-0 flex-wrap items-center gap-1 font-mono text-2xs wrap-anywhere text-fg-muted">
						<span class="select-text">@{selectedAccount.username}</span>
						<span class="text-fg-faint" aria-hidden="true">·</span>
						<span class="select-text">ID: {selectedAccount.userId}</span>
					</div>
					{#if sessionAttention}
						<div class="flex items-center gap-2 text-xs text-warning">
							<Icon
								svg={warningCircleIcon}
								size={14}
								aria-hidden="true" />
							<span>{sessionAttention}</span>
						</div>
					{/if}
					{#if store.vault.unlocked && !moderated}
						<AccountOnlineStatus
							onlineStatus={selectedAccount.onlineStatus}
							onJoin={onJoinServer} />
					{/if}
					{#if profileState}
						{@const sections = profileState.sections}
						<ProfileStats
							label="Relationship counts"
							stats={relationshipStats(
								{
									friends: profile?.friendCount ?? null,
									followers: profile?.followerCount ?? null,
									following: profile?.followingCount ?? null,
								},
								(list) =>
									sections.isSectionPending(
										list === SocialList.SocialFollowers
											? AccountProfileSection.ProfileSectionFollowers
											: list === SocialList.SocialFollowing
												? AccountProfileSection.ProfileSectionFollowing
												: AccountProfileSection.ProfileSectionFriends,
									),
								onShowSocial,
							)} />
					{/if}
					<div class="mt-1 flex min-h-6 flex-wrap items-center gap-2">
						{#if store.vault.unlocked && !moderated}
							<AccountProfileBalance
								snapshot={profile}
								loading={isBalanceLoading} />
							{#if profile}<AccountMembershipBadges
									snapshot={profile} />{/if}
						{/if}
						{#each selectedCustomTags as tag (tag.id)}
							<span class="tag-pill">
								<Icon svg={hashIcon} size={12} aria-hidden="true" />
								<span>{tag.name}</span>
							</span>
						{/each}
						<div class="relative inline-flex" data-account-menu>
							<button
								bind:this={tagsButton}
								class="button-icon size-6"
								type="button"
								aria-label="Edit tags"
								aria-expanded={workspace.activeAccountMenu ===
									"profile-tags"}
								data-open={workspace.activeAccountMenu ===
									"profile-tags" || undefined}
								data-tooltip="Edit tags"
								disabled={store.busy}
								onclick={() =>
									workspace.toggleAccountMenu("profile-tags")}>
								<Icon svg={tagIcon} size={14} aria-hidden="true" />
							</button>
							{#if workspace.activeAccountMenu === "profile-tags"}
								<div
									class="menu"
									data-account-menu
									use:floatMenu={{
										anchor: tagsButton,
										align: "start",
									}}
									use:menuKeyboard
									in:menuIn
									out:menuOut>
									<TagPicker
										{store}
										accounts={[selectedAccount]}
										onManageTags={() =>
											onManageTags(selectedAccount.id)} />
								</div>
							{/if}
						</div>
					</div>
				{/if}
			</div>
			<div class="relative flex shrink-0 items-center gap-1">
				{#if hasAccount && sessionAttention && !moderated}
					<button
						class="button-primary animate-enter"
						type="button"
						disabled={store.busy}
						onclick={onReplaceCookie}>
						<Icon svg={cookieIcon} size={14} aria-hidden="true" />
						Update cookie
					</button>
				{/if}
				{#if canBeReactivated && store.vault.unlocked}
					<button
						class="button-primary animate-enter"
						type="button"
						aria-busy={reactivating}
						data-tooltip={reactivatable
							? "Accept the moderation and use this account again"
							: "Available when the ban ends"}
						disabled={!reactivatable ||
							store.reactivatingAccountId !== null}
						onclick={() =>
							void store.reactivateAccount(selectedAccount.id)}>
						{#if reactivating}
							<Icon
								svg={circleNotchIcon}
								class="spinner"
								size={14}
								aria-hidden="true" />
							Reactivating…
						{:else}
							<Icon
								svg={arrowCounterClockwiseIcon}
								data-turn
								size={14}
								aria-hidden="true" />
							Reactivate
						{/if}
					</button>
				{/if}
				{#if canJoinFromHeader}
					<OnlineStatusJoinButton
						onlineStatus={selectedAccount.onlineStatus}
						onJoin={onJoinServer} />
				{/if}
				{#if (hasAccount && sessionAttention && !moderated) || (canBeReactivated && store.vault.unlocked) || hasJoinAction}<span
						class="mx-1 h-4 w-px bg-line-strong"
						aria-hidden="true"></span
					>{/if}
				<button
					class="button-icon"
					type="button"
					data-tooltip={moderated
						? "Check moderation status"
						: "Refresh account data"}
					aria-label={moderated
						? "Check moderation status"
						: refreshing
							? "Refreshing account data…"
							: "Refresh account data"}
					aria-busy={refreshing}
					disabled={!hasAccount || !store.vault.unlocked || refreshing}
					onclick={() => {
						if (moderated) {
							void store.refreshModeration(selectedAccount.id)
							return
						}
						showcaseRefreshCount++
						void profileState?.refresh(true)
					}}>
					<RefreshIcon isRefreshing={refreshing} />
				</button>
				<button
					class="button-icon"
					data-account-menu
					type="button"
					aria-label="More actions"
					aria-expanded={workspace.accountContextMenu?.accountId ===
						selectedAccount.id && !workspace.accountContextMenu.anchor}
					data-tooltip="More actions"
					data-tooltip-side="bottom-end"
					disabled={!hasAccount}
					onclick={(event) => {
						if (workspace.accountContextMenu) workspace.closeAccountMenus()
						else
							workspace.openAccountContextMenu(
								event,
								selectedAccount.id,
								true,
							)
					}}>
					<Icon svg={dotsThreeIcon} size={16} aria-hidden="true" />
				</button>
			</div>
		</div>

		{#if hasAccount && moderated}
			{#key selectedAccount.id}
				<AccountModerationPanel
					moderation={selectedAccount.moderation}
					onOpenBrowser={() => onOpenBrowser(selectedAccount.id)} />
			{/key}
		{/if}

		{#if hasAccount}
			<div class="-mx-4 h-8 border-b border-line-subtle">
				<ProfileTabs />
			</div>

			<div class="grid min-w-0 items-start gap-5 @4xl:grid-cols-3 @4xl:gap-x-6">
				{#if profileState && hasAbout}
					<AccountProfileDescription
						snapshot={profileState.snapshot}
						isDescriptionUnavailable={!profileState.sections.hasSection(
							AccountProfileSection.ProfileSectionAbout,
						)}
						disabled={store.busy ||
							profileState.sections.loadingSections.includes(
								AccountProfileSection.ProfileSectionAbout,
							) ||
							!store.vault.unlocked}
						{notifications}
						{onEditDescription} />
				{:else if profileVisible && profileState?.sections.isSectionPending(AccountProfileSection.ProfileSectionAbout)}
					<section
						class="flex flex-col gap-2 @4xl:col-span-2"
						aria-busy="true">
						<h3
							class="flex h-6 items-center text-xs font-semibold text-fg-muted">
							About
						</h3>
						<p class="text-block" data-variant="prose">
							<Skeleton className="w-72" label="Loading description…" />
						</p>
					</section>
				{/if}

				<AccountProfileDetails
					account={selectedAccount}
					snapshot={profile}
					sections={profileState?.sections ?? null}
					portraitLoad={avatarPortrait} />
			</div>
			{#if profile && profileState?.sections.hasSection(AccountProfileSection.ProfileSectionConnections) && store.accountTotal > 1}
				<FriendConnections
					connections={profile.connections ?? []}
					accounts={store} />
			{/if}
			{#if profileVisible && hasAccount && Number.isSafeInteger(robloxUserId) && store.vault.unlocked}
				<ProfileShowcase
					accounts={store}
					accountId={selectedAccount.id}
					userId={robloxUserId}
					username={selectedAccount.username}
					refreshCount={showcaseRefreshCount}
					{onShowCreations} />
			{/if}
		{/if}
	</section>
{/if}

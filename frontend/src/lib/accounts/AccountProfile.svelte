<script lang="ts">
	import bellIcon from "@phosphor-icons/core/regular/bell.svg?raw"
	import chatsIcon from "@phosphor-icons/core/regular/chats.svg?raw"
	import dotsThreeIcon from "@phosphor-icons/core/regular/dots-three.svg?raw"
	import gearIcon from "@phosphor-icons/core/regular/gear.svg?raw"
	import hashIcon from "@phosphor-icons/core/regular/hash.svg?raw"
	import sealCheckIcon from "@phosphor-icons/core/regular/seal-check.svg?raw"
	import starFillIcon from "@phosphor-icons/core/fill/star-fill.svg?raw"
	import tagIcon from "@phosphor-icons/core/regular/tag.svg?raw"
	import warningCircleIcon from "@phosphor-icons/core/regular/warning-circle.svg?raw"
	import Icon from "../shared/Icon.svelte"
	import RefreshIcon from "../shared/RefreshIcon.svelte"
	import { tick, untrack } from "svelte"
	import arrowCounterClockwiseIcon from "@phosphor-icons/core/regular/arrow-counter-clockwise.svg?raw"
	import circleNotchIcon from "@phosphor-icons/core/regular/circle-notch.svg?raw"
	import { SessionState } from "../backend/bridge"
	import type { RobloxNotificationUser } from "../backend/bridge"
	import AccountModerationPanel from "./AccountModerationPanel.svelte"
	import {
		canReactivate,
		hasReactivation,
		ModerationClock,
	} from "./moderation.svelte"
	import type { SocialListKind } from "../social/social-store.svelte"
	import ChatsPage from "../chats/ChatsPage.svelte"
	import RobloxNotificationsPage from "../roblox-notifications/RobloxNotificationsPage.svelte"
	import SocialCounts from "../social/SocialCounts.svelte"
	import type { WorkspaceState } from "../layout/workspace-state.svelte"
	import type { NotificationCenter } from "../notifications/notification-center.svelte"
	import { appSettings } from "../settings/settings-store.svelte"
	import { floatMenu, menuIn, menuKeyboard, menuOut } from "../shared/presence"
	import AccountAvatar from "./AccountAvatar.svelte"
	import AccountMembershipBadges from "./AccountMembershipBadges.svelte"
	import AccountPresence from "./AccountPresence.svelte"
	import AccountProfileBalance from "./AccountProfileBalance.svelte"
	import AccountProfileAbout from "./AccountProfileAbout.svelte"
	import AccountProfileDetails from "./AccountProfileDetails.svelte"
	import AccountSettingsPage from "./AccountSettingsPage.svelte"
	import type { AccountStore } from "./account-store.svelte"
	import { getCustomTags } from "./account-model"
	import TagPicker from "./TagPicker.svelte"

	let {
		store,
		workspace,
		onManageTags,
		onJoinServer,
		onFollowUser,
		onSearchPlace,
		onSearchUniverse,
		onReplaceCookie,
		onEditDescription,
		onShowSocial,
		onOpenBrowser,
		notifications,
	}: {
		store: AccountStore
		workspace: WorkspaceState
		onManageTags: (accountId: number) => void
		onJoinServer: (placeId: number, jobId: string) => void
		onFollowUser: (user: RobloxNotificationUser) => void
		onSearchPlace: (placeId: number) => void
		onSearchUniverse: (universeId: number) => void
		onReplaceCookie: () => void
		onEditDescription: (description: string) => void
		onShowSocial: (list: SocialListKind) => void
		onOpenBrowser: (accountId: number) => void
		notifications: NotificationCenter
	} = $props()

	const selectedAccount = $derived(store.selectedAccount),
		selectedCustomTags = $derived(getCustomTags(selectedAccount)),
		profileAccountId = $derived(selectedAccount.id),
		hasAccount = $derived(profileAccountId !== 0),
		moderated = $derived(selectedAccount.state === SessionState.StateModerated),
		profileVisible = $derived(
			store.vault.unlocked &&
				hasAccount &&
				!moderated &&
				workspace.accountPage === "profile",
		),
		moderationClock = new ModerationClock(),
		canBeReactivated = $derived(
			moderated && hasReactivation(selectedAccount.moderation),
		),
		reactivatable = $derived(
			canReactivate(selectedAccount.moderation, moderationClock.nowMs),
		),
		reactivating = $derived(store.reactivatingAccountId === selectedAccount.id),
		checkingModeration = $derived(store.checkingAccountId === selectedAccount.id)

	let settingsButton = $state<HTMLButtonElement | undefined>(undefined),
		chatsButton = $state<HTMLButtonElement | undefined>(undefined),
		notificationsButton = $state<HTMLButtonElement | undefined>(undefined),
		tagsButton = $state<HTMLButtonElement | undefined>(undefined)
	const profileState = $derived(
			profileVisible ? store.getAccountProfile(profileAccountId) : null,
		),
		profile = $derived(
			profileState?.snapshot?.accountId === profileAccountId
				? profileState.snapshot
				: null,
		),
		refreshing = $derived(
			moderated ? checkingModeration : (profileState?.loading ?? false),
		)

	$effect(() => {
		if (!store.vault.unlocked || !hasAccount || moderated) {
			workspace.accountPage = "profile"
		}
	})

	$effect(() => moderationClock.track(selectedAccount.moderation))

	$effect(() => {
		const state = profileState
		if (!state) return
		untrack(() => state.enter())
	})

	$effect(() => {
		const presence = appSettings.presence.profile,
			accountId = profileAccountId
		if (!appSettings.initialized || !profileVisible || !presence.enabled) return
		untrack(() => void store.refreshPresences(accountId))
		const timer = window.setInterval(
			() => void store.refreshPresences(accountId),
			presence.intervalSeconds * 1000,
		)
		return () => window.clearInterval(timer)
	})

	function openPage(page: "settings" | "chats" | "notifications"): void {
		workspace.closeAccountMenus()
		workspace.accountPage = page
	}

	async function backToProfile(): Promise<void> {
		const opener = {
			profile: settingsButton,
			settings: settingsButton,
			chats: chatsButton,
			notifications: notificationsButton,
		}[workspace.accountPage]
		workspace.accountPage = "profile"
		await tick()
		opener?.closest("[data-workspace-scroll]")?.scrollTo({ top: 0 })
		opener?.focus({ preventScroll: true })
	}
</script>

{#if workspace.accountPage === "settings" && hasAccount && store.vault.unlocked}
	<AccountSettingsPage
		{store}
		{workspace}
		{notifications}
		onBack={() => void backToProfile()} />
{:else if workspace.accountPage === "chats" && hasAccount && store.vault.unlocked}
	<ChatsPage {store} {workspace} onBack={() => void backToProfile()} />
{:else if workspace.accountPage === "notifications" && hasAccount && store.vault.unlocked}
	<RobloxNotificationsPage
		{store}
		onBack={() => void backToProfile()}
		onOpenChat={(userId) => {
			workspace.chatUserIdToOpen = userId
			openPage("chats")
		}}
		{onFollowUser}
		{onSearchPlace} />
{:else}
	<section class="flex flex-col gap-4 p-3 pb-4" aria-labelledby="account-name">
		<div class="flex items-start gap-3 @max-xl:flex-col">
			<div class="flex min-w-0 flex-1 items-center gap-4">
				<button
					type="button"
					class="shrink-0 cursor-default rounded-md"
					aria-label="Profile image"
					aria-haspopup="menu"
					aria-expanded={workspace.imageContextMenu?.imageUrl ===
						selectedAccount.avatarUrl}
					disabled={!selectedAccount.avatarUrl}
					onkeydown={(event) => {
						if (
							event.key === "ContextMenu" ||
							(event.shiftKey && event.key === "F10")
						) {
							workspace.openImageContextMenu(
								event,
								selectedAccount.avatarUrl,
							)
						}
					}}
					oncontextmenu={(event) =>
						workspace.openImageContextMenu(
							event,
							selectedAccount.avatarUrl,
						)}>
					<AccountAvatar
						account={selectedAccount}
						className="size-20 text-lg" />
				</button>
				<div class="flex min-w-0 flex-col gap-1">
					<h2
						id="account-name"
						class="flex min-w-0 items-center gap-1 text-lg leading-tight font-semibold tracking-tight text-fg">
						<span class="truncate">{selectedAccount.displayName}</span>
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
							<span>@{selectedAccount.username}</span>
							<span class="text-fg-faint" aria-hidden="true">-</span>
							<span>ID: {selectedAccount.userId}</span>
						</div>
						{#if selectedAccount.state === SessionState.StateReauthRequired || selectedAccount.state === SessionState.StateChallenged}
							<div class="flex items-center gap-2 text-xs text-warning">
								<Icon
									svg={warningCircleIcon}
									size={14}
									aria-hidden="true" />
								<span
									>{selectedAccount.state ===
									SessionState.StateReauthRequired
										? "Sign-in required"
										: "Security challenge"}</span>
								<button
									class="button"
									type="button"
									disabled={store.busy}
									onclick={onReplaceCookie}>
									Replace cookie
								</button>
							</div>
						{/if}
						{#if store.vault.unlocked && !moderated}
							<AccountPresence
								presence={selectedAccount.presence}
								onJoin={onJoinServer}
								{onSearchPlace}
								{onSearchUniverse} />
							<div class="flex flex-wrap items-center gap-2">
								<AccountProfileBalance
									snapshot={profile}
									loading={profileState?.loading ||
										(!profile && !profileState?.error)} />
								{#if profile}<AccountMembershipBadges
										snapshot={profile} />{/if}
							</div>
						{/if}
					{/if}
				</div>
			</div>
			<div class="relative flex items-center gap-1 @max-xl:self-end">
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
				<button
					class="button-icon"
					type="button"
					data-tooltip={moderated
						? "Check moderation status"
						: "Refresh account data"}
					aria-label={moderated
						? "Check moderation status"
						: refreshing
							? "Refreshing account data"
							: "Refresh account data"}
					aria-busy={refreshing}
					disabled={!hasAccount || !store.vault.unlocked || refreshing}
					onclick={() =>
						moderated
							? void store.refreshModeration(selectedAccount.id)
							: void profileState?.refresh(true)}>
					<RefreshIcon isRefreshing={refreshing} />
				</button>
				<button
					class="button-icon"
					type="button"
					aria-label="Notifications"
					data-tooltip={moderated
						? "Notifications are unavailable while moderated"
						: "Notifications"}
					disabled={!hasAccount || !store.vault.unlocked || moderated}
					bind:this={notificationsButton}
					onclick={() => openPage("notifications")}>
					<Icon svg={bellIcon} size={16} aria-hidden="true" />
				</button>
				<button
					class="button-icon"
					type="button"
					aria-label="Chats"
					data-tooltip={moderated
						? "Chats are unavailable while moderated"
						: "Chats"}
					disabled={!hasAccount || !store.vault.unlocked || moderated}
					bind:this={chatsButton}
					onclick={() => openPage("chats")}>
					<Icon svg={chatsIcon} size={16} aria-hidden="true" />
				</button>
				<button
					class="button-icon"
					type="button"
					aria-label="Account settings"
					data-tooltip={moderated
						? "Settings are unavailable while moderated"
						: "Account settings"}
					disabled={!hasAccount || !store.vault.unlocked || moderated}
					bind:this={settingsButton}
					onclick={() => openPage("settings")}>
					<Icon svg={gearIcon} size={16} aria-hidden="true" />
				</button>
				<button
					class="button-icon"
					data-account-menu
					type="button"
					aria-label="More account actions"
					aria-expanded={workspace.accountContextMenu?.accountId ===
						selectedAccount.id}
					data-tooltip="Show more account actions"
					data-tooltip-side="bottom-end"
					disabled={!hasAccount}
					onclick={(event) => {
						if (workspace.accountContextMenu) workspace.closeAccountMenus()
						else workspace.openAccountContextMenu(event, selectedAccount.id)
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
			<div class="flex flex-wrap items-center gap-1">
				{#if selectedAccount.favorite}
					<span class="tag-pill-favorite">
						<Icon svg={starFillIcon} size={12} aria-hidden="true" />
						<span>Favorite</span>
					</span>
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
						aria-label="Edit account tags"
						aria-expanded={workspace.activeAccountMenu === "profile-tags"}
						data-open={workspace.activeAccountMenu === "profile-tags" ||
							undefined}
						data-tooltip="Edit account tags"
						disabled={store.busy}
						onclick={() => workspace.toggleAccountMenu("profile-tags")}>
						<Icon svg={tagIcon} size={14} aria-hidden="true" />
					</button>
					{#if workspace.activeAccountMenu === "profile-tags"}
						<div
							class="menu"
							data-account-menu
							use:floatMenu={{ anchor: tagsButton, align: "start" }}
							use:menuKeyboard
							in:menuIn
							out:menuOut>
							<TagPicker
								{store}
								accounts={[selectedAccount]}
								onManageTags={() => onManageTags(selectedAccount.id)} />
						</div>
					{/if}
				</div>
			</div>

			{#if profile}
				<SocialCounts
					friends={profile.friendCount}
					followers={profile.followerCount}
					following={profile.followingCount}
					onShow={onShowSocial} />
			{/if}

			{#if profile}
				<AccountProfileAbout
					snapshot={profile}
					disabled={store.busy ||
						!!profileState?.loading ||
						!store.vault.unlocked}
					{notifications}
					{onEditDescription} />
			{/if}

			<AccountProfileDetails account={selectedAccount} snapshot={profile} />
			{#if profileVisible && profileState?.error}
				<p class="text-xs text-danger-muted" role="alert">
					{profileState.error}
				</p>
			{:else if (profile?.unavailable?.length ?? 0) > 0}
				<p class="text-xs text-danger-muted" role="alert">
					Some profile details could not be refreshed. Refresh to retry.
				</p>
			{/if}
		{/if}
	</section>
{/if}

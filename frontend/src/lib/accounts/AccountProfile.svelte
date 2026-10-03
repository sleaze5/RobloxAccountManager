<script lang="ts">
	import BadgeCheck from "@lucide/svelte/icons/badge-check"
	import Settings from "@lucide/svelte/icons/settings"
	import Hash from "@lucide/svelte/icons/hash"
	import LoaderCircle from "@lucide/svelte/icons/loader-circle"
	import MoreHorizontal from "@lucide/svelte/icons/ellipsis"
	import MessagesSquare from "@lucide/svelte/icons/messages-square"
	import RefreshCw from "@lucide/svelte/icons/refresh-cw"
	import Star from "@lucide/svelte/icons/star"
	import Tags from "@lucide/svelte/icons/tags"
	import { tick, untrack } from "svelte"
	import { PresenceType } from "../backend/bridge"
	import ChatsPage from "../chats/ChatsPage.svelte"
	import type { WorkspaceState } from "../layout/workspace-state.svelte"
	import type { NotificationCenter } from "../notifications/notification-center.svelte"
	import CompactNumber from "../shared/CompactNumber.svelte"
	import Timestamp from "../shared/Timestamp.svelte"
	import { appSettings } from "../settings/settings-store.svelte"
	import { menuIn, menuOut } from "../shared/presence"
	import AccountAvatar from "./AccountAvatar.svelte"
	import AccountMembershipBadges from "./AccountMembershipBadges.svelte"
	import AccountPresenceCard from "./AccountPresenceCard.svelte"
	import AccountProfileBalance from "./AccountProfileBalance.svelte"
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
		notifications,
	}: {
		store: AccountStore
		workspace: WorkspaceState
		onManageTags: (accountId: number) => void
		onJoinServer: (placeId: number, jobId: string) => void
		notifications: NotificationCenter
	} = $props()

	const selectedAccount = $derived(store.selectedAccount),
		selectedCustomTags = $derived(getCustomTags(selectedAccount)),
		profileAccountId = $derived(selectedAccount.id),
		hasAccount = $derived(profileAccountId !== 0),
		profileVisible = $derived(
			store.vault.unlocked && hasAccount && workspace.accountPage === "profile",
		)

	let settingsButton = $state<HTMLButtonElement | undefined>(undefined),
		chatsButton = $state<HTMLButtonElement | undefined>(undefined)
	const profileState = $derived(
			profileVisible ? store.getAccountProfile(profileAccountId) : null,
		),
		profile = $derived(
			profileState?.snapshot?.accountId === profileAccountId
				? profileState.snapshot
				: null,
		)

	$effect(() => {
		if (!store.vault.unlocked || !hasAccount) workspace.accountPage = "profile"
	})

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

	function openPage(page: "settings" | "chats"): void {
		workspace.closeAccountMenus()
		workspace.accountPage = page
	}

	async function backToProfile(): Promise<void> {
		const opener = workspace.accountPage === "chats" ? chatsButton : settingsButton
		workspace.accountPage = "profile"
		await tick()
		opener?.closest(".workspace-scroll")?.scrollTo({ top: 0 })
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
	<ChatsPage {store} onBack={() => void backToProfile()} />
{:else}
	<section class="profile-page" aria-labelledby="account-name">
		<div class="profile-head">
			<div class="selected-identity">
				<button
					type="button"
					class="profile-image-trigger"
					aria-label="Profile image"
					aria-haspopup="menu"
					aria-expanded={workspace.imageContextMenu?.accountId ===
						selectedAccount.id}
					disabled={!selectedAccount.avatarUrl}
					onkeydown={(event) => {
						if (
							event.key === "ContextMenu" ||
							(event.shiftKey && event.key === "F10")
						) {
							workspace.openImageContextMenu(
								event,
								selectedAccount.id,
								selectedAccount.avatarUrl,
							)
						}
					}}
					oncontextmenu={(event) =>
						workspace.openImageContextMenu(
							event,
							selectedAccount.id,
							selectedAccount.avatarUrl,
						)}>
					<AccountAvatar
						account={selectedAccount}
						className="selected-avatar"
						showPresence={true} />
				</button>
				<div class="identity-copy">
					<h2 id="account-name" class="account-display-name">
						<span>{selectedAccount.displayName}</span>
						{#if selectedAccount.favorite}<Star
								class="favorite-name-star"
								size={13}
								fill="currentColor"
								aria-hidden="true" />{/if}
						{#if profile?.verifiedBadge}<BadgeCheck
								class="profile-verified-badge"
								size={15}
								aria-label="Verified" />{/if}
					</h2>
					{#if hasAccount}
						<div class="identity-meta">
							<span>@{selectedAccount.username}</span>
							<span class="meta-separator" aria-hidden="true">-</span>
							<span>ID: {selectedAccount.userId}</span>
						</div>
						{#if store.vault.unlocked}
							<div class="profile-balance-row">
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
			<div class="header-actions">
				<button
					class="control-button"
					type="button"
					data-tooltip="Refresh account data"
					aria-label={profileState?.loading
						? "Refreshing account data"
						: "Refresh account data"}
					aria-busy={profileState?.loading ?? false}
					disabled={!hasAccount ||
						!store.vault.unlocked ||
						profileState?.loading}
					onclick={() => void profileState?.refresh(true)}>
					{#if profileState?.loading}
						<LoaderCircle class="spinner" size={15} aria-hidden="true" />
					{:else}
						<RefreshCw size={15} aria-hidden="true" />
					{/if}
				</button>
				<button
					class="control-button"
					type="button"
					aria-label="Chats"
					data-tooltip="Chats"
					disabled={!hasAccount || !store.vault.unlocked}
					bind:this={chatsButton}
					onclick={() => openPage("chats")}>
					<MessagesSquare size={15} />
				</button>
				<button
					class="control-button"
					type="button"
					aria-label="Account settings"
					data-tooltip="Account settings"
					disabled={!hasAccount || !store.vault.unlocked}
					bind:this={settingsButton}
					onclick={() => openPage("settings")}>
					<Settings size={15} />
				</button>
				<button
					class="icon-action bordered"
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
					<MoreHorizontal size={18} />
				</button>
			</div>
		</div>

		{#if hasAccount}
			<div class="identity-section">
				{#if selectedAccount.favorite}
					<span class="tag-pill favorite">
						<Star size={11} fill="currentColor" />
						<span>Favorite</span>
					</span>
				{/if}
				{#each selectedCustomTags as tag (tag.id)}
					<span class="tag-pill">
						<Hash size={11} aria-hidden="true" />
						<span>{tag.name}</span>
					</span>
				{/each}
				<div class="edit-tags-host" data-account-menu>
					<button
						class:open={workspace.activeAccountMenu === "profile-tags"}
						class="icon-action bordered"
						type="button"
						aria-label="Edit account tags"
						aria-expanded={workspace.activeAccountMenu === "profile-tags"}
						data-tooltip="Edit account tags"
						disabled={store.busy}
						onclick={() => workspace.toggleAccountMenu("profile-tags")}>
						<Tags size={13} />
					</button>
					{#if workspace.activeAccountMenu === "profile-tags"}
						<div
							class="profile-tag-menu account-menu"
							data-account-menu
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

			{#if profile && (profile.friendCount !== null || profile.followerCount !== null || profile.followingCount !== null)}
				<dl class="profile-social" aria-label="Connections">
					{#each [["Friends", profile.friendCount], ["Followers", profile.followerCount], ["Following", profile.followingCount]] as const as [label, value] (label)}
						{#if value !== null}
							<div>
								<dt>{label}</dt>
								<dd><CompactNumber {value} /></dd>
							</div>
						{/if}
					{/each}
				</dl>
			{/if}

			{#if store.vault.unlocked && selectedAccount.presence.userPresenceType !== PresenceType.PresenceTypeOffline}
				<AccountPresenceCard
					presence={selectedAccount.presence}
					updatesEnabled={appSettings.presence.profile.enabled}
					onJoin={onJoinServer} />
			{/if}

			<AccountProfileDetails account={selectedAccount} snapshot={profile} />
			{#if profileVisible && profileState?.error}
				<p class="profile-read-error" role="alert">{profileState.error}</p>
			{:else if (profile?.unavailable?.length ?? 0) > 0}
				<p class="profile-read-error" role="alert">
					Some profile details could not be refreshed. Refresh to retry.
				</p>
			{/if}

			{#if profile?.description}
				<section
					class="profile-section"
					aria-labelledby="profile-description-heading">
					<h3 id="profile-description-heading" class="profile-section-label">
						Description
					</h3>
					<pre class="text-block">{profile.description}</pre>
				</section>
			{/if}
		{/if}
	</section>
{/if}

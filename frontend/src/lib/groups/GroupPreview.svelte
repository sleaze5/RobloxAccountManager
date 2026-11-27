<script lang="ts">
	import circleNotchIcon from "@phosphor-icons/core/regular/circle-notch.svg?raw"
	import signInIcon from "@phosphor-icons/core/regular/sign-in.svg?raw"
	import signOutIcon from "@phosphor-icons/core/regular/sign-out.svg?raw"
	import sealCheckIcon from "@phosphor-icons/core/regular/seal-check.svg?raw"
	import tagIcon from "@phosphor-icons/core/regular/tag.svg?raw"
	import warningIcon from "@phosphor-icons/core/regular/warning.svg?raw"
	import { accountBackend, GroupSource } from "../backend/bridge"
	import type { GroupProfile } from "../backend/bridge"
	import CreatorByline from "../shared/CreatorByline.svelte"
	import { errorMessage } from "../shared/errors"
	import { notifyErrors } from "../notifications/error-notifications.svelte"
	import RobloxActionStrip from "../shared/RobloxActionStrip.svelte"
	import Icon from "../shared/Icon.svelte"
	import ProfilePreviewLayout from "../shared/ProfilePreviewLayout.svelte"
	import type { ProfilePreviewFrame } from "../shared/ProfilePreviewLayout.svelte"
	import ProfileStats from "../shared/ProfileStats.svelte"
	import Skeleton from "../shared/Skeleton.svelte"
	import GroupIcon from "./GroupIcon.svelte"
	import GroupMembershipFacts from "./GroupMembershipFacts.svelte"
	import GroupStatusBadges from "./GroupStatusBadges.svelte"
	import { groupStats } from "./group-stats"

	// GroupPreview shows the information of a group as the account sees it:
	// its owner, members, description, and roles, and joins or leaves it.
	let {
		accountId,
		accountUsername,
		groupId,
		frame,
		onMembershipChanged,
	}: {
		accountId: number
		// accountUsername names the account that acts on Roblox.
		accountUsername: string
		groupId: number
		frame: ProfilePreviewFrame
		onMembershipChanged: (groupId: number) => void
	} = $props()

	const metaClass =
			"flex min-w-0 flex-wrap items-center gap-1 font-mono text-2xs wrap-anywhere text-fg-muted",
		sectionHeadingClass = "text-xs font-semibold text-fg-muted"

	let profile = $state.raw<GroupProfile | null>(null),
		loadError = $state(""),
		loadCount = $state(0),
		pendingAction = $state<"join" | "leave" | null>(null),
		actionError = $state("")

	$effect(() => {
		void loadCount
		const request = accountBackend.GetGroupProfile(accountId, groupId)
		let isCurrent = true
		loadError = ""
		void (async () => {
			try {
				const loadedProfile = await request
				if (isCurrent) profile = loadedProfile
			} catch (error) {
				if (isCurrent)
					loadError = errorMessage(
						error,
						"Could not load the group. Try again.",
					)
			}
		})()
		return () => {
			isCurrent = false
			void request.cancel()
		}
	})

	const group = $derived(profile?.details.group ?? null),
		membership = $derived(profile?.membership ?? null),
		isMember = $derived(!!membership && membership.role.rank > 0),
		roles = $derived(profile?.roles ?? []),
		canJoin = $derived(
			!!membership &&
				!isMember &&
				!membership.isPendingJoin &&
				!profile?.details.isLocked,
		),
		canLeave = $derived(isMember && !membership?.isOwner),
		isRolesUnavailable = $derived(
			profile?.unavailable?.includes(GroupSource.GroupSourceRoles) ?? false,
		)

	// changeMembership joins or leaves the group, then reads the group again,
	// and returns whether it succeeded.
	async function changeMembership(action: "join" | "leave"): Promise<boolean> {
		if (pendingAction) return false
		pendingAction = action
		actionError = ""
		try {
			await (action === "join"
				? accountBackend.JoinGroup(accountId, groupId)
				: accountBackend.LeaveGroup(accountId, groupId))
			onMembershipChanged(groupId)
			loadCount++
			return true
		} catch (error) {
			actionError = errorMessage(
				error,
				action === "join"
					? "Could not join the group. Try again."
					: "Could not leave the group. Try again.",
			)
			return false
		} finally {
			pendingAction = null
		}
	}

	function confirmJoin(): void {
		if (!group) return
		frame.confirm({
			title: group.isPublicEntryAllowed
				? "Join group on Roblox?"
				: "Request to join on Roblox?",
			description: group.isPublicEntryAllowed
				? `This account becomes a member of ${group.name}.`
				: `${group.name} reviews the request before this account becomes a member.`,
			confirmLabel: group.isPublicEntryAllowed ? "Join group" : "Request to join",
			progressLabel: "Joining…",
			tone: "primary",
			run: () => changeMembership("join"),
		})
	}

	function confirmLeave(): void {
		if (!group) return
		frame.confirm({
			title: "Leave group?",
			description: `This account will leave ${group.name} and lose its role. Joining again may need approval.`,
			confirmLabel: "Leave group",
			progressLabel: "Leaving…",
			tone: "danger",
			run: () => changeMembership("leave"),
		})
	}

	notifyErrors("Group not changed", () => actionError)
</script>

{#snippet quickActions()}
	{#if canLeave || canJoin}
		<RobloxActionStrip username={accountUsername}>
			{#if canLeave}
				<button
					type="button"
					aria-label="Leave group"
					data-tooltip="Leave group"
					disabled={pendingAction !== null}
					onclick={confirmLeave}>
					<Icon svg={signOutIcon} size={14} aria-hidden="true" />
				</button>
			{:else}
				{@const joinLabel = group?.isPublicEntryAllowed
					? "Join group"
					: "Request to join"}
				<button
					type="button"
					aria-label={pendingAction === "join" ? "Joining…" : joinLabel}
					aria-busy={pendingAction === "join"}
					data-tooltip={joinLabel}
					disabled={pendingAction !== null}
					onclick={confirmJoin}>
					<Icon
						svg={pendingAction === "join" ? circleNotchIcon : signInIcon}
						class={pendingAction === "join" ? "spinner" : undefined}
						size={14}
						aria-hidden="true" />
				</button>
			{/if}
		</RobloxActionStrip>
	{/if}
{/snippet}

<ProfilePreviewLayout
	{frame}
	quickActions={canJoin || canLeave ? quickActions : undefined}>
	{#snippet image()}
		<GroupIcon url={group?.iconUrl ?? ""} size="xl" />
	{/snippet}
	{#snippet title()}
		{#if group}<span class="select-text">{group.name}</span
			>{#if group.hasVerifiedBadge}<Icon
					svg={sealCheckIcon}
					class="relative top-px ml-1 inline-block text-accent-strong"
					size={16}
					aria-label="Verified group" />{/if}{:else if loadError}Group
			unavailable{:else}<Skeleton className="w-48" label="Loading group…" />{/if}
	{/snippet}
	{#snippet summary()}
		<div class={metaClass}>
			<span class="select-text">ID: {groupId}</span>
		</div>
		{#if profile?.details.owner}
			<CreatorByline
				kind="User"
				creatorId={profile.details.owner.userId}
				name={profile.details.owner.username}
				hasVerifiedBadge={profile.details.owner.hasVerifiedBadge}
				role="Owner" />
		{:else if profile}
			<p class="text-xs text-fg-muted">Nobody owns this group.</p>
		{:else if !loadError}
			<span class="text-xs"><Skeleton className="w-32" label="" /></span>
		{/if}
		{#if group}<ProfileStats label="Group numbers" stats={groupStats(group)} />{/if}
		{#if profile}<GroupStatusBadges
				isPrimary={profile.membership.isPrimary}
				isPendingJoin={profile.membership.isPendingJoin}
				isLocked={profile.details.isLocked} />{/if}
	{/snippet}

	{#if loadError}
		<div class="empty-state py-6">
			<Icon svg={warningIcon} size={28} aria-hidden="true" />
			<p>{loadError}</p>
			<button class="button" type="button" onclick={() => loadCount++}
				>Try again</button>
		</div>
	{:else if group}
		{#if group.description}
			<section class="flex flex-col gap-3" aria-label="Description">
				<h3 class={sectionHeadingClass}>Description</h3>
				<p
					class="max-h-40 text-block overflow-y-auto select-text"
					data-variant="prose">
					{group.description}
				</p>
			</section>
		{/if}
		<section class="flex flex-col gap-3" aria-label="Details">
			<h3 class={sectionHeadingClass}>Details</h3>
			<dl class="facts">
				{#if membership}<GroupMembershipFacts
						{membership}
						isPublicEntryAllowed={group.isPublicEntryAllowed} />{/if}
				<div data-tone={isRolesUnavailable ? "muted" : undefined}>
					<dt><Icon svg={tagIcon} size={16} aria-hidden="true" />Roles</dt>
					<dd>
						{#if isRolesUnavailable}Unavailable{:else}{roles.length}
							<small class="truncate"
								>{roles
									.slice(-3)
									.reverse()
									.map((role) => role.name)
									.join(" · ")}</small
							>{/if}
					</dd>
				</div>
			</dl>
		</section>
	{:else}
		<span class="h-24 w-full skeleton rounded-lg" role="status"
			><span class="sr-only">Loading group…</span></span>
	{/if}
</ProfilePreviewLayout>

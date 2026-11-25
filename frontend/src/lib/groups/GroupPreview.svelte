<script lang="ts">
	import checkCircleFillIcon from "@phosphor-icons/core/fill/check-circle-fill.svg?raw"
	import clockIcon from "@phosphor-icons/core/regular/clock.svg?raw"
	import lockIcon from "@phosphor-icons/core/regular/lock.svg?raw"
	import signInIcon from "@phosphor-icons/core/regular/sign-in.svg?raw"
	import signOutIcon from "@phosphor-icons/core/regular/sign-out.svg?raw"
	import sealCheckIcon from "@phosphor-icons/core/regular/seal-check.svg?raw"
	import tagIcon from "@phosphor-icons/core/regular/tag.svg?raw"
	import usersThreeIcon from "@phosphor-icons/core/regular/users-three.svg?raw"
	import warningIcon from "@phosphor-icons/core/regular/warning.svg?raw"
	import { accountBackend, GroupSource } from "../backend/bridge"
	import type { GroupProfile } from "../backend/bridge"
	import CreatorByline from "../shared/CreatorByline.svelte"
	import { errorMessage } from "../shared/errors"
	import { notifyErrors } from "../notifications/error-notifications.svelte"
	import Icon from "../shared/Icon.svelte"
	import ProfilePreviewLayout from "../shared/ProfilePreviewLayout.svelte"
	import type { ProfilePreviewFrame } from "../shared/ProfilePreviewLayout.svelte"
	import ProfileStats from "../shared/ProfileStats.svelte"
	import Skeleton from "../shared/Skeleton.svelte"
	import GroupIcon from "./GroupIcon.svelte"
	import { groupStats } from "./group-stats"

	// GroupPreview shows the information of a group as the account sees it:
	// its owner, members, description, and roles, and joins or leaves it.
	let {
		accountId,
		groupId,
		frame,
		onMembershipChanged,
	}: {
		accountId: number
		groupId: number
		frame: ProfilePreviewFrame
		onMembershipChanged: (groupId: number) => void
	} = $props()

	const exactNumbers = new Intl.NumberFormat("en-US"),
		metaClass =
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
	{#if canLeave}
		<button
			class="button"
			type="button"
			disabled={pendingAction !== null}
			onclick={confirmLeave}>
			<Icon svg={signOutIcon} size={16} aria-hidden="true" />Leave
		</button>
	{:else if canJoin}
		<button
			class="button-primary"
			type="button"
			aria-busy={pendingAction === "join"}
			disabled={pendingAction !== null}
			onclick={() => void changeMembership("join")}>
			<Icon svg={signInIcon} size={16} aria-hidden="true" />
			{pendingAction === "join"
				? "Joining…"
				: group?.isPublicEntryAllowed
					? "Join"
					: "Request to join"}
		</button>
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
					class="relative top-px ml-1 inline-block text-fg-muted"
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
		{#if isMember || membership?.isPendingJoin || profile?.details.isLocked}
			<ul class="mt-1 flex flex-wrap gap-1" aria-label="Status">
				{#if isMember && membership}
					<li class="tag-pill-success">
						<Icon svg={checkCircleFillIcon} size={12} aria-hidden="true" />
						<span>{membership.role.name}</span>
					</li>
				{/if}
				{#if membership?.isPendingJoin}
					<li class="tag-pill">
						<Icon svg={clockIcon} size={12} aria-hidden="true" />
						<span>Join requested</span>
					</li>
				{/if}
				{#if profile?.details.isLocked}
					<li class="tag-pill-warning">
						<Icon svg={lockIcon} size={12} aria-hidden="true" />
						<span>Locked</span>
					</li>
				{/if}
			</ul>
		{/if}
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
			<section class="flex flex-col gap-3" aria-label="About">
				<h3 class={sectionHeadingClass}>About</h3>
				<p
					class="max-h-40 text-block overflow-y-auto select-text"
					data-variant="prose">
					{group.description}
				</p>
			</section>
		{/if}
		<section class="flex flex-col gap-3" aria-label="Information">
			<h3 class={sectionHeadingClass}>Information</h3>
			<dl class="facts">
				<div>
					<dt>
						<Icon
							svg={usersThreeIcon}
							size={16}
							aria-hidden="true" />Members
					</dt>
					<dd>{exactNumbers.format(group.memberCount)}</dd>
				</div>
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

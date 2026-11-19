<script lang="ts">
	import circleNotchIcon from "@phosphor-icons/core/regular/circle-notch.svg?raw"
	import sealCheckIcon from "@phosphor-icons/core/regular/seal-check.svg?raw"
	import userMinusIcon from "@phosphor-icons/core/regular/user-minus.svg?raw"
	import usersThreeIcon from "@phosphor-icons/core/regular/users-three.svg?raw"
	import warningIcon from "@phosphor-icons/core/regular/warning.svg?raw"
	import { onMount, untrack } from "svelte"
	import { accountBackend } from "../backend/bridge"
	import type { GroupMember, GroupMemberPage, GroupRole } from "../backend/bridge"
	import type { NotificationCenter } from "../notifications/notification-center.svelte"
	import { getCreatorNavigation } from "../shared/creator-navigation"
	import { CursorPagedList } from "../shared/cursor-paged-list.svelte"
	import { errorMessage } from "../shared/errors"
	import Icon from "../shared/Icon.svelte"
	import { loadMoreWhenVisible } from "../shared/load-more-when-visible"
	import ProfileImage from "../shared/ProfileImage.svelte"
	import RefreshIcon from "../shared/RefreshIcon.svelte"
	import Select from "../shared/Select.svelte"
	import SkeletonList from "../shared/SkeletonList.svelte"
	import SkeletonMediaRow from "../shared/SkeletonMediaRow.svelte"
	import ConfirmationDialog from "../shared/ConfirmationDialog.svelte"
	import GroupSubpageHeader from "./GroupSubpageHeader.svelte"
	import type { GroupsStore } from "./groups-store.svelte"

	let {
		store,
		notifications,
	}: {
		store: GroupsStore
		notifications: NotificationCenter
	} = $props()

	const profile = untrack(() => store.profile!),
		group = profile.details.group,
		membership = profile.membership,
		roles = profile.roles ?? [],
		navigation = getCreatorNavigation(),
		exactNumbers = new Intl.NumberFormat("en-US"),
		allRolesValue = 0

	let roleFilterId = $state(allRolesValue),
		members = $state.raw(untrack(() => membersOf(null))),
		imageUrlByUserId = $state.raw(new Map<number, string>()),
		changingUserId = $state(0),
		memberError = $state(""),
		removedMember = $state<GroupMember | null>(null),
		isRemoving = $state(false),
		removeError = $state(""),
		memberListElement = $state<HTMLElement | undefined>(undefined)

	const roleFilterOptions = $derived([
			{ value: allRolesValue, label: "All roles" },
			...[...roles].reverse().map((role) => ({
				value: role.roleId,
				label: `${role.name} (${exactNumbers.format(role.memberCount)})`,
			})),
		]),
		assignableRoles = $derived(
			roles.filter((role) => role.rank < membership.role.rank && role.rank > 0),
		)

	function membersOf(
		role: GroupRole | null,
	): CursorPagedList<GroupMemberPage, GroupMember> {
		return new CursorPagedList<GroupMemberPage, GroupMember>(
			store.scope!,
			(cursor) =>
				accountBackend.GetGroupMembers(
					store.accountId,
					group.groupId,
					role,
					cursor,
				),
			(page) => page.members,
			"Could not load the members. Try again.",
		)
	}

	// canManage tells whether the account may change or remove the member.
	// Roblox allows it only for the members ranked below the account, which
	// leaves out the account itself.
	function canManage(member: GroupMember): boolean {
		return member.role.rank < membership.role.rank
	}

	function showRole(roleId: number): void {
		roleFilterId = roleId
		members = membersOf(roles.find((role) => role.roleId === roleId) ?? null)
		memberListElement?.scrollTo({ top: 0 })
		void members.refresh()
	}

	async function loadImages(loadedMembers: GroupMember[]): Promise<void> {
		const userIds = loadedMembers
			.map((member) => member.user.userId)
			.filter((userId) => !imageUrlByUserId.has(userId))
		if (userIds.length === 0 || !store.scope) return
		try {
			const headshots = await store.scope.track(
				accountBackend.GetAvatarHeadshots(userIds),
			)
			const nextImageUrlByUserId = new Map(imageUrlByUserId)
			for (const headshot of headshots ?? []) {
				nextImageUrlByUserId.set(headshot.robloxUserId, headshot.imageUrl)
			}
			imageUrlByUserId = nextImageUrlByUserId
		} catch {
			// The rows show the default image, and the backend logs the failure.
		}
	}

	async function changeRole(member: GroupMember, roleId: number): Promise<void> {
		const role = roles.find((listedRole) => listedRole.roleId === roleId)
		if (!role || !store.scope || roleId === member.role.roleId || changingUserId)
			return
		changingUserId = member.user.userId
		memberError = ""
		try {
			await store.scope.track(
				accountBackend.SetGroupMemberRole(
					store.accountId,
					group.groupId,
					member.user.userId,
					roleId,
				),
			)
			members.changeItems((listedMembers) =>
				listedMembers.map((listedMember) =>
					listedMember.user.userId === member.user.userId
						? { ...listedMember, role }
						: listedMember,
				),
			)
			notifications.show({
				title: "Role changed",
				tone: "success",
				message: `@${member.user.username} is now ${role.name}.`,
			})
		} catch (error) {
			memberError = errorMessage(error, "Could not change the role. Try again.")
		} finally {
			changingUserId = 0
		}
	}

	async function removeMember(): Promise<void> {
		const member = removedMember
		if (!member || !store.scope || isRemoving) return
		isRemoving = true
		removeError = ""
		try {
			await store.scope.track(
				accountBackend.RemoveGroupMember(
					store.accountId,
					group.groupId,
					member.user.userId,
				),
			)
			members.changeItems((listedMembers) =>
				listedMembers.filter(
					(listedMember) => listedMember.user.userId !== member.user.userId,
				),
			)
			removedMember = null
			notifications.show({
				title: "Member removed",
				tone: "success",
				message: `@${member.user.username} is no longer in ${group.name}.`,
			})
		} catch (error) {
			removeError = errorMessage(error, "Could not remove the member. Try again.")
		} finally {
			isRemoving = false
		}
	}

	$effect(() => {
		const loadedMembers = members.items
		untrack(() => void loadImages(loadedMembers))
	})

	onMount(() => void members.refresh())
</script>

<section class="settings-page min-w-0" aria-labelledby="group-members-title">
	<GroupSubpageHeader
		titleId="group-members-title"
		title="Members"
		groupName={group.name}
		onBack={() => (store.subpage = null)}>
		<button
			class="button-ghost"
			type="button"
			aria-label={members.isLoading ? "Refreshing members…" : "Refresh members"}
			data-tooltip="Refresh members"
			data-tooltip-side="bottom-end"
			aria-busy={members.isLoading}
			disabled={members.isLoading || members.isLoadingMore}
			onclick={() => void members.refresh()}>
			<RefreshIcon isRefreshing={members.isLoading && members.hasLoaded} />
		</button>
	</GroupSubpageHeader>
	<div
		class="flex min-h-10 shrink-0 items-center gap-2 border-b border-line-subtle px-3 py-1.5"
		role="toolbar"
		aria-label="Filter members">
		<Select
			label="Role"
			value={roleFilterId}
			options={roleFilterOptions}
			className="w-56"
			disabled={roles.length === 0}
			onChange={showRole} />
		<span class="text-2xs text-fg-muted"
			>{exactNumbers.format(group.memberCount)} members in total</span>
	</div>
	<div
		class="min-h-0 flex-1 scrollbar-stable overflow-y-auto"
		data-load-more-root
		bind:this={memberListElement}>
		{#if memberError}
			<p class="mx-3 mt-3 notice-error" role="alert">{memberError}</p>
		{/if}
		{#if members.error && members.hasLoaded}
			<p class="mx-3 mt-3 notice-error" role="alert">{members.error}</p>
		{/if}
		{#if !members.hasLoaded && members.error}
			<div class="empty-state h-full" role="alert">
				<Icon svg={warningIcon} size={28} aria-hidden="true" />
				<h2>Members unavailable</h2>
				<p>{members.error}</p>
				<button
					class="button"
					type="button"
					onclick={() => void members.refresh()}>Try again</button>
			</div>
		{:else if !members.hasLoaded}
			<SkeletonList label="Loading members…">
				{#snippet row(index)}<SkeletonMediaRow {index} />{/snippet}
			</SkeletonList>
		{:else if members.items.length === 0}
			<div class="empty-state h-full" role="status">
				<Icon svg={usersThreeIcon} size={28} aria-hidden="true" />
				<h2>No members</h2>
				<p>Nobody holds this role. Select another role.</p>
				<button
					class="button"
					type="button"
					onclick={() => showRole(allRolesValue)}>Show all roles</button>
			</div>
		{:else}
			<ul class="animate-enter" aria-label="Members">
				{#each members.items as member (member.user.userId)}
					{@const isManaged = canManage(member)}
					<li
						class="flex h-12 items-center gap-2 border-b border-line-subtle px-3 transition-colors duration-100 hover:bg-hover">
						<button
							class="flex min-w-0 flex-1 items-center gap-2 rounded-md text-left"
							type="button"
							data-tooltip="Show in Social"
							data-tooltip-strong={member.user.displayName ||
								member.user.username}
							onclick={() => navigation.showUser(member.user)}>
							<span
								class="grid size-9 shrink-0 place-items-center overflow-hidden rounded-md border border-line-strong bg-raised"
								aria-hidden="true">
								<ProfileImage
									url={imageUrlByUserId.get(member.user.userId) ??
										""} />
							</span>
							<span class="flex min-w-0 flex-col leading-snug">
								<strong
									class="inline-flex min-w-0 items-center gap-1 text-xs font-semibold text-fg"
									><span class="truncate"
										>{member.user.displayName ||
											member.user.username}</span
									>{#if member.user.hasVerifiedBadge}<Icon
											svg={sealCheckIcon}
											class="shrink-0 text-fg-muted"
											size={12}
											aria-label="Verified user" />{/if}</strong>
								<span class="truncate text-2xs text-fg-muted"
									>@{member.user.username}</span>
							</span>
						</button>
						{#if isManaged && membership.permissions.canChangeRank && assignableRoles.length > 0}
							<Select
								label={`Role of @${member.user.username}`}
								value={member.role.roleId}
								options={[...assignableRoles].reverse().map((role) => ({
									value: role.roleId,
									label: role.name,
								}))}
								className="w-40"
								disabled={changingUserId !== 0}
								onChange={(roleId) =>
									void changeRole(member, roleId)} />
							{#if changingUserId === member.user.userId}
								<Icon
									svg={circleNotchIcon}
									class="shrink-0 spinner text-fg-muted"
									size={16}
									aria-label="Changing the role…" />
							{/if}
						{:else}
							<span class="shrink-0 text-2xs text-fg-muted"
								>{member.role.name} · Rank {member.role.rank}</span>
						{/if}
						{#if isManaged && membership.permissions.canRemoveMembers}
							<button
								class="button-ghost shrink-0"
								type="button"
								aria-label={`Remove @${member.user.username}`}
								data-tooltip="Remove from group"
								data-tooltip-side="bottom-end"
								onclick={() => {
									removeError = ""
									removedMember = member
								}}>
								<Icon
									svg={userMinusIcon}
									size={16}
									aria-hidden="true" />
							</button>
						{/if}
					</li>
				{/each}
			</ul>
			{#if members.nextCursor}
				<div
					class="flex min-h-12 items-center justify-center text-fg-faint"
					use:loadMoreWhenVisible={{
						pageKey: members.items.length,
						loadMore: () => {
							if (!members.error) void members.loadMore()
						},
					}}>
					{#if members.error}
						<button
							class="button"
							type="button"
							onclick={() => void members.loadMore()}
							>Load more members</button>
					{:else}
						<Icon
							svg={circleNotchIcon}
							class="spinner"
							size={16}
							aria-label="Loading more members…" />
					{/if}
				</div>
			{/if}
		{/if}
	</div>
</section>

{#if removedMember}
	<ConfirmationDialog
		title="Remove member?"
		description={`@${removedMember.user.username} will be removed from ${group.name} and lose the ${removedMember.role.name} role. They can ask to join again.`}
		confirmLabel="Remove member"
		progressLabel="Removing…"
		isWorking={isRemoving}
		error={removeError}
		onConfirm={() => void removeMember()}
		onCancel={() => (removedMember = null)} />
{/if}

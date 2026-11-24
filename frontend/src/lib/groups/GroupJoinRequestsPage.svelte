<script lang="ts">
	import { notifyErrors } from "../notifications/error-notifications.svelte"
	import LoadMore from "../shared/LoadMore.svelte"
	import checkIcon from "@phosphor-icons/core/regular/check.svg?raw"
	import circleNotchIcon from "@phosphor-icons/core/regular/circle-notch.svg?raw"
	import userPlusIcon from "@phosphor-icons/core/regular/user-plus.svg?raw"
	import warningIcon from "@phosphor-icons/core/regular/warning.svg?raw"
	import xIcon from "@phosphor-icons/core/regular/x.svg?raw"
	import { onMount, untrack } from "svelte"
	import { accountBackend } from "../backend/bridge"
	import type { GroupJoinRequest, GroupJoinRequestPage } from "../backend/bridge"
	import type { NotificationCenter } from "../notifications/notification-center.svelte"
	import { getCreatorNavigation } from "../shared/creator-navigation"
	import { CursorPagedList } from "../shared/cursor-paged-list.svelte"
	import { errorMessage } from "../shared/errors"
	import Icon from "../shared/Icon.svelte"
	import RefreshIcon from "../shared/RefreshIcon.svelte"
	import SkeletonList from "../shared/SkeletonList.svelte"
	import SkeletonMediaRow from "../shared/SkeletonMediaRow.svelte"
	import Timestamp from "../shared/Timestamp.svelte"
	import { relativeTime } from "../shared/timestamp"
	import ProfilePageHeader from "../shared/ProfilePageHeader.svelte"
	import type { GroupsStore } from "./groups-store.svelte"

	let {
		store,
		notifications,
	}: {
		store: GroupsStore
		notifications: NotificationCenter
	} = $props()

	const group = untrack(() => store.profile!.details.group),
		navigation = getCreatorNavigation(),
		requests = untrack(
			() =>
				new CursorPagedList<GroupJoinRequestPage, GroupJoinRequest>(
					store.scope!,
					(cursor) =>
						accountBackend.GetGroupJoinRequests(
							store.accountId,
							group.groupId,
							cursor,
						),
					(page) => page.requests,
					"Could not load the join requests. Try again.",
				),
		)
	let answeringUserId = $state(0),
		answerError = $state("")

	async function answer(
		request: GroupJoinRequest,
		isAccepted: boolean,
	): Promise<void> {
		if (!store.scope || answeringUserId) return
		answeringUserId = request.user.userId
		answerError = ""
		try {
			await store.scope.track(
				accountBackend.AnswerGroupJoinRequest(
					store.accountId,
					group.groupId,
					request.user.userId,
					isAccepted,
				),
			)
			requests.changeItems((listedRequests) =>
				listedRequests.filter(
					(listedRequest) =>
						listedRequest.user.userId !== request.user.userId,
				),
			)
			notifications.show({
				title: isAccepted ? "Request accepted" : "Request declined",
				tone: "success",
				message: isAccepted
					? `@${request.user.username} joined ${group.name}.`
					: `@${request.user.username} did not join ${group.name}.`,
			})
		} catch (error) {
			answerError = errorMessage(
				error,
				isAccepted
					? "Could not accept the request. Try again."
					: "Could not decline the request. Try again.",
			)
		} finally {
			answeringUserId = 0
		}
	}

	onMount(() => void requests.refresh())

	notifyErrors("Join requests not loaded", () => requests.error)
	notifyErrors("Join request not answered", () => answerError)
</script>

<section class="settings-page min-w-0" aria-labelledby="group-requests-title">
	<ProfilePageHeader
		titleId="group-requests-title"
		title="Join requests"
		subtitle={group.name}>
		<button
			class="button-ghost"
			type="button"
			aria-label={requests.isLoading
				? "Refreshing join requests…"
				: "Refresh join requests"}
			data-tooltip="Refresh join requests"
			data-tooltip-side="bottom-end"
			aria-busy={requests.isLoading}
			disabled={requests.isLoading || requests.isLoadingMore}
			onclick={() => void requests.refresh()}>
			<RefreshIcon isRefreshing={requests.isLoading && requests.hasLoaded} />
		</button>
	</ProfilePageHeader>
	<div class="min-h-0 flex-1 scrollbar-stable overflow-y-auto" data-load-more-root>
		{#if !requests.hasLoaded && requests.error}
			<div class="empty-state h-full">
				<Icon svg={warningIcon} size={28} aria-hidden="true" />
				<h2>Join requests unavailable</h2>
				<button
					class="button"
					type="button"
					onclick={() => void requests.refresh()}>Try again</button>
			</div>
		{:else if !requests.hasLoaded}
			<SkeletonList label="Loading join requests…">
				{#snippet row(index)}<SkeletonMediaRow {index} />{/snippet}
			</SkeletonList>
		{:else if requests.items.length === 0}
			<div class="empty-state h-full" role="status">
				<Icon svg={userPlusIcon} size={28} aria-hidden="true" />
				<h2>No join requests</h2>
				<p>Nobody is waiting to join {group.name}.</p>
			</div>
		{:else}
			<ul class="animate-enter" aria-label="Join requests">
				{#each requests.items as request (request.user.userId)}
					{@const isAnswering = answeringUserId === request.user.userId}
					<li
						class="flex h-12 animate-enter items-center gap-2 border-b border-line-subtle px-3 transition-colors duration-100 hover:bg-hover">
						<button
							class="flex min-w-0 flex-1 flex-col rounded-md text-left leading-snug"
							type="button"
							data-tooltip="Show in Social"
							data-tooltip-strong={request.user.displayName ||
								request.user.username}
							onclick={() => navigation.showUser(request.user)}>
							<strong class="truncate text-xs font-semibold text-fg"
								>{request.user.displayName ||
									request.user.username}</strong>
							<span class="truncate text-2xs text-fg-muted"
								>@{request.user.username}{#if request.requestedAtMs}
									· <Timestamp value={request.requestedAtMs} /> · {relativeTime(
										request.requestedAtMs,
									)}{/if}</span>
						</button>
						{#if isAnswering}
							<Icon
								svg={circleNotchIcon}
								class="shrink-0 spinner text-fg-muted"
								size={16}
								aria-label="Answering the request…" />
						{/if}
						<button
							class="button shrink-0"
							type="button"
							disabled={answeringUserId !== 0}
							onclick={() => void answer(request, false)}>
							<Icon svg={xIcon} size={16} aria-hidden="true" />Decline
						</button>
						<button
							class="button shrink-0"
							type="button"
							disabled={answeringUserId !== 0}
							onclick={() => void answer(request, true)}>
							<Icon svg={checkIcon} size={16} aria-hidden="true" />Accept
						</button>
					</li>
				{/each}
			</ul>
			{#if requests.nextCursor}
				<LoadMore
					pageKey={requests.items.length}
					label="Load more requests"
					loadingLabel="Loading more requests…"
					isLoadingMore={requests.isLoadingMore}
					hasFailed={!!requests.error}
					onLoadMore={() => void requests.loadMore()} />
			{/if}
		{/if}
	</div>
</section>

<script lang="ts">
	import calendarDotsIcon from "@phosphor-icons/core/regular/calendar-dots.svg?raw"
	import prohibitIcon from "@phosphor-icons/core/regular/prohibit.svg?raw"
	import sealCheckIcon from "@phosphor-icons/core/regular/seal-check.svg?raw"
	import { accountBackend, SocialProfileSection } from "../backend/bridge"
	import type { SocialProfileSnapshot, SocialUser, UserView } from "../backend/bridge"
	import circleNotchIcon from "@phosphor-icons/core/regular/circle-notch.svg?raw"
	import AvatarPortraitFrame from "../avatar/AvatarPortraitFrame.svelte"
	import { AvatarPortraitLoad } from "../avatar/avatar-portrait.svelte"
	import { loadCreatorImage } from "../shared/creator-images"
	import Icon from "../shared/Icon.svelte"
	import PrimaryGroupFact from "../shared/PrimaryGroupFact.svelte"
	import ProfilePreviewLayout from "../shared/ProfilePreviewLayout.svelte"
	import type { ProfilePreviewFrame } from "../shared/ProfilePreviewLayout.svelte"
	import ProfileStats from "../shared/ProfileStats.svelte"
	import { SectionLoad } from "../shared/section-load.svelte"
	import Timestamp from "../shared/Timestamp.svelte"
	import { relativeTime } from "../shared/timestamp"
	import { relationshipCountSection, relationshipStats } from "./relationship-stats"
	import {
		relationshipActions,
		relationshipButtonClass,
		type RelationshipActionId,
	} from "./relationship"
	import { emptySocialProfile, socialProfileSections } from "./social-store.svelte"

	// UserPreview shows the information of a Roblox user as the account sees
	// it: the avatar, the relationship counts, the description, and when the
	// user joined. It sends, accepts, declines, and cancels friend requests.
	let {
		accountId,
		user,
		frame,
		onRelationshipAction,
	}: {
		accountId: number
		user: { userId: number; username: string; displayName: string }
		frame: ProfilePreviewFrame
		onRelationshipAction: (
			person: SocialUser,
			actionId: RelationshipActionId,
		) => Promise<boolean>
	} = $props()

	// The preview reads only the sections that it shows.
	const previewSections = [
			SocialProfileSection.SocialSectionRelationship,
			SocialProfileSection.SocialSectionAbout,
			SocialProfileSection.SocialSectionFriends,
			SocialProfileSection.SocialSectionFollowers,
			SocialProfileSection.SocialSectionFollowing,
			SocialProfileSection.SocialSectionPrimaryGroup,
		],
		sectionClass = "text-xs font-semibold text-fg-muted",
		// A preview answers or sends a friend request. Ending a friendship
		// and the other connections stay on the full profile.
		quickActionIds = new Set<RelationshipActionId>([
			"add-friend",
			"accept-friend",
			"decline-friend",
			"cancel-friend",
		])

	const sections = new SectionLoad<SocialProfileSnapshot, SocialProfileSection>(
			socialProfileSections.filter(({ section }) =>
				previewSections.includes(section),
			),
			() => emptySocialProfile(user.userId),
			"Could not load this profile. Try again.",
		),
		avatarPortrait = new AvatarPortraitLoad()

	let userView = $state.raw<UserView | null>(null),
		headshotUrl = $state(""),
		pendingActionId = $state<RelationshipActionId | null>(null),
		loadCount = $state(0)

	// An action reads the sections again, so that the actions follow the new
	// relationship.
	$effect(() => {
		const userId = user.userId
		let isCurrent = true
		void loadCount
		void sections.load(
			(section) => accountBackend.GetSocialProfile(accountId, userId, section),
			() => isCurrent,
		)
		return () => {
			isCurrent = false
			sections.cancel()
		}
	})

	$effect(() => {
		const userId = user.userId
		let isCurrent = true
		const userRequest = accountBackend.GetRobloxUser(userId)
		void (async () => {
			try {
				const loadedUser = await userRequest
				if (isCurrent) userView = loadedUser
			} catch {
				// The backend logs the failure. The preview keeps the names
				// that the page that opened it gave.
			}
		})()
		void loadCreatorImage("User", userId).then((loadedUrl) => {
			if (isCurrent) headshotUrl = loadedUrl
			return loadedUrl
		})
		const cancelPortrait = avatarPortrait.load(accountId, userId)
		return () => {
			isCurrent = false
			void userRequest.cancel()
			cancelPortrait()
		}
	})

	const profile = $derived(sections.snapshot),
		username = $derived(userView?.username || user.username),
		displayName = $derived(
			userView?.displayName ||
				user.displayName ||
				username ||
				`User ${user.userId}`,
		),
		isVerified = $derived(userView?.hasVerifiedBadge || profile.verified),
		actions = $derived(
			sections.hasSection(SocialProfileSection.SocialSectionRelationship)
				? relationshipActions(
						profile.relationship,
						profile.unavailable ?? [],
					).filter((action) => quickActionIds.has(action.id))
				: [],
		),
		person = $derived<SocialUser>({
			userId: user.userId,
			username,
			displayName,
			nickname: profile.relationship.nickname,
			imageUrl: headshotUrl,
			presence: null,
			request: null,
			hasVerifiedBadge: isVerified,
			trusted: false,
		})

	// runAction runs a relationship action, then reads the profile again so
	// that its actions follow the new relationship, and returns whether it
	// succeeded.
	async function runAction(actionId: RelationshipActionId): Promise<boolean> {
		if (pendingActionId) return false
		pendingActionId = actionId
		try {
			const isDone = await onRelationshipAction(person, actionId)
			if (isDone) loadCount++
			return isDone
		} finally {
			pendingActionId = null
		}
	}

	function startAction(actionId: RelationshipActionId): void {
		const action = actions.find((listedAction) => listedAction.id === actionId)
		if (!action) return
		if (action.tone === "neutral") {
			void runAction(actionId)
			return
		}
		frame.confirm({
			title: action.question(displayName),
			description: action.consequence,
			confirmLabel: action.label,
			progressLabel: action.progress,
			tone: action.tone,
			run: () => runAction(actionId),
		})
	}
</script>

{#snippet quickActions()}
	{#each actions as action (action.id)}
		<button
			class={[relationshipButtonClass(action.tone), "animate-enter"]}
			type="button"
			aria-busy={pendingActionId === action.id}
			disabled={pendingActionId !== null}
			onclick={() => startAction(action.id)}>
			<Icon
				svg={pendingActionId === action.id ? circleNotchIcon : action.icon}
				class={pendingActionId === action.id ? "spinner" : undefined}
				size={14}
				aria-hidden="true" />
			{pendingActionId === action.id ? action.progress : action.label}
		</button>
	{/each}
{/snippet}

<ProfilePreviewLayout {frame} quickActions={actions.length ? quickActions : undefined}>
	{#snippet image()}
		<AvatarPortraitFrame
			portraitLoad={avatarPortrait}
			{headshotUrl}
			label={`@${username || user.userId}'s avatar`} />
	{/snippet}
	{#snippet title()}
		<span class="select-text">{displayName}</span>{#if isVerified}<Icon
				svg={sealCheckIcon}
				class="relative top-px ml-1 inline-block text-accent-strong"
				size={16}
				aria-label="Verified" />{/if}
	{/snippet}
	{#snippet summary()}
		<div
			class="flex min-w-0 flex-wrap items-center gap-1 font-mono text-2xs wrap-anywhere text-fg-muted">
			{#if username}<span class="select-text">@{username}</span>
				<span class="text-fg-faint" aria-hidden="true">·</span>{/if}
			<span class="select-text">ID: {user.userId}</span>
		</div>
		<ProfileStats
			label="Relationship counts"
			stats={relationshipStats(
				{
					friends: profile.friendCount,
					followers: profile.followerCount,
					following: profile.followingCount,
				},
				(list) => sections.isSectionPending(relationshipCountSection(list)),
			)} />
		{#if profile.banned}
			<ul class="mt-1 flex flex-wrap gap-1" aria-label="Status">
				<li class="tag-pill-warning">
					<Icon svg={prohibitIcon} size={12} aria-hidden="true" />
					<span>Banned</span>
				</li>
			</ul>
		{/if}
	{/snippet}

	{#if profile.description}
		<section class="flex flex-col gap-3" aria-label="About">
			<h3 class={sectionClass}>About</h3>
			<p
				class="max-h-40 text-block overflow-y-auto select-text"
				data-variant="prose">
				{profile.description}
			</p>
		</section>
	{/if}
	{#if profile.createdAtMs || profile.primaryGroup}
		<section class="flex flex-col gap-3" aria-label="Details">
			<h3 class={sectionClass}>Details</h3>
			<dl class="facts">
				{#if profile.createdAtMs}
					<div>
						<dt>
							<Icon
								svg={calendarDotsIcon}
								size={16}
								aria-hidden="true" />Joined
						</dt>
						<dd>
							<Timestamp value={profile.createdAtMs} />
							<small
								>{relativeTime(profile.createdAtMs, Date.now(), false)} old</small>
						</dd>
					</div>
				{/if}
				{#if profile.primaryGroup}
					<PrimaryGroupFact group={profile.primaryGroup} />
				{/if}
			</dl>
		</section>
	{:else if sections.isLoading}
		<span class="h-16 w-full skeleton rounded-lg" role="status"
			><span class="sr-only">Loading profile…</span></span>
	{:else if sections.error}
		<p class="text-xs text-fg-muted">{sections.error}</p>
	{/if}
</ProfilePreviewLayout>

<script lang="ts">
	import arrowSquareOutIcon from "@phosphor-icons/core/regular/arrow-square-out.svg?raw"
	import arrowsClockwiseIcon from "@phosphor-icons/core/regular/arrows-clockwise.svg?raw"
	import backpackIcon from "@phosphor-icons/core/regular/backpack.svg?raw"
	import calendarDotsIcon from "@phosphor-icons/core/regular/calendar-dots.svg?raw"
	import circleNotchIcon from "@phosphor-icons/core/regular/circle-notch.svg?raw"
	import clockCounterClockwiseIcon from "@phosphor-icons/core/regular/clock-counter-clockwise.svg?raw"
	import clockIcon from "@phosphor-icons/core/regular/clock.svg?raw"
	import dotsThreeIcon from "@phosphor-icons/core/regular/dots-three.svg?raw"
	import linkIcon from "@phosphor-icons/core/regular/link.svg?raw"
	import medalIcon from "@phosphor-icons/core/regular/medal.svg?raw"
	import pencilSimpleIcon from "@phosphor-icons/core/regular/pencil-simple.svg?raw"
	import sealCheckIcon from "@phosphor-icons/core/regular/seal-check.svg?raw"
	import signpostIcon from "@phosphor-icons/core/regular/signpost.svg?raw"
	import userCircleMinusIcon from "@phosphor-icons/core/regular/user-circle-minus.svg?raw"
	import userCirclePlusIcon from "@phosphor-icons/core/regular/user-circle-plus.svg?raw"
	import usersThreeIcon from "@phosphor-icons/core/regular/users-three.svg?raw"
	import Icon from "../shared/Icon.svelte"
	import { FriendStatus } from "../backend/bridge"
	import type { SocialUser } from "../backend/bridge"
	import type { AccountStore } from "../accounts/account-store.svelte"
	import AccountPresence from "../accounts/AccountPresence.svelte"
	import Timestamp from "../shared/Timestamp.svelte"
	import { relativeTime } from "../shared/timestamp"
	import FriendConnections from "./FriendConnections.svelte"
	import SocialBadges from "./SocialBadges.svelte"
	import SocialAvatar from "./SocialAvatar.svelte"
	import SocialCounts from "./SocialCounts.svelte"
	import { personName } from "./social-store.svelte"
	import type { SocialStore } from "./social-store.svelte"

	let {
		store,
		accounts,
		person,
		onFillLaunch,
		onSearchPlace,
		onEditNickname,
	}: {
		store: SocialStore
		accounts: AccountStore
		person: SocialUser
		onFillLaunch: (placeId: number, jobId: string) => void
		onSearchPlace: (placeId: number) => void
		onEditNickname: (nickname: string) => void
	} = $props()

	const origins: Record<string, string> = {
			FriendRecommendations: "Friend recommendations",
			InGame: "An experience",
			PhoneContactImporter: "Phone contacts",
			PlayerSearch: "Player search",
			QrCode: "QR code",
			UserProfile: "Profile page",
		},
		numbers = new Intl.NumberFormat("en-US", {
			notation: "compact",
			maximumFractionDigits: 1,
		})

	const request = $derived(person.request),
		profile = $derived(
			store.profile?.userId === person.userId ? store.profile : null,
		),
		responding = $derived(
			store.responding?.userId === person.userId ? store.responding : null,
		),
		origin = $derived(request ? (origins[request.origin] ?? "") : ""),
		nickname = $derived(profile ? profile.relationship.nickname : person.nickname),
		canNickname = $derived(
			profile?.relationship.friend === FriendStatus.FriendStatusFriends,
		),
		hasFacts = $derived(
			!!request ||
				!!profile?.createdAtMs ||
				!!profile?.previousNames?.length ||
				!!profile?.primaryGroup ||
				!!profile?.robloxBadges?.length ||
				(profile?.inventoryVisible ?? null) !== null,
		)
</script>

<section class="flex flex-col gap-4 p-3 pb-4" aria-labelledby="social-name">
	<div class="flex items-start gap-3 @max-xl:flex-col">
		<div class="flex min-w-0 flex-1 items-center gap-4">
			<SocialAvatar imageUrl={person.imageUrl} className="size-20" />
			<div class="flex min-w-0 flex-col gap-1">
				<div class="min-w-0 leading-6">
					<h2
						id="social-name"
						class="mr-1 inline text-lg font-semibold tracking-tight wrap-anywhere text-fg">
						{person.displayName ||
							person.username ||
							`User ${person.userId}`}{#if profile?.verified}<Icon
								svg={sealCheckIcon}
								class="relative top-px ml-1 inline-block text-accent-strong"
								size={16}
								aria-label="Verified" />{/if}
					</h2>
					{#if nickname || canNickname}
						<span
							class="inline-flex max-w-full min-w-0 items-center gap-0.5 align-top text-sm text-fg-muted">
							{#if nickname}<span aria-hidden="true">(</span><span
									class="truncate"
									><span class="sr-only"
										>Nickname:
									</span>{nickname}</span
								>{/if}
							{#if canNickname}<button
									class="btn-ghost-sm"
									type="button"
									aria-label={nickname
										? "Edit nickname"
										: "Add a nickname"}
									data-tooltip={nickname
										? "Edit nickname on Roblox"
										: "Add a nickname on Roblox"}
									disabled={store.updating !== null}
									onclick={() => onEditNickname(nickname)}
									><Icon
										svg={pencilSimpleIcon}
										size={14}
										aria-hidden="true" /></button
								>{/if}
							{#if nickname}<span aria-hidden="true">)</span>{/if}
						</span>
					{/if}
				</div>
				<div
					class="flex min-w-0 flex-wrap items-center gap-1 font-mono text-2xs wrap-anywhere text-fg-muted">
					{#if person.username}<span>@{person.username}</span>
						<span class="text-fg-faint" aria-hidden="true">-</span>{/if}
					<span>ID: {person.userId}</span>
				</div>
				{#if person.presence}
					<AccountPresence
						presence={person.presence}
						onJoin={onFillLaunch}
						{onSearchPlace} />
				{/if}
				{#if profile}<SocialBadges {profile} />{/if}
			</div>
		</div>
		<div class="flex items-center gap-1 @max-xl:self-end">
			{#if profile && !profile.relationship.self && !profile.unavailable?.includes("follow")}
				{@const following = profile.relationship.following}
				<button
					class="btn"
					type="button"
					disabled={store.updating !== null}
					onclick={() => void store.setFollowing(person, !following)}>
					<Icon
						svg={following ? userCircleMinusIcon : userCirclePlusIcon}
						size={14}
						aria-hidden="true" />
					{#if store.updating === "follow"}{following
							? "Unfollowing…"
							: "Following…"}{:else}{following
							? "Unfollow"
							: "Follow"}{/if}
				</button>
			{/if}
			{#if request}
				<button
					class="btn-primary"
					type="button"
					disabled={store.responding !== null}
					onclick={() => void store.respond(person, true)}>
					{responding?.accept ? "Accepting…" : "Accept"}
				</button>
				<button
					class="btn"
					type="button"
					disabled={store.responding !== null}
					onclick={() => void store.respond(person, false)}
					>{responding && !responding.accept
						? "Declining…"
						: "Decline"}</button>
			{/if}
			<button
				class="btn-icon"
				type="button"
				data-tooltip="Refresh profile"
				aria-label={store.loadingProfile
					? "Refreshing profile"
					: "Refresh profile"}
				aria-busy={store.loadingProfile}
				disabled={store.loadingProfile}
				onclick={() => void store.loadProfile(person.userId, true)}>
				{#if store.loadingProfile}
					<Icon
						svg={circleNotchIcon}
						class="spinner"
						size={16}
						aria-hidden="true" />
				{:else}
					<Icon
						svg={arrowsClockwiseIcon}
						data-turn
						size={16}
						aria-hidden="true" />
				{/if}
			</button>
			<button
				class="btn-icon"
				type="button"
				aria-label={`More actions for ${personName(person)}`}
				aria-haspopup="menu"
				aria-expanded={store.menu?.person.userId === person.userId}
				data-open={store.menu?.person.userId === person.userId || undefined}
				data-social-menu
				data-tooltip="Show more actions"
				data-tooltip-side="bottom-end"
				onclick={(event) => {
					if (store.menu) store.closeMenu()
					else store.openMenu(event, person)
				}}>
				<Icon svg={dotsThreeIcon} size={16} aria-hidden="true" />
			</button>
		</div>
	</div>

	{#if profile}
		<SocialCounts
			friends={profile.friendCount}
			followers={profile.followerCount}
			following={profile.followingCount}
			onShow={(list) => store.explore(person, list)} />
	{/if}

	{#if profile && !profile.unavailable?.includes("about")}
		<section class="flex flex-col gap-2" aria-labelledby="social-about-heading">
			<h3 id="social-about-heading" class="text-xs font-semibold text-fg-muted">
				About
			</h3>
			<pre class="text-block">{profile.description || "No description."}</pre>
			{#if profile.socialLinks?.length}
				<div class="flex flex-wrap items-center gap-1">
					<Icon
						svg={linkIcon}
						class="mr-1 text-fg-faint"
						size={16}
						aria-label="Social links" />
					{#each profile.socialLinks as link (link.name)}
						<button
							class="btn"
							type="button"
							aria-label={`Open ${link.name} (${link.url})`}
							data-tooltip={link.url}
							onclick={() => void store.openLink(link.url)}>
							<Icon
								svg={arrowSquareOutIcon}
								size={16}
								aria-hidden="true" />
							{link.name}
						</button>
					{/each}
				</div>
			{/if}
		</section>
	{/if}

	{#if profile && !profile.unavailable?.includes("connections") && accounts.accountTotal > 1}
		<FriendConnections connections={profile.connections ?? []} {accounts} />
	{/if}

	{#if hasFacts}
		<section class="flex flex-col gap-2" aria-labelledby="social-details-heading">
			<h3 id="social-details-heading" class="text-xs font-semibold text-fg-muted">
				Details
			</h3>
			<dl class="facts">
				{#if profile?.createdAtMs}
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
				{#if request?.sentAtMs}
					<div>
						<dt>
							<Icon svg={clockIcon} size={16} aria-hidden="true" />Request
							sent
						</dt>
						<dd>
							<Timestamp value={request.sentAtMs} />
							<small>{relativeTime(request.sentAtMs)}</small>
						</dd>
					</div>
				{/if}
				{#if origin}
					<div>
						<dt>
							<Icon svg={signpostIcon} size={16} aria-hidden="true" />Sent
							from
						</dt>
						<dd><span>{origin}</span></dd>
					</div>
				{/if}
				{#if request}
					<div
						data-tone={request.mutualFriends?.length ? undefined : "muted"}>
						<dt>
							<Icon
								svg={usersThreeIcon}
								size={16}
								aria-hidden="true" />Mutual friends
						</dt>
						<dd>
							<span>{request.mutualFriends?.join(", ") || "None"}</span>
						</dd>
					</div>
				{/if}
				{#if profile?.previousNames?.length}
					<div>
						<dt>
							<Icon
								svg={clockCounterClockwiseIcon}
								size={16}
								aria-hidden="true" />Previous names
						</dt>
						<dd><span>{profile.previousNames.join(", ")}</span></dd>
					</div>
				{/if}
				{#if profile?.primaryGroup}
					{@const group = profile.primaryGroup}
					<div>
						<dt>
							<Icon
								svg={usersThreeIcon}
								size={16}
								aria-hidden="true" />Primary group
						</dt>
						<dd>
							<span class="inline-flex min-w-0 items-center gap-1"
								>{group.name}{#if group.verified}<Icon
										svg={sealCheckIcon}
										class="shrink-0 text-fg-muted"
										size={14}
										aria-label="Verified group" />{/if}</span>
							<small
								>{group.role ? `${group.role} - ` : ""}{numbers.format(
									group.members,
								)}
								{group.members === 1 ? "member" : "members"}</small>
						</dd>
					</div>
				{/if}
				{#if profile?.robloxBadges?.length}
					<div>
						<dt>
							<Icon svg={medalIcon} size={16} aria-hidden="true" />Roblox
							badges
						</dt>
						<dd><span>{profile.robloxBadges.join(", ")}</span></dd>
					</div>
				{/if}
				{#if profile && profile.inventoryVisible !== null}
					<div data-tone={profile.inventoryVisible ? undefined : "muted"}>
						<dt>
							<Icon
								svg={backpackIcon}
								size={16}
								aria-hidden="true" />Inventory
						</dt>
						<dd>
							<span
								>{profile.inventoryVisible
									? "Public"
									: "Private"}</span>
						</dd>
					</div>
				{/if}
			</dl>
		</section>
	{/if}

	{#if store.profileError}
		<p class="text-xs text-danger-muted" role="alert">{store.profileError}</p>
	{:else if (profile?.unavailable?.length ?? 0) > 0}
		<p class="text-xs text-danger-muted" role="alert">
			Some profile details could not be loaded. Refresh to retry.
		</p>
	{:else if !profile && store.loadingProfile}
		<div class="flex items-center gap-2 text-xs text-fg-muted" role="status">
			<Icon svg={circleNotchIcon} class="spinner" size={16} aria-hidden="true" />
			Loading profile…
		</div>
	{/if}
</section>

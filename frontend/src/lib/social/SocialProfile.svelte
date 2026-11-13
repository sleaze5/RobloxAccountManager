<script lang="ts">
	import arrowSquareOutIcon from "@phosphor-icons/core/regular/arrow-square-out.svg?raw"
	import backpackIcon from "@phosphor-icons/core/regular/backpack.svg?raw"
	import calendarDotsIcon from "@phosphor-icons/core/regular/calendar-dots.svg?raw"
	import circleNotchIcon from "@phosphor-icons/core/regular/circle-notch.svg?raw"
	import clockCounterClockwiseIcon from "@phosphor-icons/core/regular/clock-counter-clockwise.svg?raw"
	import clockIcon from "@phosphor-icons/core/regular/clock.svg?raw"
	import dotsThreeIcon from "@phosphor-icons/core/regular/dots-three.svg?raw"
	import linkIcon from "@phosphor-icons/core/regular/link.svg?raw"
	import swapIcon from "@phosphor-icons/core/regular/swap.svg?raw"
	import diamondIcon from "@phosphor-icons/core/regular/diamond.svg?raw"
	import usersFourIcon from "@phosphor-icons/core/regular/users-four.svg?raw"
	import medalIcon from "@phosphor-icons/core/regular/medal.svg?raw"
	import pencilSimpleIcon from "@phosphor-icons/core/regular/pencil-simple.svg?raw"
	import sealCheckIcon from "@phosphor-icons/core/regular/seal-check.svg?raw"
	import signpostIcon from "@phosphor-icons/core/regular/signpost.svg?raw"
	import usersThreeIcon from "@phosphor-icons/core/regular/users-three.svg?raw"
	import Icon from "../shared/Icon.svelte"
	import RefreshIcon from "../shared/RefreshIcon.svelte"
	import { FriendStatus, SocialList, SocialProfileSection } from "../backend/bridge"
	import type { SocialUser } from "../backend/bridge"
	import type { AccountStore } from "../accounts/account-store.svelte"
	import AccountPresence from "../accounts/AccountPresence.svelte"
	import MissingFactValue from "../shared/MissingFactValue.svelte"
	import PrimaryGroupFact from "../shared/PrimaryGroupFact.svelte"
	import Skeleton from "../shared/Skeleton.svelte"
	import Timestamp from "../shared/Timestamp.svelte"
	import { relativeTime } from "../shared/timestamp"
	import FriendConnections from "./FriendConnections.svelte"
	import SocialBadges from "./SocialBadges.svelte"
	import SocialAvatar from "./SocialAvatar.svelte"
	import SocialCounts from "./SocialCounts.svelte"
	import {
		pendingRequestActionIds,
		relationshipActions,
		relationshipButtonClass,
		requestRelationship,
	} from "./relationship"
	import type { RelationshipActionId } from "./relationship"
	import { personName } from "./social-store.svelte"
	import type { SocialStore } from "./social-store.svelte"

	let {
		store,
		accounts,
		person,
		onFillLaunch,
		onSearchPlace,
		onSearchUniverse,
		onEditNickname,
		onRelationshipAction,
	}: {
		store: SocialStore
		accounts: AccountStore
		person: SocialUser
		onFillLaunch: (placeId: number, jobId: string) => void
		onSearchPlace: (placeId: number) => void
		onSearchUniverse: (universeId: number) => void
		onEditNickname: (nickname: string) => void
		onRelationshipAction: (person: SocialUser, action: RelationshipActionId) => void
	} = $props()

	const origins: Record<string, string> = {
			FriendRecommendations: "Friend recommendations",
			InGame: "An experience",
			PhoneContactImporter: "Phone contacts",
			PlayerSearch: "Player search",
			QrCode: "QR code",
			UserProfile: "Profile page",
		},
		exactNumbers = new Intl.NumberFormat("en-US"),
		// Roblox mutual trade eligibility statuses from the trades v2 API.
		tradeEligibilityLabels: Record<string, string> = {
			CallingUserIneligible: "This account cannot trade",
			TargetUserIneligible: "They cannot trade",
			CallingUserPrivacySettingsRestricted: "Blocked by privacy settings",
			CallingUserAgeCheckRequired: "Needs an age check",
		},
		numbers = new Intl.NumberFormat("en-US", {
			notation: "compact",
			maximumFractionDigits: 1,
		})

	// The profile rows stay in place while their sections load, so that the
	// pane does not move when a value arrives.
	const request = $derived(person.request),
		profile = $derived(
			store.profile?.userId === person.userId ? store.profile : null,
		),
		sections = $derived(
			store.profileLoad?.snapshot.userId === person.userId
				? store.profileLoad
				: null,
		),
		profileRelationship = $derived(profile ? store.profileRelationship : null),
		relationship = $derived(
			profileRelationship?.relationship ?? requestRelationship(person),
		),
		actions = $derived(
			relationship
				? relationshipActions(
						relationship,
						profileRelationship?.unavailable ?? [],
					).filter((action) => pendingRequestActionIds.has(action.id))
				: [],
		),
		origin = $derived(request ? (origins[request.origin] ?? "") : ""),
		nickname = $derived(
			profileRelationship
				? profileRelationship.relationship.nickname
				: person.nickname,
		),
		canNickname = $derived(
			profileRelationship?.relationship.friend ===
				FriendStatus.FriendStatusFriends,
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
									class="button-ghost-sm"
									type="button"
									aria-label={nickname
										? "Edit nickname"
										: "Add a nickname"}
									data-tooltip={nickname
										? "Edit nickname on Roblox"
										: "Add a nickname on Roblox"}
									disabled={store.savingNickname}
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
						{onSearchPlace}
						{onSearchUniverse} />
				{/if}
				{#if profile}<SocialBadges
						{profile}
						relationship={profileRelationship?.relationship ?? null} />{/if}
			</div>
		</div>
		<div class="flex flex-wrap items-center justify-end gap-1 @max-xl:self-end">
			{#each actions as action (action.id)}
				{@const pending = store.isActionPending(person.userId, action.id)}
				<button
					class={[relationshipButtonClass(action.tone), "animate-enter"]}
					type="button"
					aria-busy={pending}
					disabled={store.pendingAction !== null}
					onclick={() => onRelationshipAction(person, action.id)}>
					{#if pending}<Icon
							svg={circleNotchIcon}
							class="spinner"
							size={14}
							aria-hidden="true" />{:else}<Icon
							svg={action.icon}
							size={14}
							aria-hidden="true" />{/if}
					{pending ? action.progress : action.label}
				</button>
			{/each}
			<button
				class="button-icon"
				type="button"
				data-tooltip="Refresh profile"
				aria-label={store.loadingProfile
					? "Refreshing profile"
					: "Refresh profile"}
				aria-busy={store.loadingProfile}
				disabled={store.loadingProfile}
				onclick={() => void store.loadProfile(person.userId, true)}>
				<RefreshIcon isRefreshing={store.loadingProfile} />
			</button>
			<button
				class="button-icon"
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

	{#if sections}
		<SocialCounts
			friends={profile?.friendCount ?? null}
			followers={profile?.followerCount ?? null}
			following={profile?.followingCount ?? null}
			isCountPending={(list) =>
				sections.isSectionPending(
					list === SocialList.SocialFollowers
						? SocialProfileSection.SocialSectionFollowers
						: list === SocialList.SocialFollowing
							? SocialProfileSection.SocialSectionFollowing
							: SocialProfileSection.SocialSectionFriends,
				)}
			onShow={(list) => store.explore(person, list)} />
	{/if}

	{#if profile && store.profileLoad?.hasSection(SocialProfileSection.SocialSectionAbout)}
		<section class="flex flex-col gap-2" aria-labelledby="social-about-heading">
			<h3 id="social-about-heading" class="text-xs font-semibold text-fg-muted">
				About
			</h3>
			<p class="text-block" data-variant="prose">
				{profile.description || "No description."}
			</p>
			{#if profile.socialLinks?.length}
				<div class="flex flex-wrap items-center gap-1">
					<Icon
						svg={linkIcon}
						class="mr-1 text-fg-faint"
						size={16}
						aria-label="Social links" />
					{#each profile.socialLinks as link (link.name)}
						<button
							class="button"
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
	{:else if sections?.isSectionPending(SocialProfileSection.SocialSectionAbout)}
		<section class="flex flex-col gap-2" aria-busy="true">
			<h3 class="text-xs font-semibold text-fg-muted">About</h3>
			<p class="text-block" data-variant="prose">
				<Skeleton className="w-72" label="Loading the description" />
			</p>
		</section>
	{/if}

	{#if profile && store.profileLoad?.hasSection(SocialProfileSection.SocialSectionConnections) && accounts.accountTotal > 1}
		<FriendConnections connections={profile.connections ?? []} {accounts} />
	{/if}

	{#if request || sections}
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
				{:else if sections}
					<div>
						<dt>
							<Icon
								svg={calendarDotsIcon}
								size={16}
								aria-hidden="true" />Joined
						</dt>
						<dd>
							<MissingFactValue
								{sections}
								section={SocialProfileSection.SocialSectionAbout} />
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
				{:else if sections}
					<div>
						<dt>
							<Icon
								svg={clockCounterClockwiseIcon}
								size={16}
								aria-hidden="true" />Previous names
						</dt>
						<dd>
							<MissingFactValue
								{sections}
								section={SocialProfileSection.SocialSectionPreviousNames}
								emptyLabel="None" />
						</dd>
					</div>
				{/if}
				{#if profile?.primaryGroup}
					<PrimaryGroupFact group={profile.primaryGroup} />
				{:else if sections}
					<div>
						<dt>
							<Icon
								svg={usersThreeIcon}
								size={16}
								aria-hidden="true" />Primary group
						</dt>
						<dd>
							<MissingFactValue
								{sections}
								section={SocialProfileSection.SocialSectionPrimaryGroup}
								emptyLabel="None" />
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
				{:else if sections}
					<div>
						<dt>
							<Icon svg={medalIcon} size={16} aria-hidden="true" />Roblox
							badges
						</dt>
						<dd>
							<MissingFactValue
								{sections}
								section={SocialProfileSection.SocialSectionRobloxBadges}
								emptyLabel="None" />
						</dd>
					</div>
				{/if}
				{#if profile?.tradeEligibility}
					{@const trading = profile.tradeEligibility}
					<div data-tone={trading.canTrade ? undefined : "warning"}>
						<dt>
							<Icon svg={swapIcon} size={16} aria-hidden="true" />Trading
						</dt>
						<dd>
							<span
								>{trading.canTrade
									? "Can trade"
									: (tradeEligibilityLabels[trading.eligibility] ??
										"Cannot trade")}</span>
						</dd>
					</div>
				{/if}
				{#if profile && profile.communityCount !== null}
					<div>
						<dt>
							<Icon
								svg={usersFourIcon}
								size={16}
								aria-hidden="true" />Communities
						</dt>
						<dd>
							<span>{exactNumbers.format(profile.communityCount)}</span>
						</dd>
					</div>
				{:else if sections}
					<div>
						<dt>
							<Icon
								svg={usersFourIcon}
								size={16}
								aria-hidden="true" />Communities
						</dt>
						<dd>
							<MissingFactValue
								{sections}
								section={SocialProfileSection.SocialSectionCommunities} />
						</dd>
					</div>
				{/if}
				{#if profile && profile.collectibleCount !== null}
					<div>
						<dt>
							<Icon
								svg={diamondIcon}
								size={16}
								aria-hidden="true" />Collectibles
						</dt>
						<dd>
							<span
								>{exactNumbers.format(profile.collectibleCount)}
								{profile.collectibleCount === 1
									? "item"
									: "items"}</span>
							{#if profile.collectibleCount > 0 && profile.collectibleRecentAveragePriceTotal !== null}<small
									class="font-mono"
									>RAP R$ {exactNumbers.format(
										profile.collectibleRecentAveragePriceTotal,
									)}</small
								>{/if}
						</dd>
					</div>
				{:else if sections}
					<div>
						<dt>
							<Icon
								svg={diamondIcon}
								size={16}
								aria-hidden="true" />Collectibles
						</dt>
						<dd>
							<MissingFactValue
								{sections}
								section={SocialProfileSection.SocialSectionCollectibles} />
						</dd>
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
				{:else if sections}
					<div>
						<dt>
							<Icon
								svg={backpackIcon}
								size={16}
								aria-hidden="true" />Inventory
						</dt>
						<dd>
							<MissingFactValue
								{sections}
								section={SocialProfileSection.SocialSectionInventory} />
						</dd>
					</div>
				{/if}
			</dl>
		</section>
	{/if}

	{#if store.profileError}
		<p class="text-xs text-danger-muted" role="alert">{store.profileError}</p>
	{:else if (store.profileLoad?.failedSections.length ?? 0) > 0 || (profileRelationship?.unavailable?.length ?? 0) > 0}
		<p class="text-xs text-danger-muted" role="alert">
			Some profile details could not be loaded. Refresh to retry.
		</p>
	{/if}
</section>

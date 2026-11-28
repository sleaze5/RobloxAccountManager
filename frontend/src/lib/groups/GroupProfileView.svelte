<script lang="ts">
	import circleNotchIcon from "@phosphor-icons/core/regular/circle-notch.svg?raw"
	import { notifyErrors } from "../notifications/error-notifications.svelte"
	import arrowSquareOutIcon from "@phosphor-icons/core/regular/arrow-square-out.svg?raw"
	import clockCounterClockwiseIcon from "@phosphor-icons/core/regular/clock-counter-clockwise.svg?raw"
	import coinsIcon from "@phosphor-icons/core/regular/coins.svg?raw"
	import dotsThreeIcon from "@phosphor-icons/core/regular/dots-three.svg?raw"
	import starFillIcon from "@phosphor-icons/core/fill/star-fill.svg?raw"
	import starIcon from "@phosphor-icons/core/regular/star.svg?raw"
	import linkIcon from "@phosphor-icons/core/regular/link.svg?raw"
	import medalIcon from "@phosphor-icons/core/regular/medal.svg?raw"
	import megaphoneIcon from "@phosphor-icons/core/regular/megaphone.svg?raw"
	import pencilSimpleIcon from "@phosphor-icons/core/regular/pencil-simple.svg?raw"
	import signInIcon from "@phosphor-icons/core/regular/sign-in.svg?raw"
	import signOutIcon from "@phosphor-icons/core/regular/sign-out.svg?raw"
	import tagIcon from "@phosphor-icons/core/regular/tag.svg?raw"
	import userPlusIcon from "@phosphor-icons/core/regular/user-plus.svg?raw"
	import warningIcon from "@phosphor-icons/core/regular/warning.svg?raw"
	import sealCheckIcon from "@phosphor-icons/core/regular/seal-check.svg?raw"
	import { Browser } from "@wailsio/runtime"
	import { GroupSource } from "../backend/bridge"
	import type { GroupSummary } from "../backend/bridge"
	import CreatorCard from "../shared/CreatorCard.svelte"
	import CreatorByline from "../shared/CreatorByline.svelte"
	import ProfileStats from "../shared/ProfileStats.svelte"
	import ProfileTabs from "../shared/ProfileTabs.svelte"
	import type { CreatorRole } from "../shared/CreatorCard.svelte"
	import { errorMessage } from "../shared/errors"
	import Icon from "../shared/Icon.svelte"
	import RefreshIcon from "../shared/RefreshIcon.svelte"
	import Timestamp from "../shared/Timestamp.svelte"
	import { relativeTime } from "../shared/timestamp"
	import ConfirmationDialog from "../shared/ConfirmationDialog.svelte"
	import RobloxActionStrip from "../shared/RobloxActionStrip.svelte"
	import type { ConfirmationText } from "../shared/roblox-confirmations"
	import { joinConfirmation, primaryGroupConfirmation } from "./group-confirmations"
	import GroupIcon from "./GroupIcon.svelte"
	import GroupMembershipFacts from "./GroupMembershipFacts.svelte"
	import GroupStatusBadges from "./GroupStatusBadges.svelte"
	import { groupStats } from "./group-stats"
	import type { GroupsStore, GroupSubpage } from "./groups-store.svelte"

	let { store }: { store: GroupsStore } = $props()

	const exactNumbers = new Intl.NumberFormat("en-US"),
		sectionHeadingClass = "text-xs font-semibold text-fg-muted",
		metaClass =
			"flex min-w-0 flex-wrap items-center gap-1 font-mono text-2xs wrap-anywhere text-fg-muted",
		maxShoutLength = 255,
		maxDescriptionLength = 1000

	// GroupRobloxChange names a change to the group on Roblox that the user
	// confirms first.
	type GroupRobloxChange =
		| "join"
		| "primary"
		| "removePrimary"
		| "shout"
		| "clearShout"
		| "description"

	let isConfirmingLeave = $state(false),
		pendingChange = $state<GroupRobloxChange | null>(null),
		isEditingShout = $state(false),
		shoutDraft = $state(""),
		isEditingDescription = $state(false),
		descriptionDraft = $state(""),
		linkError = $state("")

	const profile = $derived(store.profile),
		details = $derived(profile?.details),
		group = $derived(details?.group),
		membership = $derived(profile?.membership),
		permissions = $derived(membership?.permissions),
		isMember = $derived(!!membership && membership.role.rank > 0),
		unavailable = $derived(new Set(profile?.unavailable ?? [])),
		roles = $derived(profile?.roles ?? []),
		hasMembershipAction = $derived(
			isMember
				? !membership?.isOwner
				: !membership?.isPendingJoin && !details?.isLocked,
		),
		headlineStats = $derived(group ? groupStats(group) : [])

	function startShout(): void {
		store.clearActionError()
		shoutDraft = details?.shout?.body ?? ""
		isEditingShout = true
	}

	async function confirmChange(change: GroupRobloxChange): Promise<void> {
		let isSaved = false
		if (change === "join") isSaved = await store.join()
		else if (change === "primary")
			isSaved = await store.makePrimary(store.selectedGroupId)
		else if (change === "removePrimary")
			isSaved = await store.removePrimary(store.selectedGroupId)
		else if (change === "description")
			isSaved = await store.saveDescription(descriptionDraft)
		else isSaved = await store.postShout(change === "shout" ? shoutDraft : "")
		if (!isSaved) return
		if (change === "description") isEditingDescription = false
		if (change === "shout" || change === "clearShout") isEditingShout = false
		pendingChange = null
	}

	function confirmationFor(
		change: GroupRobloxChange,
		changedGroup: GroupSummary,
	): ConfirmationText {
		if (change === "join") return joinConfirmation(changedGroup)
		if (change === "primary" || change === "removePrimary")
			return primaryGroupConfirmation(
				changedGroup.name,
				change === "removePrimary",
			)
		if (change === "shout")
			return {
				title: "Post shout on Roblox?",
				description: `Every member of ${changedGroup.name} sees the shout. It replaces the current shout.`,
				confirmLabel: "Post shout",
				progressLabel: "Posting…",
			}
		if (change === "clearShout")
			return {
				title: "Clear shout on Roblox?",
				description: `The shout leaves the page of ${changedGroup.name} for every member.`,
				confirmLabel: "Clear shout",
				progressLabel: "Clearing…",
			}
		return {
			title: "Save description on Roblox?",
			description: `Every visitor of ${changedGroup.name} sees the new description.`,
			confirmLabel: "Save description",
			progressLabel: "Saving…",
		}
	}

	function requestChange(change: GroupRobloxChange): void {
		store.clearActionError()
		pendingChange = change
	}

	function startDescription(): void {
		store.clearActionError()
		descriptionDraft = group?.description ?? ""
		isEditingDescription = true
	}

	async function leave(): Promise<void> {
		if (await store.leave()) isConfirmingLeave = false
	}

	async function openLink(url: string): Promise<void> {
		linkError = ""
		try {
			await Browser.OpenURL(url)
		} catch (error) {
			linkError = errorMessage(error, "Could not open the link. Try again.")
		}
	}

	notifyErrors("Group not loaded", () => store.profileError)
	notifyErrors("Group action not completed", () => store.actionError)

	notifyErrors("Link not opened", () => linkError)
</script>

{#snippet relatedGroups(
	title: string,
	relatedRole: CreatorRole,
	summaries: GroupSummary[],
	source: GroupSource,
)}
	{#if summaries.length > 0 || unavailable.has(source)}
		<section class="flex flex-col gap-3" aria-label={title}>
			<h3 class={["flex items-center gap-2", sectionHeadingClass]}>
				{title}
				{#if summaries.length > 0}<span
						class="font-mono text-2xs font-normal text-fg-faint"
						>{summaries.length}</span
					>{/if}
			</h3>
			{#if unavailable.has(source)}
				<p class="text-xs text-fg-muted">Unavailable</p>
			{:else}
				<ul class="grid grid-cols-1 gap-2 @lg:grid-cols-2 @3xl:grid-cols-3">
					{#each summaries as summary (summary.groupId)}
						<li class="min-w-0">
							<CreatorCard
								kind="Group"
								creatorId={summary.groupId}
								name={summary.name}
								hasVerifiedBadge={summary.hasVerifiedBadge}
								role={relatedRole}
								memberCount={summary.memberCount} />
						</li>
					{/each}
				</ul>
			{/if}
		</section>
	{/if}
{/snippet}

{#if !profile && store.profileError}
	<div class="empty-state h-full py-16">
		<Icon svg={warningIcon} size={28} aria-hidden="true" />
		<h2>Group unavailable</h2>
		<button class="button" type="button" onclick={() => void store.refreshProfile()}
			>Try again</button>
	</div>
{:else if !profile || !details || !group || !membership || !permissions}
	<div class="flex flex-col gap-4 p-3" role="status" aria-busy="true">
		<span class="sr-only">Loading group…</span>
		<div class="flex items-start gap-4">
			<span class="size-24 shrink-0 skeleton rounded-md"></span>
			<div class="flex min-w-0 flex-1 flex-col gap-1 pt-1">
				<span class="text-lg"><span class="skeleton-line w-48"></span></span>
				<span class="text-2xs"><span class="skeleton-line w-24"></span></span>
				<span class="text-xs"><span class="skeleton-line w-40"></span></span>
				<span class="text-sm"><span class="skeleton-line w-28"></span></span>
			</div>
		</div>
		<span class="h-20 w-full skeleton rounded-lg"></span>
	</div>
{:else}
	<section class="flex flex-col gap-4 p-3" aria-labelledby="group-name">
		<div class="flex items-start gap-4">
			<GroupIcon url={group.iconUrl} size="xl" />
			<div class="flex min-w-0 flex-1 flex-col gap-1 pt-1">
				<h2
					id="group-name"
					class="inline-flex min-w-0 items-center gap-1 text-lg font-semibold text-fg">
					<span class="wrap-anywhere select-text">{group.name}</span>
					{#if group.hasVerifiedBadge}<Icon
							svg={sealCheckIcon}
							class="shrink-0 text-accent-strong"
							size={16}
							aria-label="Verified group" />{/if}
				</h2>
				<div class={metaClass}>
					<span class="select-text">ID: {group.groupId}</span>
				</div>
				{#if details.owner}
					<CreatorByline
						kind="User"
						creatorId={details.owner.userId}
						name={details.owner.username}
						hasVerifiedBadge={details.owner.hasVerifiedBadge}
						role="Owner" />
				{:else}
					<p class="text-xs text-fg-muted">Nobody owns this group.</p>
				{/if}
				<ProfileStats label="Group numbers" stats={headlineStats} />
				<GroupStatusBadges
					isPrimary={membership.isPrimary}
					isPendingJoin={membership.isPendingJoin}
					isLocked={details.isLocked} />
			</div>
			<div class="flex shrink-0 flex-wrap items-center justify-end gap-1">
				{#if isMember || hasMembershipAction}
					<RobloxActionStrip username={store.account.username}>
						{#if isMember && !membership.isOwner}
							<button
								type="button"
								aria-label="Leave group"
								data-tooltip="Leave group"
								disabled={store.pendingAction !== null}
								onclick={() => {
									store.clearActionError()
									isConfirmingLeave = true
								}}>
								<Icon svg={signOutIcon} size={14} aria-hidden="true" />
							</button>
						{:else if !isMember && hasMembershipAction}
							{@const joinLabel = group.isPublicEntryAllowed
								? "Join group"
								: "Request to join"}
							{@const isJoining = store.pendingAction === "join"}
							<button
								type="button"
								aria-label={isJoining ? "Joining…" : joinLabel}
								aria-busy={isJoining}
								data-tooltip={joinLabel}
								disabled={store.pendingAction !== null}
								onclick={() => requestChange("join")}>
								<Icon
									svg={isJoining ? circleNotchIcon : signInIcon}
									class={isJoining ? "spinner" : undefined}
									size={14}
									aria-hidden="true" />
							</button>
						{/if}
						{#if isMember}
							{@const primaryLabel = membership.isPrimary
								? "Remove primary group"
								: "Make primary group"}
							<button
								type="button"
								aria-label={primaryLabel}
								aria-pressed={membership.isPrimary}
								data-tooltip={primaryLabel}
								disabled={store.pendingAction !== null}
								onclick={() =>
									requestChange(
										membership.isPrimary
											? "removePrimary"
											: "primary",
									)}>
								<Icon
									svg={membership.isPrimary ? starFillIcon : starIcon}
									class={[
										membership.isPrimary && "text-accent-strong",
									]}
									size={14}
									aria-hidden="true" />
							</button>
						{/if}
					</RobloxActionStrip>
					<span class="mx-1 h-4 w-px bg-line-strong" aria-hidden="true"
					></span>
				{/if}
				<button
					class="button-icon"
					type="button"
					aria-label={store.isLoadingProfile
						? "Refreshing group…"
						: "Refresh group"}
					data-tooltip="Refresh group"
					aria-busy={store.isLoadingProfile}
					disabled={store.isLoadingProfile}
					onclick={() => void store.refreshProfile()}>
					<RefreshIcon isRefreshing={store.isLoadingProfile} />
				</button>
				<button
					class="button-icon"
					type="button"
					aria-label="More actions"
					aria-haspopup="menu"
					aria-expanded={store.menu?.group.groupId === group.groupId}
					data-open={store.menu?.group.groupId === group.groupId || undefined}
					data-tooltip="More actions"
					data-tooltip-side="bottom-end"
					data-group-menu
					onclick={(event) => {
						if (store.menu) store.closeMenu()
						else store.openMenu(event, group, true)
					}}>
					<Icon svg={dotsThreeIcon} size={16} aria-hidden="true" />
				</button>
			</div>
		</div>

		<div class="-mx-3 tab-bar">
			<ProfileTabs />
		</div>

		<!-- A section with nothing to show is left out, unless the account can
		fill it. -->
		{#if details.shout || permissions.canPostShout}
			<section class="flex flex-col gap-2" aria-labelledby="group-shout-heading">
				<div class="flex min-h-6 items-center gap-2">
					<h3 id="group-shout-heading" class={sectionHeadingClass}>Shout</h3>
					{#if permissions.canPostShout && !isEditingShout}
						<button
							class="ml-auto button-ghost-sm"
							type="button"
							aria-label="Post shout"
							data-tooltip="Post shout"
							onclick={startShout}>
							<Icon svg={pencilSimpleIcon} size={14} aria-hidden="true" />
						</button>
					{/if}
				</div>
				{#if isEditingShout}
					<form
						novalidate
						class="flex animate-enter flex-col gap-2"
						onsubmit={(event) => {
							event.preventDefault()
							requestChange("shout")
						}}>
						<label class="sr-only" for="group-shout-input">Shout</label>
						<textarea
							id="group-shout-input"
							class="field h-20 resize-none py-1.5"
							maxlength={maxShoutLength}
							placeholder="Write a shout for every member"
							disabled={store.pendingAction === "shout"}
							bind:value={shoutDraft}></textarea>
						<div class="flex items-center gap-2">
							<span class="font-mono text-2xs text-fg-muted"
								>{shoutDraft.length}/{maxShoutLength}</span>
							{#if details.shout}
								<button
									class="button-text"
									type="button"
									disabled={store.pendingAction === "shout"}
									onclick={() => requestChange("clearShout")}
									>Clear shout</button>
							{/if}
							<button
								class="ml-auto button"
								type="button"
								disabled={store.pendingAction === "shout"}
								onclick={() => (isEditingShout = false)}>Cancel</button>
							<button
								class="button-primary"
								type="submit"
								aria-busy={store.pendingAction === "shout"}
								disabled={store.pendingAction === "shout" ||
									!shoutDraft.trim()}
								>{store.pendingAction === "shout"
									? "Posting…"
									: "Post shout"}</button>
						</div>
					</form>
				{:else if details.shout}
					<div
						class="flex flex-col gap-2 rounded-lg border border-line bg-surface p-3">
						<div
							class="flex min-w-0 items-center gap-2 text-2xs text-fg-muted">
							<Icon
								svg={megaphoneIcon}
								class="shrink-0"
								size={14}
								aria-hidden="true" />
							<span class="truncate font-medium text-fg"
								>{details.shout.poster.displayName ||
									details.shout.poster.username}</span>
							{#if details.shout.poster.username}<span class="truncate"
									>@{details.shout.poster.username}</span
								>{/if}
							{#if details.shout.updatedAtMs}
								<span class="ml-auto shrink-0"
									><Timestamp value={details.shout.updatedAtMs} /> · {relativeTime(
										details.shout.updatedAtMs,
									)}</span>
							{/if}
						</div>
						<p
							class="text-sm wrap-anywhere whitespace-pre-line text-fg select-text">
							{details.shout.body}
						</p>
					</div>
				{:else}
					<p class="text-xs text-fg-muted">No shout.</p>
				{/if}
			</section>
		{/if}

		{#if group.description || permissions.canConfigure}
			<section
				class="flex flex-col gap-2"
				aria-labelledby="group-description-heading">
				<div class="flex min-h-6 items-center gap-2">
					<h3 id="group-description-heading" class={sectionHeadingClass}>
						Description
					</h3>
					{#if permissions.canConfigure && !isEditingDescription}
						<button
							class="ml-auto button-ghost-sm"
							type="button"
							aria-label="Edit description"
							data-tooltip="Edit description"
							onclick={startDescription}>
							<Icon svg={pencilSimpleIcon} size={14} aria-hidden="true" />
						</button>
					{/if}
				</div>
				{#if isEditingDescription}
					<form
						novalidate
						class="flex animate-enter flex-col gap-2"
						onsubmit={(event) => {
							event.preventDefault()
							requestChange("description")
						}}>
						<label class="sr-only" for="group-description-input"
							>Description</label>
						<textarea
							id="group-description-input"
							class="field h-36 resize-none py-1.5"
							maxlength={maxDescriptionLength}
							disabled={store.pendingAction === "description"}
							bind:value={descriptionDraft}></textarea>
						<div class="flex items-center gap-2">
							<span class="font-mono text-2xs text-fg-muted"
								>{descriptionDraft.length}/{maxDescriptionLength}</span>
							<button
								class="ml-auto button"
								type="button"
								disabled={store.pendingAction === "description"}
								onclick={() => (isEditingDescription = false)}
								>Cancel</button>
							<button
								class="button-primary"
								type="submit"
								aria-busy={store.pendingAction === "description"}
								disabled={store.pendingAction === "description"}
								>{store.pendingAction === "description"
									? "Saving…"
									: "Save description"}</button>
						</div>
					</form>
				{:else}
					<p
						class="max-h-60 overflow-y-auto text-block select-text"
						data-variant="prose">
						{group.description || "No description."}
					</p>
				{/if}
			</section>
		{/if}

		<section
			class="flex flex-col gap-3"
			aria-labelledby="group-information-heading">
			<h3 id="group-information-heading" class={sectionHeadingClass}>
				Information
			</h3>
			<dl class="facts">
				<GroupMembershipFacts
					{membership}
					isPublicEntryAllowed={group.isPublicEntryAllowed} />
				<div
					data-tone={unavailable.has(GroupSource.GroupSourceRoles)
						? "muted"
						: undefined}>
					<dt><Icon svg={tagIcon} size={16} aria-hidden="true" />Roles</dt>
					<dd>
						{#if unavailable.has(GroupSource.GroupSourceRoles)}Unavailable{:else}{roles.length}
							<small class="truncate"
								>{roles
									.slice(-3)
									.reverse()
									.map((role) => role.name)
									.join(" · ")}</small
							>{/if}
					</dd>
				</div>
				{#if profile.settings}
					{@const settings = profile.settings}
					<div>
						<dt>
							<Icon
								svg={pencilSimpleIcon}
								size={16}
								aria-hidden="true" />Name changes
						</dt>
						<dd>
							{settings.isNameChangeEnabled ? "Allowed" : "Not allowed"}
							<small>Roblox charges Robux to rename a group</small>
						</dd>
					</div>
					<div
						data-tone={settings.isRoleAssignedOnJoin == null
							? "muted"
							: undefined}>
						<dt>
							<Icon svg={userPlusIcon} size={16} aria-hidden="true" />Role
							on join
						</dt>
						<dd>
							{settings.isRoleAssignedOnJoin == null
								? "Not shared by Roblox"
								: settings.isRoleAssignedOnJoin
									? "Lowest role"
									: "No role until a member gives one"}
						</dd>
					</div>
					<div
						data-tone={settings.communityTier == null
							? "muted"
							: undefined}>
						<dt>
							<Icon
								svg={medalIcon}
								size={16}
								aria-hidden="true" />Community tier
						</dt>
						<dd>
							{settings.communityTier == null
								? "Not shared by Roblox"
								: `Tier ${settings.communityTier}`}
							{#if settings.communityTier != null && settings.communityTierUpdatedMs}<small
									>Updated {relativeTime(
										settings.communityTierUpdatedMs,
									)}</small
								>{/if}
						</dd>
					</div>
				{/if}
				{#if permissions.canSpendFunds}
					<div
						data-tone={profile.fundsInRobux === null ? "muted" : undefined}>
						<dt>
							<Icon svg={coinsIcon} size={16} aria-hidden="true" />Funds
						</dt>
						<dd class="font-mono">
							{profile.fundsInRobux === null
								? "Unavailable"
								: `R$ ${exactNumbers.format(profile.fundsInRobux)}`}
						</dd>
					</div>
				{/if}
				{#if (profile.previousNames?.length ?? 0) > 0}
					<div>
						<dt>
							<Icon
								svg={clockCounterClockwiseIcon}
								size={16}
								aria-hidden="true" />Previous names
						</dt>
						<dd class="flex flex-col">
							{#each profile.previousNames ?? [] as nameChange (`${nameChange.name}:${nameChange.changedAtMs}`)}
								<span class="wrap-anywhere"
									>{nameChange.name}{#if nameChange.changedAtMs}<small
											class="ml-1"
											>· until <Timestamp
												value={nameChange.changedAtMs} /></small
										>{/if}</span>
							{/each}
						</dd>
					</div>
				{/if}
				{#if (profile.socialLinks?.length ?? 0) > 0}
					<div>
						<dt>
							<Icon svg={linkIcon} size={16} aria-hidden="true" />Social
							links
						</dt>
						<dd>
							{#each profile.socialLinks ?? [] as link (link.url)}
								<button
									class="button"
									type="button"
									aria-label={`Open ${link.title || link.type} (${link.url})`}
									data-tooltip={link.url}
									onclick={() => void openLink(link.url)}>
									<Icon
										svg={arrowSquareOutIcon}
										size={16}
										aria-hidden="true" />
									{link.title || link.type}
								</button>
							{/each}
						</dd>
					</div>
				{/if}
			</dl>
		</section>

		{@render relatedGroups(
			"Allies",
			"Ally",
			profile.allies ?? [],
			GroupSource.GroupSourceAllies,
		)}
		{@render relatedGroups(
			"Enemies",
			"Enemy",
			profile.enemies ?? [],
			GroupSource.GroupSourceEnemies,
		)}
	</section>
{/if}

{#if isConfirmingLeave && group}
	<ConfirmationDialog
		title="Leave group?"
		description={`This account will leave ${group.name} and lose its role. Joining again may need approval.`}
		confirmLabel="Leave group"
		progressLabel="Leaving…"
		account={store.account}
		isWorking={store.pendingAction === "leave"}
		onConfirm={() => void leave()}
		onCancel={() => {
			store.clearActionError()
			isConfirmingLeave = false
		}} />
{/if}

{#if pendingChange && group}
	{@const confirmation = confirmationFor(pendingChange, group)}
	<ConfirmationDialog
		{...confirmation}
		tone="primary"
		account={store.account}
		isWorking={store.pendingAction !== null}
		onConfirm={() => pendingChange && void confirmChange(pendingChange)}
		onCancel={() => {
			store.clearActionError()
			pendingChange = null
		}} />
{/if}

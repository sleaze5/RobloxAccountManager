<script lang="ts">
	import arrowSquareOutIcon from "@phosphor-icons/core/regular/arrow-square-out.svg?raw"
	import checkCircleFillIcon from "@phosphor-icons/core/fill/check-circle-fill.svg?raw"
	import clockIcon from "@phosphor-icons/core/regular/clock.svg?raw"
	import coinsIcon from "@phosphor-icons/core/regular/coins.svg?raw"
	import crownIcon from "@phosphor-icons/core/regular/crown.svg?raw"
	import doorOpenIcon from "@phosphor-icons/core/regular/door-open.svg?raw"
	import gameControllerIcon from "@phosphor-icons/core/regular/game-controller.svg?raw"
	import globeIcon from "@phosphor-icons/core/regular/globe-hemisphere-west.svg?raw"
	import identificationBadgeIcon from "@phosphor-icons/core/regular/identification-badge.svg?raw"
	import linkIcon from "@phosphor-icons/core/regular/link.svg?raw"
	import lockIcon from "@phosphor-icons/core/regular/lock.svg?raw"
	import megaphoneIcon from "@phosphor-icons/core/regular/megaphone.svg?raw"
	import pencilSimpleIcon from "@phosphor-icons/core/regular/pencil-simple.svg?raw"
	import signInIcon from "@phosphor-icons/core/regular/sign-in.svg?raw"
	import signOutIcon from "@phosphor-icons/core/regular/sign-out.svg?raw"
	import starFillIcon from "@phosphor-icons/core/fill/star-fill.svg?raw"
	import starIcon from "@phosphor-icons/core/regular/star.svg?raw"
	import storefrontIcon from "@phosphor-icons/core/regular/storefront.svg?raw"
	import tagIcon from "@phosphor-icons/core/regular/tag.svg?raw"
	import userPlusIcon from "@phosphor-icons/core/regular/user-plus.svg?raw"
	import usersThreeIcon from "@phosphor-icons/core/regular/users-three.svg?raw"
	import warningIcon from "@phosphor-icons/core/regular/warning.svg?raw"
	import sealCheckIcon from "@phosphor-icons/core/regular/seal-check.svg?raw"
	import { Browser } from "@wailsio/runtime"
	import { GroupSource } from "../backend/bridge"
	import type { GroupSummary } from "../backend/bridge"
	import CreatorCard from "../shared/CreatorCard.svelte"
	import { errorMessage } from "../shared/errors"
	import Icon from "../shared/Icon.svelte"
	import RefreshIcon from "../shared/RefreshIcon.svelte"
	import Timestamp from "../shared/Timestamp.svelte"
	import { relativeTime } from "../shared/timestamp"
	import ConfirmationDialog from "../shared/ConfirmationDialog.svelte"
	import GroupIcon from "./GroupIcon.svelte"
	import type { GroupsStore, GroupSubpage } from "./groups-store.svelte"

	let {
		store,
		onOpenInBrowser,
	}: {
		store: GroupsStore
		onOpenInBrowser: (accountId: number, pageUrl: string) => void
	} = $props()

	const exactNumbers = new Intl.NumberFormat("en-US"),
		compactNumbers = new Intl.NumberFormat("en-US", {
			notation: "compact",
			maximumFractionDigits: 1,
		}),
		sectionHeadingClass = "text-xs font-semibold text-fg-muted",
		metaClass =
			"flex min-w-0 flex-wrap items-center gap-1 font-mono text-2xs wrap-anywhere text-fg-muted",
		maxShoutLength = 255,
		maxDescriptionLength = 1000

	let isConfirmingLeave = $state(false),
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
		isPrimary = $derived(!!membership?.isPrimary),
		unavailable = $derived(new Set(profile?.unavailable ?? [])),
		roles = $derived(profile?.roles ?? []),
		subpageButtons = $derived<
			{ subpage: GroupSubpage; label: string; icon: string; isShown: boolean }[]
		>([
			{
				subpage: "members",
				label: "Members",
				icon: usersThreeIcon,
				isShown: true,
			},
			{
				subpage: "games",
				label: "Games",
				icon: gameControllerIcon,
				isShown: true,
			},
			{
				subpage: "items",
				label: "Items",
				icon: storefrontIcon,
				isShown: true,
			},
			{
				subpage: "joinRequests",
				label: "Join requests",
				icon: userPlusIcon,
				isShown: !!permissions?.canAcceptJoinRequests,
			},
		])

	function startShout(): void {
		store.clearActionError()
		shoutDraft = details?.shout?.body ?? ""
		isEditingShout = true
	}

	async function postShout(message: string): Promise<void> {
		if (await store.postShout(message)) isEditingShout = false
	}

	function startDescription(): void {
		store.clearActionError()
		descriptionDraft = group?.description ?? ""
		isEditingDescription = true
	}

	async function saveDescription(): Promise<void> {
		if (await store.saveDescription(descriptionDraft)) isEditingDescription = false
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
</script>

{#snippet relatedGroups(
	title: string,
	relatedRole: string,
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
								role={`${relatedRole} · ${compactNumbers.format(summary.memberCount)} members`} />
						</li>
					{/each}
				</ul>
			{/if}
		</section>
	{/if}
{/snippet}

{#if !profile && store.profileError}
	<div class="empty-state h-full py-16" role="alert">
		<Icon svg={warningIcon} size={28} aria-hidden="true" />
		<h2>Group unavailable</h2>
		<p>{store.profileError}</p>
		<button class="button" type="button" onclick={() => void store.refreshProfile()}
			>Try again</button>
	</div>
{:else if !profile || !details || !group || !membership || !permissions}
	<div class="flex flex-col gap-4 p-3" role="status" aria-busy="true">
		<span class="sr-only">Loading group…</span>
		<div class="flex items-start gap-3">
			<span class="size-24 shrink-0 skeleton rounded-md"></span>
			<div class="flex min-w-0 flex-1 flex-col gap-2 pt-1">
				<span class="text-lg"><span class="skeleton-line w-48"></span></span>
				<span class="text-2xs"><span class="skeleton-line w-40"></span></span>
				<span class="h-12 w-72 skeleton rounded-lg"></span>
			</div>
		</div>
		<span class="h-20 w-full skeleton rounded-lg"></span>
	</div>
{:else}
	<section class="flex flex-col gap-4 p-3 pb-4" aria-labelledby="group-name">
		<div class="flex items-start gap-3 @max-xl:flex-col">
			<div class="flex min-w-0 flex-1 items-start gap-3">
				<GroupIcon url={group.iconUrl} size="xl" />
				<div class="flex min-w-0 flex-col gap-1">
					<h2
						id="group-name"
						class="inline-flex min-w-0 items-center gap-1 text-lg font-semibold text-fg">
						<span class="wrap-anywhere select-text">{group.name}</span>
						{#if group.hasVerifiedBadge}<Icon
								svg={sealCheckIcon}
								class="shrink-0 text-fg-muted"
								size={16}
								aria-label="Verified group" />{/if}
					</h2>
					<div class={metaClass}>
						<span class="select-text">ID: {group.groupId}</span>
						<span class="text-fg-faint" aria-hidden="true">·</span>
						<span
							>{exactNumbers.format(group.memberCount)}
							{group.memberCount === 1 ? "member" : "members"}</span>
					</div>
					<ul class="mt-1 flex min-h-6 flex-wrap gap-1" aria-label="Status">
						{#if isMember}
							<li class="tag-pill-success">
								<Icon
									svg={checkCircleFillIcon}
									size={12}
									aria-hidden="true" />
								<span>{membership.role.name}</span>
							</li>
						{/if}
						{#if membership.isOwner}
							<li class="tag-pill-favorite">
								<Icon svg={crownIcon} size={12} aria-hidden="true" />
								<span>Owner</span>
							</li>
						{/if}
						{#if isPrimary}
							<li class="tag-pill-favorite">
								<Icon svg={starFillIcon} size={12} aria-hidden="true" />
								<span>Primary group</span>
							</li>
						{/if}
						{#if membership.isPendingJoin}
							<li class="tag-pill-warning">
								<Icon svg={clockIcon} size={12} aria-hidden="true" />
								<span>Join request sent</span>
							</li>
						{/if}
						{#if details.isLocked}
							<li class="tag-pill-warning">
								<Icon svg={lockIcon} size={12} aria-hidden="true" />
								<span>Locked by Roblox</span>
							</li>
						{/if}
						<li class="tag-pill">
							<Icon
								svg={group.isPublicEntryAllowed
									? doorOpenIcon
									: identificationBadgeIcon}
								size={12}
								aria-hidden="true" />
							<span
								>{group.isPublicEntryAllowed
									? "Anyone can join"
									: "Join by request"}</span>
						</li>
					</ul>
					{#if details.owner}
						<div class="mt-1 w-72 max-w-full">
							<CreatorCard
								kind="User"
								creatorId={details.owner.userId}
								name={details.owner.username}
								displayName={details.owner.displayName}
								hasVerifiedBadge={details.owner.hasVerifiedBadge}
								role="Owner" />
						</div>
					{:else}
						<p class="text-xs text-fg-muted">Nobody owns this group.</p>
					{/if}
				</div>
			</div>
			<div
				class="flex flex-col items-end gap-2 @max-xl:items-start @max-xl:self-stretch">
				<div class="flex flex-wrap items-center justify-end gap-1">
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
					{#each subpageButtons.filter((button) => button.isShown) as button (button.subpage)}
						<button
							class="button-icon"
							type="button"
							aria-label={button.label}
							data-tooltip={button.label}
							data-group-subpage={button.subpage}
							onclick={() => (store.subpage = button.subpage)}>
							<Icon svg={button.icon} size={16} aria-hidden="true" />
						</button>
					{/each}
					{#if isMember}
						<button
							class="button-icon"
							type="button"
							aria-label={isPrimary
								? "Remove primary group"
								: "Make primary group"}
							aria-pressed={isPrimary}
							data-tooltip={isPrimary
								? "Remove primary group"
								: "Make primary group"}
							aria-busy={store.pendingAction === "primary" ||
								store.pendingAction === "removePrimary"}
							disabled={store.pendingAction !== null}
							onclick={() =>
								void (isPrimary
									? store.removePrimary()
									: store.makePrimary())}>
							<Icon
								svg={isPrimary ? starFillIcon : starIcon}
								class={isPrimary
									? "animate-pop text-accent-strong"
									: ""}
								size={16}
								aria-hidden="true" />
						</button>
					{/if}
					<button
						class="button-icon"
						type="button"
						aria-label="Open group in browser"
						data-tooltip="Open group in browser"
						data-tooltip-side="bottom-end"
						onclick={() =>
							onOpenInBrowser(
								store.accountId,
								`https://www.roblox.com/communities/${group.groupId}`,
							)}>
						<Icon svg={globeIcon} size={16} aria-hidden="true" />
					</button>
				</div>
				{#if isMember && !membership.isOwner}
					<button
						class="button"
						type="button"
						disabled={store.pendingAction !== null}
						onclick={() => {
							store.clearActionError()
							isConfirmingLeave = true
						}}>
						<Icon svg={signOutIcon} size={16} aria-hidden="true" />Leave
						group
					</button>
				{:else if !isMember && !membership.isPendingJoin && !details.isLocked}
					<button
						class="button-primary"
						type="button"
						aria-busy={store.pendingAction === "join"}
						disabled={store.pendingAction !== null}
						onclick={() => void store.join()}>
						<Icon svg={signInIcon} size={16} aria-hidden="true" />
						{store.pendingAction === "join"
							? "Joining…"
							: group.isPublicEntryAllowed
								? "Join group"
								: "Request to join"}
					</button>
				{/if}
			</div>
		</div>

		{#if store.actionError && !isConfirmingLeave}
			<p class="notice-error" role="alert">{store.actionError}</p>
		{/if}

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
						void postShout(shoutDraft)
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
								onclick={() => void postShout("")}>Clear shout</button>
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
					<div class="flex min-w-0 items-center gap-2 text-2xs text-fg-muted">
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

		<section class="flex flex-col gap-2" aria-labelledby="group-about-heading">
			<div class="flex min-h-6 items-center gap-2">
				<h3 id="group-about-heading" class={sectionHeadingClass}>About</h3>
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
						void saveDescription()
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
					class="max-h-60 text-block overflow-y-auto select-text"
					data-variant="prose">
					{group.description || "No description."}
				</p>
			{/if}
		</section>

		<section
			class="flex flex-col gap-3"
			aria-labelledby="group-information-heading">
			<h3 id="group-information-heading" class={sectionHeadingClass}>
				Information
			</h3>
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
				<div data-tone={isMember ? undefined : "muted"}>
					<dt>
						<Icon
							svg={identificationBadgeIcon}
							size={16}
							aria-hidden="true" />Your role
					</dt>
					<dd>
						{isMember ? membership.role.name : "Not a member"}
						{#if isMember}<small>Rank {membership.role.rank}</small>{/if}
					</dd>
				</div>
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
								svg={clockIcon}
								size={16}
								aria-hidden="true" />Previous names
						</dt>
						<dd class="flex flex-col">
							{#each profile.previousNames ?? [] as nameChange (`${nameChange.name}:${nameChange.changedAtMs}`)}
								<span class="wrap-anywhere"
									>{nameChange.name}{#if nameChange.changedAtMs}<small>
											· until <Timestamp
												value={nameChange.changedAtMs} /></small
										>{/if}</span>
							{/each}
						</dd>
					</div>
				{/if}
			</dl>
		</section>

		{#if (profile.socialLinks?.length ?? 0) > 0}
			<section class="flex flex-col gap-2" aria-labelledby="group-links-heading">
				<h3
					id="group-links-heading"
					class={["flex items-center gap-2", sectionHeadingClass]}>
					<Icon svg={linkIcon} size={14} aria-hidden="true" />Social links
				</h3>
				<div class="flex flex-wrap items-center gap-1">
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
				</div>
				{#if linkError}<p class="text-xs text-danger-muted" role="alert">
						{linkError}
					</p>{/if}
			</section>
		{/if}

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
		isWorking={store.pendingAction === "leave"}
		error={store.actionError}
		onConfirm={() => void leave()}
		onCancel={() => {
			store.clearActionError()
			isConfirmingLeave = false
		}} />
{/if}

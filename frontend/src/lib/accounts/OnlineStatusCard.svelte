<script lang="ts">
	import { getProfilePreviews } from "../shared/profile-preview"
	import arrowsOutSimpleIcon from "@phosphor-icons/core/regular/arrows-out-simple.svg?raw"
	import signInIcon from "@phosphor-icons/core/regular/sign-in.svg?raw"
	import Icon from "../shared/Icon.svelte"
	import { OnlineStatusType, UserJoinability } from "../backend/bridge"
	import type { OnlineStatus } from "../backend/bridge"
	import PlaceHeader from "../games/PlaceHeader.svelte"
	import { floatMenu, menuIn, menuOut } from "../shared/transitions"
	import {
		onlineStatusIcon,
		onlineStatusLabel,
		onlineStatusTextClass,
	} from "./account-model"
	import { notJoinableMessage } from "./user-joinability"

	let {
		id,
		anchor,
		onlineStatus,
		isJoinable,
		joinability,
		onPointerEnter,
		onPointerLeave,
		onFillLaunch,
		onFollowUser,
	}: {
		id: string
		anchor: HTMLElement
		onlineStatus: OnlineStatus
		isJoinable: boolean
		// joinability is whether the user's profile offers to join them.
		joinability: UserJoinability
		onPointerEnter: () => void
		onPointerLeave: () => void
		onFillLaunch: () => void
		// onFollowUser fills the launch options to follow a user whose server
		// the online status hides.
		onFollowUser?: () => void
	} = $props()
	const previews = getProfilePreviews()

	const headingId = $derived(`${id}-heading`),
		isNotJoinable = $derived(joinability === UserJoinability.UserNotJoinable)
	const universeId = $derived(onlineStatus.universeId ?? 0),
		placeId = $derived(onlineStatus.placeId ?? 0),
		rootPlaceId = $derived(onlineStatus.rootPlaceId ?? 0)
</script>

<div
	class="fixed flex w-80 flex-col gap-3 rounded-xl border border-line-strong bg-overlay p-3 shadow-menu"
	{id}
	role="dialog"
	tabindex="-1"
	aria-labelledby={headingId}
	use:floatMenu={{ anchor, align: "start" }}
	onpointerenter={onPointerEnter}
	onpointerleave={onPointerLeave}
	in:menuIn
	out:menuOut>
	<h2
		class={[
			"inline-flex items-center gap-1 text-xs font-semibold",
			onlineStatusTextClass(onlineStatus),
		]}
		id={headingId}>
		<Icon
			svg={onlineStatusIcon(onlineStatus)}
			size={14}
			aria-hidden="true" />{onlineStatusLabel(onlineStatus)}
	</h2>
	{#if universeId > 0 && placeId > 0}<PlaceHeader {universeId} {placeId} />{/if}
	<dl class="detail-grid" aria-label="Game IDs">
		{#if placeId > 0}<div>
				<dt>Place ID</dt>
				{@render previewableId(placeId, "Show game", () =>
					previews.show({ kind: "Place", placeId }),
				)}
			</div>{/if}
		{#if universeId > 0}<div>
				<dt>Universe ID</dt>
				{@render previewableId(universeId, "Show the places of this game", () =>
					previews.show({
						kind: "Place",
						placeId: rootPlaceId || placeId,
						universeId,
					}),
				)}
			</div>{/if}
		{#if rootPlaceId > 0 && rootPlaceId !== placeId}<div>
				<dt>Root place ID</dt>
				{@render previewableId(rootPlaceId, "Show root place", () =>
					previews.show({ kind: "Place", placeId: rootPlaceId }),
				)}
			</div>{/if}
		{#if onlineStatus.statusType === OnlineStatusType.OnlineStatusTypeInGame}<div
				class="col-span-full">
				<dt>Server ID</dt>
				{#if onlineStatus.serverId}<dd class="font-mono select-text">
						{onlineStatus.serverId}
					</dd>
				{:else}<dd class="text-fg-muted">Not shared by Roblox</dd>{/if}
			</div>{/if}
	</dl>
	{#if isJoinable}
		<button
			class={isNotJoinable ? "button-danger" : "button-primary"}
			type="button"
			aria-describedby={isNotJoinable ? `${id}-not-joinable` : undefined}
			onclick={onFillLaunch}>
			<Icon svg={signInIcon} size={16} aria-hidden="true" />
			Fill launch options
		</button>
	{:else if onFollowUser}
		<button class="button-primary" type="button" onclick={onFollowUser}>
			<Icon svg={signInIcon} size={16} aria-hidden="true" />
			Fill launch options to follow
		</button>
	{/if}
	{#if isNotJoinable}
		<p class="-mt-1 text-xs text-danger-muted" id={`${id}-not-joinable`}>
			{notJoinableMessage}
		</p>
	{/if}
</div>

{#snippet previewableId(
	placeOrUniverseId: number,
	previewLabel: string,
	onPreview: () => void,
)}
	<dd class="flex min-w-0 items-center gap-1">
		<span class="truncate font-mono select-text">{placeOrUniverseId}</span>
		<button
			class="button-ghost-sm"
			type="button"
			aria-label={previewLabel}
			data-tooltip={previewLabel}
			onclick={onPreview}>
			<Icon svg={arrowsOutSimpleIcon} size={14} aria-hidden="true" />
		</button>
	</dd>
{/snippet}

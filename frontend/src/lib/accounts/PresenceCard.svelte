<script lang="ts">
	import { getProfilePreviews } from "../shared/profile-preview"
	import magnifyingGlassIcon from "@phosphor-icons/core/regular/magnifying-glass.svg?raw"
	import signInIcon from "@phosphor-icons/core/regular/sign-in.svg?raw"
	import Icon from "../shared/Icon.svelte"
	import { PresenceType, UserJoinability } from "../backend/bridge"
	import type { UserPresence } from "../backend/bridge"
	import PlaceHeader from "../games/PlaceHeader.svelte"
	import { floatMenu, menuIn, menuOut } from "../shared/presence"
	import { presenceIcon, presenceLabel, presenceTextClass } from "./account-model"
	import { notJoinableMessage } from "./user-joinability"

	let {
		id,
		anchor,
		presence,
		isJoinable,
		joinability,
		onPointerEnter,
		onPointerLeave,
		onFillLaunch,
		onFollowUser,
	}: {
		id: string
		anchor: HTMLElement
		presence: UserPresence
		isJoinable: boolean
		// joinability is whether the user's profile offers to join them.
		joinability: UserJoinability
		onPointerEnter: () => void
		onPointerLeave: () => void
		onFillLaunch: () => void
		// onFollowUser fills the launch options to follow a user whose server
		// the presence hides.
		onFollowUser?: () => void
	} = $props()
	const previews = getProfilePreviews()

	const headingId = $derived(`${id}-heading`),
		isNotJoinable = $derived(joinability === UserJoinability.UserNotJoinable)
	const universeId = $derived(presence.universeId ?? 0),
		placeId = $derived(presence.placeId ?? 0),
		rootPlaceId = $derived(presence.rootPlaceId ?? 0)
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
			presenceTextClass(presence),
		]}
		id={headingId}>
		<Icon
			svg={presenceIcon(presence)}
			size={14}
			aria-hidden="true" />{presenceLabel(presence)}
	</h2>
	{#if universeId > 0 && placeId > 0}<PlaceHeader {universeId} {placeId} />{/if}
	<dl class="detail-grid" aria-label="Game identifiers">
		{#if placeId > 0}<div>
				<dt>Place ID</dt>
				{@render searchableId(placeId, "Show game", () =>
					previews.show({ kind: "Place", placeId }),
				)}
			</div>{/if}
		{#if universeId > 0}<div>
				<dt>Universe ID</dt>
				{@render searchableId(universeId, "Show the places of this game", () =>
					previews.show({
						kind: "Place",
						placeId: rootPlaceId || placeId,
						universeId,
					}),
				)}
			</div>{/if}
		{#if rootPlaceId > 0 && rootPlaceId !== placeId}<div>
				<dt>Root place ID</dt>
				{@render searchableId(rootPlaceId, "Show root place", () =>
					previews.show({ kind: "Place", placeId: rootPlaceId }),
				)}
			</div>{/if}
		{#if presence.userPresenceType === PresenceType.PresenceTypeInGame}<div
				class="col-span-full">
				<dt>Server ID</dt>
				{#if presence.serverId}<dd class="font-mono select-text">
						{presence.serverId}
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

{#snippet searchableId(
	placeOrUniverseId: number,
	searchLabel: string,
	onSearch: () => void,
)}
	<dd class="flex min-w-0 items-center gap-1">
		<span class="truncate font-mono select-text">{placeOrUniverseId}</span>
		<button
			class="button-ghost-sm"
			type="button"
			aria-label={searchLabel}
			data-tooltip={searchLabel}
			onclick={onSearch}>
			<Icon svg={magnifyingGlassIcon} size={14} aria-hidden="true" />
		</button>
	</dd>
{/snippet}

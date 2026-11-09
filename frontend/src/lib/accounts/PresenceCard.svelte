<script lang="ts">
	import magnifyingGlassIcon from "@phosphor-icons/core/regular/magnifying-glass.svg?raw"
	import signInIcon from "@phosphor-icons/core/regular/sign-in.svg?raw"
	import Icon from "../shared/Icon.svelte"
	import { PresenceType } from "../backend/bridge"
	import type { UserPresence } from "../backend/bridge"
	import PlaceHeader from "../games/PlaceHeader.svelte"
	import { floatMenu, menuIn, menuOut } from "../shared/presence"
	import { presenceIcon, presenceLabel, presenceTextClass } from "./account-model"

	let {
		id,
		anchor,
		presence,
		isJoinable,
		onPointerEnter,
		onPointerLeave,
		onFillLaunch,
		onSearchPlace,
		onSearchUniverse,
	}: {
		id: string
		anchor: HTMLElement
		presence: UserPresence
		isJoinable: boolean
		onPointerEnter: () => void
		onPointerLeave: () => void
		onFillLaunch: () => void
		onSearchPlace: (placeId: number) => void
		onSearchUniverse: (universeId: number) => void
	} = $props()

	const headingId = $derived(`${id}-heading`)
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
				{@render searchableId(placeId, "Find this place in Games", () =>
					onSearchPlace(placeId),
				)}
			</div>{/if}
		{#if universeId > 0}<div>
				<dt>Universe ID</dt>
				{@render searchableId(universeId, "Find places in this universe", () =>
					onSearchUniverse(universeId),
				)}
			</div>{/if}
		{#if rootPlaceId > 0 && rootPlaceId !== placeId}<div>
				<dt>Root place ID</dt>
				{@render searchableId(rootPlaceId, "Find the root place in Games", () =>
					onSearchPlace(rootPlaceId),
				)}
			</div>{/if}
		{#if presence.userPresenceType === PresenceType.PresenceTypeInGame}<div
				class="col-span-full">
				<dt>Job ID</dt>
				{#if presence.gameId}<dd class="font-mono select-text">
						{presence.gameId}
					</dd>
				{:else}<dd class="text-fg-muted">Not shared by Roblox</dd>{/if}
			</div>{/if}
	</dl>
	{#if isJoinable}
		<button class="button-primary" type="button" onclick={onFillLaunch}>
			<Icon svg={signInIcon} size={16} aria-hidden="true" />
			Fill launch options
		</button>
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

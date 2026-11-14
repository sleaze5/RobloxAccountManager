<script lang="ts">
	import circleNotchIcon from "@phosphor-icons/core/regular/circle-notch.svg?raw"
	import coatHangerIcon from "@phosphor-icons/core/regular/coat-hanger.svg?raw"
	import dotsThreeIcon from "@phosphor-icons/core/regular/dots-three.svg?raw"
	import floppyDiskIcon from "@phosphor-icons/core/regular/floppy-disk.svg?raw"
	import pencilSimpleIcon from "@phosphor-icons/core/regular/pencil-simple.svg?raw"
	import personArmsSpreadIcon from "@phosphor-icons/core/regular/person-arms-spread.svg?raw"
	import trashIcon from "@phosphor-icons/core/regular/trash.svg?raw"
	import { onMount } from "svelte"
	import type { AvatarOutfit } from "../backend/bridge"
	import AnchoredActionMenu from "../shared/AnchoredActionMenu.svelte"
	import type { MenuAction } from "../shared/AnchoredActionMenu.svelte"
	import Icon from "../shared/Icon.svelte"
	import ProfileImage from "../shared/ProfileImage.svelte"
	import type { AvatarState } from "./avatar-state.svelte"

	let {
		avatar,
		canEdit,
		username,
		onCreateOutfit,
		onOpenOutfit,
		onRenameOutfit,
		onDeleteOutfit,
	}: {
		avatar: AvatarState
		canEdit: boolean
		username: string
		onCreateOutfit: () => void
		onOpenOutfit: (outfit: AvatarOutfit) => void
		onRenameOutfit: (outfit: AvatarOutfit) => void
		onDeleteOutfit: (outfit: AvatarOutfit) => void
	} = $props()

	const skeletonTileIndexes = Array.from({ length: 8 }, (_unused, index) => index)

	let menu = $state<{ outfit: AvatarOutfit; trigger: HTMLElement } | null>(null)

	const outfits = $derived(avatar.outfits),
		isChanged = $derived(avatar.isChanged),
		menuActions = $derived.by((): MenuAction[] => {
			const outfit = menu?.outfit
			if (!outfit) return []
			return [
				...(isChanged
					? []
					: [
							{
								label: "Put on outfit",
								icon: personArmsSpreadIcon,
								onSelect: () => {
									menu = null
									void avatar.wearOutfit(outfit)
								},
							},
						]),
				{
					label: "Rename outfit…",
					icon: pencilSimpleIcon,
					onSelect: () => {
						menu = null
						onRenameOutfit(outfit)
					},
				},
				{
					label: "Delete outfit…",
					icon: trashIcon,
					danger: true,
					onSelect: () => {
						menu = null
						onDeleteOutfit(outfit)
					},
				},
			]
		})

	onMount(() => {
		if (!avatar.outfits) void avatar.loadOutfits()
	})
</script>

<div class="flex flex-col gap-4 p-3">
	<div class="flex min-h-7 items-center gap-2">
		<p class="min-w-0 flex-1 text-xs text-fg-muted">
			{#if canEdit && isChanged}
				Save or discard your changes to put on an outfit or save a new one.
			{:else if canEdit}
				Outfits save a look to put on later. Putting one on replaces the avatar.
			{:else}
				The outfits that @{username} saved.
			{/if}
		</p>
		{#if canEdit}
			<button
				class="button"
				type="button"
				disabled={isChanged || avatar.isSaving || avatar.isChangingOutfit}
				onclick={onCreateOutfit}>
				<Icon svg={floppyDiskIcon} size={16} aria-hidden="true" />
				Save as outfit…
			</button>
		{/if}
	</div>

	{#if avatar.outfitActionError}
		<p class="notice-error" role="alert">{avatar.outfitActionError}</p>
	{/if}

	{#if outfits === null && avatar.outfitsError}
		<div class="empty-state py-10" role="alert">
			<Icon svg={coatHangerIcon} size={28} aria-hidden="true" />
			<h2>Outfits are unavailable</h2>
			<p>{avatar.outfitsError}</p>
			<button
				class="button"
				type="button"
				onclick={() => void avatar.loadOutfits()}>Try again</button>
		</div>
	{:else if outfits === null}
		<div role="status" aria-busy="true">
			<span class="sr-only">Loading outfits</span>
			<ul
				class="grid grid-cols-3 gap-2 @lg:grid-cols-4 @2xl:grid-cols-5 @4xl:grid-cols-6"
				aria-hidden="true">
				{#each skeletonTileIndexes as index (index)}
					<li
						class="flex flex-col gap-1 rounded-lg border border-line-subtle bg-surface p-1">
						<span class="block aspect-square w-full skeleton rounded-sm"
						></span>
						<span class="flex px-1 pb-0.5 text-xs"
							><span class="skeleton-line w-2/3"></span></span>
					</li>
				{/each}
			</ul>
		</div>
	{:else if outfits.length === 0}
		<div class="empty-state py-10" role="status">
			<Icon svg={coatHangerIcon} size={28} aria-hidden="true" />
			<h2>No outfits</h2>
			<p>
				{canEdit
					? "Save the avatar as an outfit to put it on again later."
					: `@${username} hasn't saved any outfits.`}
			</p>
		</div>
	{:else}
		{#if avatar.outfitsError}
			<p class="notice-error" role="alert">{avatar.outfitsError}</p>
		{/if}
		<ul
			class="grid animate-enter grid-cols-3 gap-2 @lg:grid-cols-4 @2xl:grid-cols-5 @4xl:grid-cols-6"
			aria-label="Outfits"
			aria-busy={avatar.outfitsLoading}>
			{#each outfits as outfit (outfit.outfitId)}
				{@const isWearing = avatar.wearingOutfitId === outfit.outfitId}
				<li class="relative min-w-0">
					<button
						class="group/tile flex w-full min-w-0 flex-col gap-1 rounded-lg border border-line-subtle bg-surface p-1 text-left transition-colors duration-100 hover:border-line hover:bg-hover"
						type="button"
						data-tooltip="Show outfit details"
						data-tooltip-strong={outfit.name || "Unnamed outfit"}
						onclick={() => onOpenOutfit(outfit)}>
						<span
							class="relative grid aspect-square w-full place-items-center overflow-hidden rounded-sm bg-raised"
							aria-hidden="true">
							<span
								class="grid size-full place-items-center transition-transform duration-150 ease-out motion-safe:group-hover/tile:scale-105">
								<ProfileImage
									url={outfit.imageUrl}
									fallbackIcon={coatHangerIcon} />
							</span>
							{#if isWearing}
								<span
									class="absolute inset-0 grid animate-enter place-items-center bg-scrim">
									<Icon
										svg={circleNotchIcon}
										class="spinner text-fg"
										size={16} />
								</span>
							{/if}
						</span>
						<span
							class={[
								"block min-w-0 truncate px-1 pb-0.5 text-xs font-medium text-fg",
								canEdit && "pr-7",
							]}>{outfit.name || "Unnamed outfit"}</span>
					</button>
					{#if canEdit}
						<button
							class="absolute right-1 bottom-0.5 button-ghost-sm"
							type="button"
							aria-label={`Actions for ${outfit.name || "outfit"}`}
							aria-haspopup="menu"
							aria-expanded={menu?.outfit.outfitId === outfit.outfitId}
							data-open={menu?.outfit.outfitId === outfit.outfitId ||
								undefined}
							data-tooltip="Show outfit actions"
							disabled={avatar.isChangingOutfit ||
								avatar.wearingOutfitId !== 0}
							onclick={(event) => {
								const trigger = event.currentTarget
								menu =
									menu?.outfit.outfitId === outfit.outfitId
										? null
										: { outfit, trigger }
							}}>
							<Icon svg={dotsThreeIcon} size={14} aria-hidden="true" />
						</button>
					{/if}
					{#if isWearing}
						<span class="sr-only" role="status"
							>Putting on {outfit.name}…</span>
					{/if}
				</li>
			{/each}
		</ul>
		{#if avatar.outfitsNextCursor}
			<div class="flex min-h-12 items-center justify-center">
				<button
					class="button"
					type="button"
					aria-busy={avatar.outfitsLoadingMore}
					disabled={avatar.outfitsLoading || avatar.outfitsLoadingMore}
					onclick={() => void avatar.loadMoreOutfits()}
					>{avatar.outfitsLoadingMore ? "Loading…" : "Load more"}</button>
			</div>
		{/if}
	{/if}
</div>

{#if menu && menuActions.length}
	<AnchoredActionMenu
		id="avatar-outfit-menu"
		label="Outfit actions"
		anchor={menu.trigger}
		actions={menuActions}
		onClose={() => (menu = null)} />
{/if}

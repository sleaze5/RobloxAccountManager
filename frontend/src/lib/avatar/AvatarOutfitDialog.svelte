<script lang="ts">
	import coatHangerIcon from "@phosphor-icons/core/regular/coat-hanger.svg?raw"
	import cubeIcon from "@phosphor-icons/core/regular/cube.svg?raw"
	import personArmsSpreadIcon from "@phosphor-icons/core/regular/person-arms-spread.svg?raw"
	import xIcon from "@phosphor-icons/core/regular/x.svg?raw"
	import { onMount } from "svelte"
	import { AvatarType } from "../backend/bridge"
	import type { AvatarAsset, AvatarOutfit } from "../backend/bridge"
	import Dialog from "../shared/Dialog.svelte"
	import FrameUnavailable from "../shared/FrameUnavailable.svelte"
	import Icon from "../shared/Icon.svelte"
	import ImageViewToggle from "../shared/ImageViewToggle.svelte"
	import type { ImageView } from "../shared/ImageViewToggle.svelte"
	import AnimationControls from "../model-viewer/AnimationControls.svelte"
	import ModelFrame from "../model-viewer/ModelFrame.svelte"
	import type { AnimationPlayback, ModelSubject } from "../model-viewer/model-subject"
	import ProfileImage from "../shared/ProfileImage.svelte"
	import Skeleton from "../shared/Skeleton.svelte"
	import SkeletonTile from "../shared/SkeletonTile.svelte"
	import AvatarAdjustmentSummary from "./AvatarAdjustmentSummary.svelte"
	import AvatarBodyColorFacts from "./AvatarBodyColorFacts.svelte"
	import AvatarBodyFigure from "./AvatarBodyFigure.svelte"
	import AvatarItemGroups from "./AvatarItemGroups.svelte"
	import { bodyScales } from "./avatar-draft"
	import type { AvatarState } from "./avatar-state.svelte"

	let {
		avatar,
		outfit,
		canWear,
		onOpenItem,
		onClose,
	}: {
		avatar: AvatarState
		outfit: AvatarOutfit
		// canWear is true when the outfit belongs to the account itself.
		canWear: boolean
		onOpenItem: (asset: AvatarAsset) => void
		onClose: () => void
	} = $props()

	const percent = new Intl.NumberFormat("en-US", {
			style: "percent",
			maximumFractionDigits: 0,
		}),
		exactNumbers = new Intl.NumberFormat("en-US"),
		metaClass =
			"flex min-w-0 flex-wrap items-center gap-1 font-mono text-2xs wrap-anywhere text-fg-muted",
		sectionHeadingClass = "text-xs font-semibold text-fg-muted"

	let view = $state<ImageView>("2D"),
		playback = $state<AnimationPlayback>()

	// undefined while the details load, and null when they failed.
	const details = $derived(avatar.outfitDetailsById.get(outfit.outfitId)),
		name = $derived(outfit.name || "Unnamed outfit"),
		assets = $derived(details?.assets ?? []),
		isWearing = $derived(avatar.wearingOutfitId === outfit.outfitId),
		imageUrl = $derived(details?.imageUrl || outfit.imageUrl),
		// The 3D view builds the outfit, so that it shows before it is put on.
		outfitSubject = $derived<ModelSubject | null>(
			details ? { kind: "avatar", look: details } : null,
		)

	async function wear(): Promise<void> {
		if (await avatar.wearOutfit(outfit)) onClose()
	}

	function close(): void {
		if (!isWearing) onClose()
	}

	onMount(() => {
		avatar.clearOutfitActionError()
		void avatar.loadOutfitDetails(outfit)
	})
</script>

<svelte:window
	onkeydown={(event) => {
		if (event.key === "Escape") {
			event.preventDefault()
			close()
		}
	}} />

<Dialog
	class="max-w-3xl text-left"
	aria-labelledby="avatar-outfit-title"
	onBackdropClick={close}>
	<div class="@container flex flex-col gap-4">
		<div class="flex items-start gap-4">
			<div class="flex w-64 shrink-0 flex-col gap-2">
				<div
					class="relative grid size-64 place-items-center overflow-hidden rounded-lg border border-line-subtle bg-canvas">
					{#if view === "3D" && details === null}
						<FrameUnavailable
							icon={cubeIcon}
							message="The 3D view needs the outfit details, which couldn't load." />
					{:else if view === "3D"}
						<ModelFrame
							accountId={avatar.accountId}
							subject={outfitSubject}
							label={`the outfit ${name}`}
							bind:playback />
					{:else}
						<span
							class="relative grid size-full place-items-center"
							aria-hidden="true">
							<ProfileImage
								url={imageUrl}
								fallbackIcon={coatHangerIcon} />
						</span>
					{/if}
				</div>
				<ImageViewToggle bind:view />
				{#if view === "3D" && details !== null}
					<div class="animate-enter">
						<AnimationControls {playback} />
					</div>
				{/if}
			</div>

			<div class="flex min-w-0 flex-1 flex-col gap-1">
				<div class="flex items-start gap-3">
					<h2
						id="avatar-outfit-title"
						class="min-w-0 flex-1 leading-7 tracking-tight wrap-anywhere">
						{name}
					</h2>
					<div class="flex shrink-0 items-center gap-1">
						{#if canWear}
							<button
								class="button-primary"
								type="button"
								aria-busy={isWearing}
								disabled={!avatar.canWearOutfit ||
									avatar.isChangingOutfit}
								onclick={() => void wear()}>
								<Icon
									svg={personArmsSpreadIcon}
									size={16}
									aria-hidden="true" />
								{isWearing ? "Putting on…" : "Put on outfit"}
							</button>
						{/if}
						<button
							class="ml-1 button-close"
							type="button"
							aria-label="Close outfit details"
							disabled={isWearing}
							onclick={close}>
							<Icon svg={xIcon} size={14} aria-hidden="true" />
						</button>
					</div>
				</div>
				<div class={metaClass}>
					<span class="select-text">ID: {outfit.outfitId}</span>
					{#if details}
						<span class="text-fg-faint" aria-hidden="true">-</span>
						<span>{details.avatarType}</span>
						<span class="text-fg-faint" aria-hidden="true">-</span>
						<span
							>{exactNumbers.format(assets.length)}
							{assets.length === 1 ? "item" : "items"}</span>
					{/if}
				</div>
				{#if canWear && avatar.isChanged}
					<p class="mt-2 text-xs text-fg-muted">
						Save or discard your avatar changes to put on this outfit.
					</p>
				{:else if canWear}
					<p class="mt-2 text-xs text-fg-muted">
						Putting on the outfit replaces the avatar. Emotes stay as they
						are.
					</p>
				{/if}
				{#if avatar.outfitActionError}
					<p class="mt-2 notice-error" role="alert">
						{avatar.outfitActionError}
					</p>
				{/if}
			</div>
		</div>

		{#if details === undefined}
			<div class="flex flex-col gap-2" role="status" aria-busy="true">
				<span class="sr-only">Loading the outfit</span>
				<Skeleton className="w-24" label="Loading the items" />
				<ul
					class="grid grid-cols-3 gap-2 @lg:grid-cols-4 @2xl:grid-cols-5"
					aria-hidden="true">
					{#each Array.from({ length: 5 }, (_unused, index) => index) as index (index)}
						<SkeletonTile />
					{/each}
				</ul>
			</div>
		{:else if details === null}
			<div class="flex items-center gap-2 notice-error" role="alert">
				<span class="min-w-0 flex-1"
					>The outfit details could not be loaded.</span>
				<button
					class="button"
					type="button"
					onclick={() => void avatar.loadOutfitDetails(outfit)}
					>Try again</button>
			</div>
		{:else}
			{#if assets.length === 0}
				<p class="text-xs text-fg-muted">This outfit has no items.</p>
			{:else}
				<AvatarItemGroups
					{assets}
					headingIdPrefix="avatar-outfit-group"
					{onOpenItem} />
			{/if}

			<AvatarAdjustmentSummary {assets} headingIdPrefix="avatar-outfit-adjust" />

			<section class="flex flex-col gap-3" aria-labelledby="avatar-outfit-body">
				<h3 id="avatar-outfit-body" class={sectionHeadingClass}>Body</h3>
				<div class="flex flex-wrap items-start gap-6 @max-md:flex-col">
					<div class="rounded-lg border border-line-subtle bg-canvas p-4">
						<AvatarBodyFigure bodyColors={details.bodyColors} />
					</div>
					<dl class="facts min-w-0 flex-1">
						<div>
							<dt>Avatar type</dt>
							<dd>
								<span>{details.avatarType}</span>
								{#if details.avatarType === AvatarType.AvatarR6}
									<small>Scales apply only to R15</small>
								{/if}
							</dd>
						</div>
						{#each bodyScales as row (row.scale)}
							<div>
								<dt>{row.label}</dt>
								<dd>
									<span class="font-mono"
										>{percent.format(
											details.scales[row.scale],
										)}</span>
								</dd>
							</div>
						{/each}
					</dl>
				</div>
				<AvatarBodyColorFacts
					bodyColors={details.bodyColors}
					paletteColorByHex={avatar.paletteColorByHex}
					isPaletteLoaded={!!avatar.rules} />
			</section>
		{/if}
	</div>
</Dialog>

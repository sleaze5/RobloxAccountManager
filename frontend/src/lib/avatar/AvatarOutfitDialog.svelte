<script lang="ts">
	import coatHangerIcon from "@phosphor-icons/core/regular/coat-hanger.svg?raw"
	import personArmsSpreadIcon from "@phosphor-icons/core/regular/person-arms-spread.svg?raw"
	import xIcon from "@phosphor-icons/core/regular/x.svg?raw"
	import { onMount } from "svelte"
	import { AvatarType } from "../backend/bridge"
	import type { AvatarAsset, AvatarOutfit, AvatarScales } from "../backend/bridge"
	import Dialog from "../shared/Dialog.svelte"
	import Icon from "../shared/Icon.svelte"
	import ProfileImage from "../shared/ProfileImage.svelte"
	import Skeleton from "../shared/Skeleton.svelte"
	import AvatarBodyColorFacts from "./AvatarBodyColorFacts.svelte"
	import AvatarBodyFigure from "./AvatarBodyFigure.svelte"
	import AvatarItemGroups from "./AvatarItemGroups.svelte"
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

	const scaleRows: { scale: keyof AvatarScales; label: string }[] = [
			{ scale: "height", label: "Height" },
			{ scale: "width", label: "Width" },
			{ scale: "head", label: "Head" },
			{ scale: "proportion", label: "Proportions" },
			{ scale: "bodyType", label: "Body type" },
		],
		percent = new Intl.NumberFormat("en-US", {
			style: "percent",
			maximumFractionDigits: 0,
		}),
		exactNumbers = new Intl.NumberFormat("en-US"),
		metaClass =
			"flex min-w-0 flex-wrap items-center gap-1 font-mono text-2xs wrap-anywhere text-fg-muted",
		sectionHeadingClass = "text-xs font-semibold text-fg-muted"

	// undefined while the details load, and null when they failed.
	const details = $derived(avatar.outfitDetailsById.get(outfit.outfitId)),
		name = $derived(outfit.name || "Unnamed outfit"),
		assets = $derived(details?.assets ?? []),
		isWearing = $derived(avatar.wearingOutfitId === outfit.outfitId),
		imageUrl = $derived(details?.imageUrl || outfit.imageUrl)

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
			<span
				class="relative grid size-48 shrink-0 place-items-center overflow-hidden rounded-lg border border-line-subtle bg-canvas"
				aria-hidden="true">
				<span class="absolute bottom-4 h-2.5 w-24 rounded-full bg-hover-overlay"
				></span>
				<span class="relative grid size-full place-items-center">
					<ProfileImage url={imageUrl} fallbackIcon={coatHangerIcon} />
				</span>
			</span>

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
						<li
							class="flex flex-col gap-1 rounded-lg border border-line-subtle bg-surface p-1">
							<span class="block aspect-square w-full skeleton rounded-sm"
							></span>
							<span class="flex px-1 pb-0.5 text-xs"
								><span class="skeleton-line w-3/4"></span></span>
						</li>
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
						{#each scaleRows as row (row.scale)}
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

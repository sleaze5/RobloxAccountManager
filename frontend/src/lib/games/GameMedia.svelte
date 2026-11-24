<script lang="ts">
	import arrowSquareOutIcon from "@phosphor-icons/core/regular/arrow-square-out.svg?raw"
	import imageIcon from "@phosphor-icons/core/regular/image.svg?raw"
	import playFillIcon from "@phosphor-icons/core/fill/play-fill.svg?raw"
	import { accountBackend, GameMediaKind } from "../backend/bridge"
	import type { GameMedia } from "../backend/bridge"
	import Icon from "../shared/Icon.svelte"
	import TileCarousel from "../shared/TileCarousel.svelte"
	import GameMediaDialog from "./GameMediaDialog.svelte"

	// GameMedia shows the images and videos of a game in a carousel of 16:9
	// tiles. An image or a Roblox video opens in the media dialog, and a
	// YouTube video opens in the browser.
	let {
		universeId,
		gameName,
		refreshCount,
		onOpenLink,
	}: {
		universeId: number
		gameName: string
		// refreshCount changes when the game details refresh, which reads the
		// media again.
		refreshCount: number
		onOpenLink: (url: string) => void
	} = $props()

	const headingId = $props.id(),
		skeletonTileIndexes = [0, 1, 2, 3],
		tileWidthClass = "w-64 @4xl:w-80"

	let media = $state.raw<GameMedia[] | null>(null),
		hasFailed = $state(false),
		openedIndex = $state<number | null>(null)

	$effect(() => {
		const requestedUniverseId = universeId
		void refreshCount
		media = null
		hasFailed = false
		openedIndex = null
		let isCurrent = true
		const request = accountBackend.ListGameMedia(requestedUniverseId)
		void (async () => {
			try {
				const listedMedia = await request
				if (isCurrent) media = listedMedia ?? []
			} catch {
				// The backend logs the failure, and the section shows that the
				// media is unavailable.
				if (isCurrent) hasFailed = true
			}
		})()
		return () => {
			isCurrent = false
			void request.cancel()
		}
	})

	// The dialog steps through the images and Roblox videos. YouTube videos
	// open in the browser instead.
	const viewableMedia = $derived(
		(media ?? []).filter(
			(entry) => entry.kind !== GameMediaKind.MediaKindYouTubeVideo,
		),
	)

	function tileLabel(entry: GameMedia): string {
		if (entry.kind === GameMediaKind.MediaKindYouTubeVideo)
			return `Open ${entry.videoTitle || "video"} on YouTube`
		const position = viewableMedia.indexOf(entry) + 1,
			noun = entry.kind === GameMediaKind.MediaKindVideo ? "video" : "image"
		return `Show ${noun} ${position} of ${viewableMedia.length}`
	}

	function openEntry(entry: GameMedia): void {
		if (entry.kind === GameMediaKind.MediaKindYouTubeVideo)
			onOpenLink(entry.videoPageUrl)
		else openedIndex = viewableMedia.indexOf(entry)
	}
</script>

{#if media?.length}
	<TileCarousel id={headingId} label="Media">
		{#snippet heading()}
			Media<span class="font-mono text-2xs font-normal text-fg-faint"
				>({media?.length})</span>
		{/snippet}
		{#each media as entry, index (index)}
			<li class={["shrink-0", tileWidthClass]}>
				<button
					class="group/tile relative grid aspect-video w-full place-items-center overflow-hidden rounded-lg border border-line-strong bg-raised text-fg-faint transition-colors duration-100 hover:border-line-focus"
					type="button"
					aria-label={tileLabel(entry)}
					data-tooltip={tileLabel(entry)}
					onclick={() => openEntry(entry)}>
					{#if entry.imageUrl}
						<img
							class="size-full object-cover transition-transform duration-150 ease-out motion-safe:group-hover/tile:scale-105"
							src={entry.imageUrl}
							alt={entry.altText}
							loading="lazy"
							decoding="async"
							draggable="false"
							referrerpolicy="no-referrer" />
					{:else}
						<Icon svg={imageIcon} size={28} aria-hidden="true" />
					{/if}
					{#if entry.kind !== GameMediaKind.MediaKindImage}
						<span
							class="absolute inset-0 m-auto grid size-10 place-items-center rounded-full bg-scrim text-fg-bright"
							aria-hidden="true">
							<Icon svg={playFillIcon} size={18} aria-hidden="true" />
						</span>
						<span
							class="absolute bottom-1.5 left-1.5 inline-flex h-5 max-w-4/5 items-center gap-1 rounded-sm bg-scrim px-1.5 text-2xs font-medium text-fg-bright"
							aria-hidden="true">
							{#if entry.kind === GameMediaKind.MediaKindYouTubeVideo}
								<Icon
									svg={arrowSquareOutIcon}
									size={12}
									aria-hidden="true" />
								<span class="truncate"
									>{entry.videoTitle || "YouTube"}</span>
							{:else}
								<span class="truncate">Video</span>
							{/if}
						</span>
					{/if}
				</button>
			</li>
		{/each}
	</TileCarousel>
{:else}
	<section class="flex flex-col gap-2" aria-labelledby={headingId}>
		<h3
			id={headingId}
			class="flex h-6 items-center text-xs font-semibold text-fg-muted">
			Media
		</h3>
		{#if media}
			<p class="text-xs text-fg-muted">None</p>
		{:else if hasFailed}
			<p class="text-xs text-fg-muted">Unavailable</p>
		{:else}
			<ul class="flex gap-2 overflow-x-hidden" aria-hidden="true">
				{#each skeletonTileIndexes as index (index)}
					<li class={["shrink-0", tileWidthClass]}>
						<span class="block aspect-video w-full skeleton rounded-lg"
						></span>
					</li>
				{/each}
			</ul>
			<span class="sr-only" role="status">Loading media…</span>
		{/if}
	</section>
{/if}

{#if openedIndex !== null && viewableMedia.length}
	<GameMediaDialog
		{gameName}
		media={viewableMedia}
		initialIndex={openedIndex}
		onClose={() => (openedIndex = null)} />
{/if}

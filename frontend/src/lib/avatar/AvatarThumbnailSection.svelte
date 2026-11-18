<script lang="ts">
	import imageIcon from "@phosphor-icons/core/regular/image.svg?raw"
	import type {
		AvatarScaleLimit,
		AvatarThumbnailCamera,
		AvatarThumbnailChange,
		AvatarThumbnailCustomization,
	} from "../backend/bridge"
	import type { AnimationLibrary } from "../model-viewer/model-subject"
	import Icon from "../shared/Icon.svelte"
	import Select from "../shared/Select.svelte"
	import { rangeShareOf, roundedToStep } from "./range-slider"

	// AvatarThumbnailSection shows how Roblox poses and frames one kind of
	// avatar image, with the images that it frames. With canEdit, it edits
	// the customization and saves it with onSave.
	let {
		title,
		description,
		customization,
		images,
		canEdit,
		isSaving,
		isLocked,
		library,
		onSave,
	}: {
		title: string
		description: string
		customization: AvatarThumbnailCustomization
		images: { label: string; url: string }[]
		canEdit: boolean
		isSaving: boolean
		// Whether another customization saves, which keeps this one unchanged.
		isLocked: boolean
		library: AnimationLibrary
		onSave: (change: AvatarThumbnailChange) => void
	} = $props()

	type CameraSetting = keyof AvatarThumbnailCamera

	interface PoseOption {
		value: number
		label: string
		description: string
		group?: string
		image?: string
	}

	const sectionId = $props.id(),
		degrees = new Intl.NumberFormat("en-US", { maximumFractionDigits: 1 }),
		factor = new Intl.NumberFormat("en-US", {
			minimumFractionDigits: 2,
			maximumFractionDigits: 2,
		})

	// The edited customization, which starts from the saved one and returns to
	// it when Roblox lists a new one.
	let emoteAssetId = $state(0),
		isCameraCustomized = $state(false),
		camera = $state<AvatarThumbnailCamera>({
			fieldOfViewDegrees: 0,
			yRotationDegrees: 0,
			distanceScale: 0,
		}),
		failedImageUrls = $state.raw(new Set<string>())

	$effect.pre(() => {
		emoteAssetId = customization.emote?.itemId ?? 0
		isCameraCustomized = customization.isCameraCustomized
		camera = { ...customization.camera }
	})

	const limits = $derived(customization.cameraLimits),
		cameraRows = $derived(
			(
				[
					{
						setting: "fieldOfViewDegrees",
						label: "Field of view",
						limit: limits.fieldOfViewDegrees,
						format: (value: number) => `${degrees.format(value)}°`,
					},
					{
						setting: "yRotationDegrees",
						label: "Rotation",
						limit: limits.yRotationDegrees,
						format: (value: number) => `${degrees.format(value)}°`,
					},
					{
						setting: "distanceScale",
						label: "Distance",
						limit: limits.distanceScale,
						format: (value: number) => `${factor.format(value)}×`,
					},
				] satisfies {
					setting: CameraSetting
					label: string
					limit: AvatarScaleLimit
					format: (value: number) => string
				}[]
			).filter((row) => row.limit.max > row.limit.min),
		),
		ownedChoices = $derived([
			...(library.ownedEmotes.choices ?? []),
			...(library.ownedAnimations.choices ?? []),
		]),
		ownedError = $derived(
			library.ownedEmotes.error || library.ownedAnimations.error,
		),
		poseOptions = $derived.by(() => {
			const savedEmote = customization.emote,
				options: PoseOption[] = [
					{ value: 0, label: "None", description: "Idle pose" },
				]
			if (
				savedEmote &&
				!ownedChoices.some((choice) => choice.assetId === savedEmote.itemId)
			) {
				options.push({
					value: savedEmote.itemId,
					label: savedEmote.name || `Emote ${savedEmote.itemId}`,
					description: "Saved",
					image: savedEmote.imageUrl,
				})
			}
			for (const choice of ownedChoices) {
				if (choice.assetId === undefined) continue
				options.push({
					value: choice.assetId,
					label: choice.label,
					description: choice.detail ?? "Emote",
					group:
						choice.group === "ownedEmote"
							? "Owned emotes"
							: "Owned animations",
					image: choice.imageUrl ?? "",
				})
			}
			return options
		}),
		isChanged = $derived(
			emoteAssetId !== (customization.emote?.itemId ?? 0) ||
				isCameraCustomized !== customization.isCameraCustomized ||
				(isCameraCustomized &&
					(Object.keys(camera) as CameraSetting[]).some(
						(setting) => camera[setting] !== customization.camera[setting],
					)),
		),
		isDisabled = $derived(isSaving || isLocked),
		emoteLabel = $derived(
			customization.emote
				? customization.emote.name || `Emote ${customization.emote.itemId}`
				: "None",
		)

	function setCamera(setting: CameraSetting, value: number, step: number): void {
		camera = { ...camera, [setting]: roundedToStep(value, step) }
	}

	function discard(): void {
		emoteAssetId = customization.emote?.itemId ?? 0
		isCameraCustomized = customization.isCameraCustomized
		camera = { ...customization.camera }
	}
</script>

<section class="settings-section" aria-labelledby={`${sectionId}-heading`}>
	<h3 id={`${sectionId}-heading`}>{title}</h3>
	<p class="text-xs text-fg-muted">{description}</p>

	<div class="flex flex-wrap gap-2">
		{#each images as image (image.label)}
			<figure class="flex flex-col items-center gap-1">
				<div
					class="relative grid size-28 place-items-center overflow-hidden rounded-lg border border-line-subtle bg-canvas">
					{#if image.url && !failedImageUrls.has(image.url)}
						{#key image.url}
							<img
								class="size-full animate-pop object-contain"
								src={image.url}
								alt={image.label}
								decoding="async"
								draggable="false"
								referrerpolicy="no-referrer"
								onerror={() =>
									(failedImageUrls = new Set([
										...failedImageUrls,
										image.url,
									]))} />
						{/key}
					{:else}
						<Icon
							svg={imageIcon}
							class="text-fg-faint"
							size={28}
							aria-label={`${image.label} unavailable`} />
					{/if}
				</div>
				<figcaption class="text-2xs text-fg-muted">{image.label}</figcaption>
			</figure>
		{/each}
	</div>

	{#if canEdit}
		<div class="settings-rows">
			<div class="settings-row">
				<span class="w-24 shrink-0 text-xs text-fg-muted" aria-hidden="true"
					>Pose</span>
				<Select
					label={`${title} pose`}
					value={emoteAssetId}
					options={poseOptions}
					disabled={isDisabled}
					className="min-w-0 flex-1"
					onChange={(assetId) => (emoteAssetId = assetId)} />
			</div>
			<div class="settings-row">
				<div class="settings-row-copy">
					<strong id={`${sectionId}-camera`}>Custom camera</strong>
					<span id={`${sectionId}-camera-description`}
						>Off uses the standard Roblox framing.</span>
				</div>
				<label class="switch">
					<input
						type="checkbox"
						role="switch"
						aria-labelledby={`${sectionId}-camera`}
						aria-describedby={`${sectionId}-camera-description`}
						checked={isCameraCustomized}
						disabled={isDisabled}
						onchange={(event) =>
							(isCameraCustomized = event.currentTarget.checked)} />
					<span aria-hidden="true"></span>
				</label>
			</div>
			{#each cameraRows as row (row.setting)}
				{@const value = camera[row.setting]}
				<div class="settings-row">
					<span
						class="w-24 shrink-0 text-xs text-fg-muted"
						id={`${sectionId}-${row.setting}`}>{row.label}</span>
					<input
						class="range flex-1"
						type="range"
						min={row.limit.min}
						max={row.limit.max}
						step={row.limit.increment}
						{value}
						style:--range-fill={`${rangeShareOf(value, row.limit.min, row.limit.max) * 100}%`}
						aria-labelledby={`${sectionId}-${row.setting}`}
						aria-valuetext={row.format(value)}
						disabled={!isCameraCustomized || isDisabled}
						oninput={(event) =>
							setCamera(
								row.setting,
								event.currentTarget.valueAsNumber,
								row.limit.increment,
							)} />
					<span
						class="w-12 shrink-0 text-right font-mono text-xs text-fg tabular-nums"
						>{row.format(value)}</span>
				</div>
			{/each}
		</div>
		{#if ownedError}
			<p
				class="flex flex-wrap items-center gap-x-2 text-xs text-fg-muted"
				role="alert">
				<span>{ownedError}</span>
				<button
					class="button-text"
					type="button"
					onclick={() => library.loadOwned()}>Try again</button>
			</p>
		{/if}
		{#if isChanged}
			<div class="flex animate-enter justify-end gap-2">
				<button
					class="button"
					type="button"
					disabled={isDisabled}
					onclick={discard}>Discard</button>
				<button
					class="button"
					type="button"
					aria-busy={isSaving}
					disabled={isDisabled}
					onclick={() =>
						onSave({
							type: customization.type,
							emoteAssetId,
							isCameraCustomized,
							camera,
						})}>{isSaving ? "Saving…" : "Save thumbnail"}</button>
			</div>
		{/if}
	{:else}
		<dl class="facts">
			<div>
				<dt>Pose</dt>
				<dd class="select-text">{emoteLabel}</dd>
			</div>
			<div>
				<dt>Camera</dt>
				<dd>{customization.isCameraCustomized ? "Custom" : "Standard"}</dd>
			</div>
			{#each cameraRows as row (row.setting)}
				<div data-tone={customization.isCameraCustomized ? undefined : "muted"}>
					<dt>{row.label}</dt>
					<dd class="font-mono tabular-nums">
						{customization.isCameraCustomized
							? row.format(customization.camera[row.setting])
							: "N/A"}
					</dd>
				</div>
			{/each}
		</dl>
	{/if}
</section>

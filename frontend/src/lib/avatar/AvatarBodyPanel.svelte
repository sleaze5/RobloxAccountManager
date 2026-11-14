<script lang="ts">
	import checkIcon from "@phosphor-icons/core/regular/check.svg?raw"
	import { AvatarType } from "../backend/bridge"
	import type { AvatarScaleLimit, AvatarScales } from "../backend/bridge"
	import Icon from "../shared/Icon.svelte"
	import Skeleton from "../shared/Skeleton.svelte"
	import { replayValidationNudge } from "../shared/presence"
	import AvatarBodyFigure from "./AvatarBodyFigure.svelte"
	import { bodyParts } from "./avatar-draft"
	import type { AvatarDraft, BodyPart } from "./avatar-draft"
	import type { AvatarState } from "./avatar-state.svelte"

	let { avatar, canEdit }: { avatar: AvatarState; canEdit: boolean } = $props()

	const scaleRows: {
			scale: keyof AvatarScales
			label: string
			isBodyShape: boolean
		}[] = [
			{ scale: "height", label: "Height", isBodyShape: false },
			{ scale: "width", label: "Width", isBodyShape: false },
			{ scale: "head", label: "Head", isBodyShape: false },
			{ scale: "proportion", label: "Proportions", isBodyShape: true },
			{ scale: "bodyType", label: "Body type", isBodyShape: true },
		],
		percent = new Intl.NumberFormat("en-US", {
			style: "percent",
			maximumFractionDigits: 0,
		}),
		hexColorPattern = /^#?([0-9a-f]{6})$/i

	let selectedParts = $state(new Set<BodyPart>(bodyParts.map(({ part }) => part))),
		hexInput = $state<HTMLInputElement | undefined>(undefined),
		hexText = $state(""),
		isHexTouched = $state(false)

	const draft = $derived(avatar.draft),
		rules = $derived(avatar.rules),
		isR15 = $derived(draft?.avatarType === AvatarType.AvatarR15),
		shownScaleRows = $derived(
			scaleRows.filter(
				(row) => !canEdit || !row.isBodyShape || rules?.isBodyShapeEnabled,
			),
		),
		// The color that every selected part shares, or empty.
		selectedColor = $derived.by(() => {
			if (!draft) return ""
			const colors = new Set(
				[...selectedParts].map((part) => draft.bodyColors[part]),
			)
			return colors.size === 1 ? [...colors][0] : ""
		}),
		isHexValid = $derived(hexColorPattern.test(hexText.trim())),
		isAllSelected = $derived(selectedParts.size === bodyParts.length)

	$effect(() => {
		hexText = selectedColor
		isHexTouched = false
	})

	function limitOf(scale: keyof AvatarScales): AvatarScaleLimit | undefined {
		return rules?.scaleLimits[scale]
	}

	function fillOf(value: number, limit: AvatarScaleLimit): string {
		const share =
			limit.max > limit.min ? (value - limit.min) / (limit.max - limit.min) : 0
		return `${Math.min(Math.max(share, 0), 1) * 100}%`
	}

	// setScale rounds to the increment, because range inputs add floating point
	// noise such as 0.9500000001.
	function setScale(
		scale: keyof AvatarScales,
		rawValue: number,
		limit: AvatarScaleLimit,
	): void {
		const increment = limit.increment > 0 ? limit.increment : 0.01,
			value = Number((Math.round(rawValue / increment) * increment).toFixed(4))
		avatar.changeDraft((current) => ({
			...current,
			scales: { ...current.scales, [scale]: value },
		}))
	}

	function setColor(hexColor: string): void {
		avatar.changeDraft((current) => {
			const bodyColors = { ...current.bodyColors }
			for (const part of selectedParts) bodyColors[part] = hexColor
			return { ...current, bodyColors }
		})
	}

	function selectPart(part: BodyPart, isAdding: boolean): void {
		if (!isAdding) {
			selectedParts = new Set([part])
			return
		}
		const parts = new Set(selectedParts)
		if (parts.has(part) && parts.size > 1) parts.delete(part)
		else parts.add(part)
		selectedParts = parts
	}

	function applyHexText(): void {
		isHexTouched = true
		const match = hexColorPattern.exec(hexText.trim())
		if (!match) {
			replayValidationNudge(hexInput)
			return
		}
		setColor(match[1].toUpperCase())
	}

	function setAvatarType(avatarType: AvatarType): void {
		avatar.changeDraft((current): AvatarDraft => ({ ...current, avatarType }))
	}
</script>

{#if draft}
	<div class="flex flex-col gap-3 p-3">
		<section class="settings-section" aria-labelledby="avatar-type-heading">
			<h3 id="avatar-type-heading">Avatar type</h3>
			<div class="settings-rows">
				<div class="settings-row">
					<div class="settings-row-copy">
						<strong>{isR15 ? "R15" : "R6"}</strong>
						<span
							>{isR15
								? "15 parts that bend at the elbows and knees. Scales apply."
								: "6 rigid parts, as in classic Roblox. Scales don't apply."}</span>
					</div>
					{#if canEdit}
						<div
							class="segmented w-32"
							role="group"
							aria-label="Avatar type">
							{#each [AvatarType.AvatarR6, AvatarType.AvatarR15] as avatarType (avatarType)}
								<button
									type="button"
									aria-pressed={draft.avatarType === avatarType}
									onclick={() => setAvatarType(avatarType)}
									>{avatarType}</button>
							{/each}
						</div>
					{/if}
				</div>
			</div>
		</section>

		<section class="settings-section" aria-labelledby="avatar-scale-heading">
			<h3 id="avatar-scale-heading">Scale</h3>
			{#if canEdit && !isR15}
				<p class="notice-note">Switch to R15 to change the scales.</p>
			{/if}
			<div class="settings-rows">
				{#each shownScaleRows as row (row.scale)}
					{@const value = draft.scales[row.scale]}
					{@const limit = limitOf(row.scale)}
					<div class="settings-row">
						<span
							class="w-24 shrink-0 text-xs text-fg-muted"
							id={`avatar-scale-${row.scale}`}>{row.label}</span>
						{#if canEdit && limit}
							<input
								class="range flex-1"
								type="range"
								min={limit.min}
								max={limit.max}
								step={limit.increment > 0 ? limit.increment : 0.01}
								{value}
								style:--range-fill={fillOf(value, limit)}
								aria-labelledby={`avatar-scale-${row.scale}`}
								aria-valuetext={percent.format(value)}
								disabled={!isR15 || avatar.isSaving}
								oninput={(event) =>
									setScale(
										row.scale,
										event.currentTarget.valueAsNumber,
										limit,
									)} />
						{:else}
							<span class="flex-1"></span>
						{/if}
						<span
							class="w-12 shrink-0 text-right font-mono text-xs text-fg tabular-nums"
							>{percent.format(value)}</span>
					</div>
				{/each}
			</div>
		</section>

		<section class="settings-section" aria-labelledby="avatar-colors-heading">
			<h3 id="avatar-colors-heading">Body colors</h3>
			<div class="flex flex-wrap items-start gap-6 @max-md:flex-col">
				<div class="rounded-lg border border-line-subtle bg-canvas p-4">
					{#if canEdit}
						<AvatarBodyFigure
							bodyColors={draft.bodyColors}
							{selectedParts}
							onSelectPart={selectPart} />
					{:else}
						<AvatarBodyFigure bodyColors={draft.bodyColors} />
					{/if}
				</div>
				{#if canEdit}
					<div class="flex min-w-0 flex-1 flex-col gap-3">
						<div class="flex items-center gap-2">
							<p
								class="min-w-0 flex-1 text-xs text-fg-muted"
								aria-live="polite">
								{isAllSelected
									? "Coloring every part. Select a part to color only it."
									: `Coloring ${bodyParts
											.filter(({ part }) =>
												selectedParts.has(part),
											)
											.map(({ label }) => label.toLowerCase())
											.join(", ")}. Ctrl+click adds parts.`}
							</p>
							<button
								class="button"
								type="button"
								disabled={isAllSelected}
								onclick={() =>
									(selectedParts = new Set(
										bodyParts.map(({ part }) => part),
									))}>Select all parts</button>
						</div>
						<ul
							class="flex flex-wrap gap-1"
							aria-label="Roblox body colors">
							{#each rules?.bodyColorPalette ?? [] as paletteColor (paletteColor.hexColor)}
								{@const isCurrent =
									selectedColor === paletteColor.hexColor}
								<li>
									<button
										class="grid size-6 place-items-center rounded-sm border border-line-strong outline-offset-2 transition-colors duration-100 hover:border-line-focus focus-visible:outline-2 focus-visible:outline-focus aria-pressed:ring-2 aria-pressed:ring-fg-bright aria-pressed:ring-offset-1 aria-pressed:ring-offset-surface"
										style:background-color={`#${paletteColor.hexColor}`}
										type="button"
										aria-label={paletteColor.name ||
											`#${paletteColor.hexColor}`}
										aria-pressed={isCurrent}
										data-tooltip={paletteColor.name ||
											`#${paletteColor.hexColor}`}
										disabled={avatar.isSaving}
										onclick={() => setColor(paletteColor.hexColor)}
									></button>
								</li>
							{/each}
						</ul>
						<form
							class="flex items-start gap-2"
							novalidate
							onsubmit={(event) => {
								event.preventDefault()
								applyHexText()
							}}>
							<label class="flex w-40 flex-col gap-1">
								<span class="text-xs text-fg-muted">Custom color</span>
								<span class="relative flex items-center">
									<span
										class="pointer-events-none absolute left-2 size-4 rounded-sm border border-line-strong"
										style:background-color={isHexValid
											? `#${hexText.trim().replace("#", "")}`
											: undefined}
										aria-hidden="true"></span>
									<input
										bind:this={hexInput}
										bind:value={hexText}
										class="field pl-8 font-mono text-xs uppercase"
										type="text"
										spellcheck="false"
										autocomplete="off"
										maxlength="7"
										placeholder={selectedColor ? "" : "Mixed"}
										aria-invalid={isHexTouched && !isHexValid}
										aria-describedby="avatar-hex-error"
										disabled={avatar.isSaving}
										onblur={() => {
											if (
												hexText.trim() &&
												hexText.trim() !== selectedColor
											)
												applyHexText()
										}} />
								</span>
								{#if isHexTouched && !isHexValid}
									<span
										id="avatar-hex-error"
										class="text-xs text-danger-muted"
										>Enter 6 hex digits, such as A3A2A5.</span>
								{/if}
							</label>
							<button
								class="mt-5 button-icon size-8"
								type="submit"
								aria-label="Apply custom color"
								data-tooltip="Apply custom color"
								disabled={avatar.isSaving || !hexText.trim()}>
								<Icon svg={checkIcon} size={16} aria-hidden="true" />
							</button>
						</form>
					</div>
				{:else}
					<dl class="facts min-w-0 flex-1">
						{#each bodyParts as { part, label } (part)}
							<div>
								<dt>
									<span
										class="size-3 shrink-0 rounded-sm border border-line-strong"
										style:background-color={draft.bodyColors[part]
											? `#${draft.bodyColors[part]}`
											: undefined}></span
									>{label}
								</dt>
								<dd>
									{#if draft.bodyColors[part]}
										<span class="font-mono select-text"
											>#{draft.bodyColors[part]}</span>
									{:else}
										<span class="text-fg-muted">Unavailable</span>
									{/if}
								</dd>
							</div>
						{/each}
					</dl>
				{/if}
			</div>
		</section>
	</div>
{:else}
	<div class="flex flex-col gap-3 p-3" role="status" aria-busy="true">
		<Skeleton className="w-48" label="Loading the body" />
	</div>
{/if}

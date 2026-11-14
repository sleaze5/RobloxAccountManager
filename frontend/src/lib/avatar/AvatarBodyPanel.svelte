<script lang="ts">
	import checkIcon from "@phosphor-icons/core/regular/check.svg?raw"
	import { AvatarType } from "../backend/bridge"
	import type { AvatarScaleLimit, AvatarScales } from "../backend/bridge"
	import Icon from "../shared/Icon.svelte"
	import Skeleton from "../shared/Skeleton.svelte"
	import { replayValidationNudge } from "../shared/presence"
	import AvatarBodyColorFacts from "./AvatarBodyColorFacts.svelte"
	import AvatarBodyFigure from "./AvatarBodyFigure.svelte"
	import { bodyParts, bodyScales } from "./avatar-draft"
	import type { AvatarDraft, BodyPart } from "./avatar-draft"
	import type { AvatarState } from "./avatar-state.svelte"

	let { avatar, canEdit }: { avatar: AvatarState; canEdit: boolean } = $props()

	const percent = new Intl.NumberFormat("en-US", {
			style: "percent",
			maximumFractionDigits: 0,
		}),
		hexDigitPattern = /[0-9a-f]/gi,
		fullHexColorPattern = /[0-9a-f]{6}/i,
		shortHexColorPattern = /[0-9a-f]{3}/i,
		maxHexDigits = 6

	let selectedParts = $state(new Set<BodyPart>(bodyParts.map(({ part }) => part))),
		hexInput = $state<HTMLInputElement | undefined>(undefined),
		hexText = $state(""),
		isHexTouched = $state(false)

	const draft = $derived(avatar.draft),
		rules = $derived(avatar.rules),
		isR15 = $derived(draft?.avatarType === AvatarType.AvatarR15),
		shownScaleRows = $derived(
			bodyScales.filter(
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
		expandedHexColor = $derived(expandHexColor(hexText)),
		isAllSelected = $derived(selectedParts.size === bodyParts.length),
		selectedPaletteColor = $derived(avatar.paletteColorByHex.get(selectedColor))

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
		if (parts.has(part)) parts.delete(part)
		else parts.add(part)
		selectedParts = parts
	}

	function hexDigitsOf(text: string): string {
		return (text.match(hexDigitPattern) ?? []).join("")
	}

	// expandHexColor writes a color of 1 or 3 hex digits in full, so that "0"
	// becomes 000000 and "1a2" becomes 11AA22. It returns null for any other
	// length than 1, 3, or 6.
	function expandHexColor(digits: string): string | null {
		const upperDigits = digits.toUpperCase()
		switch (upperDigits.length) {
			case 1:
				return upperDigits.repeat(6)
			case 3:
				return [...upperDigits].map((digit) => digit + digit).join("")
			case 6:
				return upperDigits
			default:
				return null
		}
	}

	// showHexDigits keeps the first 6 hex digits of the text, and keeps the
	// caret after the same digit that it followed in the text.
	function showHexDigits(input: HTMLInputElement, text: string, caret: number): void {
		const digits = hexDigitsOf(text).slice(0, maxHexDigits),
			position = Math.min(hexDigitsOf(text.slice(0, caret)).length, digits.length)
		hexText = digits
		input.value = digits
		input.setSelectionRange(position, position)
	}

	// pasteHexText takes the first full color in the pasted text, such as the
	// A3A2A5 in "#A3A2A5;", then the first short color, such as the FFF in
	// "rgb #fff". It inserts other pasted hex digits at the caret.
	function pasteHexText(input: HTMLInputElement, pastedText: string): void {
		const pastedColor =
			fullHexColorPattern.exec(pastedText)?.[0] ??
			shortHexColorPattern.exec(pastedText)?.[0]
		if (pastedColor) {
			showHexDigits(input, pastedColor, pastedColor.length)
			return
		}
		const selectionStart = input.selectionStart ?? input.value.length,
			selectionEnd = input.selectionEnd ?? selectionStart,
			textBeforeCaret = input.value.slice(0, selectionStart) + pastedText
		showHexDigits(
			input,
			textBeforeCaret + input.value.slice(selectionEnd),
			textBeforeCaret.length,
		)
	}

	// colorLabelOf names a palette color with its BrickColor, such as "Light
	// orange (#EAB892, BrickColor 125)".
	function colorLabelOf(hexColor: string): string {
		const paletteColor = avatar.paletteColorByHex.get(hexColor)
		if (!paletteColor) return `custom color #${hexColor}`
		return `${paletteColor.name || "BrickColor"} (#${hexColor}, BrickColor ${paletteColor.brickColorId})`
	}

	function applyHexText(): void {
		isHexTouched = true
		const hexColor = expandHexColor(hexText)
		if (!hexColor) {
			replayValidationNudge(hexInput)
			return
		}
		hexText = hexColor
		setColor(hexColor)
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
									disabled={avatar.isDraftLocked}
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
								disabled={!isR15 || avatar.isDraftLocked}
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
							{colorLabelOf}
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
								{selectedParts.size === 0
									? "No parts selected. Select parts on the figure to color them."
									: isAllSelected
										? "Coloring every part. Select a part to color only it."
										: `Coloring ${bodyParts
												.filter(({ part }) =>
													selectedParts.has(part),
												)
												.map(({ label }) => label.toLowerCase())
												.join(
													", ",
												)}. Ctrl+click adds or removes parts.`}
							</p>
							<button
								class="button"
								type="button"
								disabled={isAllSelected}
								onclick={() =>
									(selectedParts = new Set(
										bodyParts.map(({ part }) => part),
									))}>Select all</button>
							<button
								class="button"
								type="button"
								disabled={selectedParts.size === 0}
								onclick={() => (selectedParts = new Set())}
								>Clear selection</button>
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
										aria-label={colorLabelOf(paletteColor.hexColor)}
										aria-pressed={isCurrent}
										data-tooltip={colorLabelOf(
											paletteColor.hexColor,
										)}
										disabled={avatar.isDraftLocked ||
											selectedParts.size === 0}
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
										style:background-color={expandedHexColor
											? `#${expandedHexColor}`
											: undefined}
										aria-hidden="true"></span>
									<span
										class="pointer-events-none absolute left-7 font-mono text-xs text-fg-muted"
										aria-hidden="true">#</span>
									<input
										bind:this={hexInput}
										value={hexText}
										class="field pl-10 font-mono text-xs"
										type="text"
										spellcheck="false"
										autocomplete="off"
										maxlength={maxHexDigits}
										placeholder={selectedColor ||
										selectedParts.size === 0
											? ""
											: "Mixed"}
										aria-invalid={isHexTouched && !expandedHexColor}
										aria-describedby="avatar-hex-error"
										disabled={avatar.isDraftLocked ||
											selectedParts.size === 0}
										oninput={(event) =>
											showHexDigits(
												event.currentTarget,
												event.currentTarget.value,
												event.currentTarget.selectionStart ??
													event.currentTarget.value.length,
											)}
										onpaste={(event) => {
											event.preventDefault()
											pasteHexText(
												event.currentTarget,
												event.clipboardData?.getData("text") ??
													"",
											)
										}}
										onblur={() => {
											if (hexText) applyHexText()
										}} />
								</span>
								{#if isHexTouched && !expandedHexColor}
									<span
										id="avatar-hex-error"
										class="text-xs text-danger-muted"
										>Enter 1, 3, or 6 hex digits, such as A3A2A5.</span>
								{:else if selectedPaletteColor}
									<span class="text-xs text-fg-muted"
										>{selectedPaletteColor.name
											? `${selectedPaletteColor.name}, `
											: ""}BrickColor
										<span class="font-mono"
											>{selectedPaletteColor.brickColorId}</span
										></span>
								{:else if selectedColor}
									<span class="text-xs text-fg-muted"
										>Not a BrickColor</span>
								{/if}
							</label>
							<button
								class="mt-5 button-icon size-8"
								type="submit"
								aria-label="Apply custom color"
								data-tooltip="Apply custom color"
								disabled={avatar.isDraftLocked ||
									!hexText ||
									selectedParts.size === 0}>
								<Icon svg={checkIcon} size={16} aria-hidden="true" />
							</button>
						</form>
					</div>
				{:else}
					<AvatarBodyColorFacts
						bodyColors={draft.bodyColors}
						paletteColorByHex={avatar.paletteColorByHex}
						isPaletteLoaded={!!rules} />
				{/if}
			</div>
		</section>
	</div>
{:else}
	<div class="flex flex-col gap-3 p-3" role="status" aria-busy="true">
		<Skeleton className="w-48" label="Loading the body" />
	</div>
{/if}

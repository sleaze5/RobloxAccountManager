<script lang="ts">
	import arrowCounterClockwiseIcon from "@phosphor-icons/core/regular/arrow-counter-clockwise.svg?raw"
	import clipboardTextIcon from "@phosphor-icons/core/regular/clipboard-text.svg?raw"
	import copyIcon from "@phosphor-icons/core/regular/copy.svg?raw"
	import flipHorizontalIcon from "@phosphor-icons/core/regular/flip-horizontal.svg?raw"
	import { untrack } from "svelte"
	import type {
		AvatarAccessoryAdjustment,
		AvatarAccessoryAdjustmentLimits,
		AvatarAsset,
	} from "../backend/bridge"
	import CheckboxMark from "../shared/CheckboxMark.svelte"
	import Icon from "../shared/Icon.svelte"
	import { formatAdjustmentValue } from "./adjustment-format"
	import {
		adjustmentAxes,
		adjustmentKinds,
		adjustmentOf,
		defaultAdjustment,
		isDefaultAdjustment,
		isDefaultAdjustmentKind,
		isSameAdjustment,
	} from "./avatar-draft"
	import type { AdjustmentAxis, AdjustmentKind } from "./avatar-draft"
	import type { AvatarState } from "./avatar-state.svelte"

	type AxisRange = { min: number; max: number }

	let {
		avatar,
		asset,
		limits,
	}: {
		avatar: AvatarState
		asset: AvatarAsset
		limits: AvatarAccessoryAdjustmentLimits
	} = $props()

	const assetId = $derived(asset.item.itemId),
		name = $derived(asset.item.name || "an accessory"),
		adjustment = $derived(adjustmentOf(asset)),
		copied = $derived(avatar.copiedAdjustment),
		canPaste = $derived(!!copied && copied.sourceAssetId !== assetId),
		isCopiedFromHere = $derived(
			!!copied &&
				copied.sourceAssetId === assetId &&
				isSameAdjustment(copied.adjustment, adjustment),
		),
		// The one size range that every scale axis allows.
		uniformScaleRange = $derived<AxisRange>({
			min: Math.max(limits.scale.min.x, limits.scale.min.y, limits.scale.min.z),
			max: Math.min(limits.scale.max.x, limits.scale.max.y, limits.scale.max.z),
		})

	// The size changes on every axis at once while the axes are equal, as the
	// Roblox avatar editor offers it.
	let isUniformScale = $state(untrack(() => isEvenScale(adjustmentOf(asset))))

	// An undo or a paste can bring a size that differs between the axes, which
	// one size slider cannot show.
	$effect(() => {
		if (!isEvenScale(adjustment)) untrack(() => (isUniformScale = false))
	})

	function isEvenScale(shownAdjustment: AvatarAccessoryAdjustment): boolean {
		const { x, y, z } = shownAdjustment.scale
		return x === y && y === z
	}

	// fillOf fills the track from the default value to the current one, so
	// that a range centered on 0 shows which way the accessory moved.
	function fillOf(
		value: number,
		defaultValue: number,
		range: AxisRange,
	): { start: string; end: string } {
		const shareOf = (point: number): number =>
				range.max > range.min
					? Math.min(
							Math.max((point - range.min) / (range.max - range.min), 0),
							1,
						)
					: 0,
			valueShare = shareOf(value),
			defaultShare = shareOf(defaultValue)
		return {
			start: `${Math.min(valueShare, defaultShare) * 100}%`,
			end: `${Math.max(valueShare, defaultShare) * 100}%`,
		}
	}

	// roundedToStep rounds to the step and keeps the value within the range,
	// because range inputs add floating point noise such as 0.1500000001.
	function roundedToStep(rawValue: number, step: number, range: AxisRange): number {
		const value = Number((Math.round(rawValue / step) * step).toFixed(4))
		return Math.min(Math.max(value, range.min), range.max)
	}

	function setAxes(
		kind: AdjustmentKind,
		axes: AdjustmentAxis[],
		rawValue: number,
		step: number,
		range: AxisRange,
	): void {
		const value = roundedToStep(rawValue, step, range),
			vector = { ...adjustment[kind] }
		for (const axis of axes) vector[axis] = value
		avatar.adjust(
			assetId,
			{ ...adjustment, [kind]: vector },
			`adjust:${assetId}:${kind}:${axes.join("")}`,
		)
	}

	// stepWithShift moves a slider ten steps for Shift and an arrow key,
	// because a rotation range holds hundreds of steps.
	function stepWithShift(
		event: KeyboardEvent,
		kind: AdjustmentKind,
		axes: AdjustmentAxis[],
		step: number,
		range: AxisRange,
	): void {
		const direction = { ArrowRight: 1, ArrowUp: 1, ArrowLeft: -1, ArrowDown: -1 }[
			event.key
		]
		if (!event.shiftKey || direction === undefined) return
		event.preventDefault()
		setAxes(
			kind,
			axes,
			adjustment[kind][axes[0]] + direction * step * 10,
			step,
			range,
		)
	}

	function fieldValueOf(value: number, fieldFactor: number): number {
		return Number((value * fieldFactor).toFixed(2))
	}

	// commitField applies a typed value within the range, and then shows the
	// applied value, or the current value when the field holds no number.
	function commitField(
		field: HTMLInputElement,
		kind: AdjustmentKind,
		axes: AdjustmentAxis[],
		step: number,
		fieldFactor: number,
		range: AxisRange,
	): void {
		const typedValue = field.valueAsNumber
		let shownValue = adjustment[kind][axes[0]]
		if (Number.isFinite(typedValue)) {
			shownValue = roundedToStep(typedValue / fieldFactor, step, range)
			setAxes(kind, axes, shownValue, step, range)
			avatar.finishDraftChange()
		}
		field.value = String(fieldValueOf(shownValue, fieldFactor))
	}

	function setUniformScale(isUniform: boolean): void {
		isUniformScale = isUniform
		if (!isUniform) return
		// The axes take the width, kept within the range of every axis.
		const size = Math.min(
			Math.max(adjustment.scale.x, uniformScaleRange.min),
			uniformScaleRange.max,
		)
		avatar.adjust(assetId, { ...adjustment, scale: { x: size, y: size, z: size } })
	}

	function resetKind(kind: AdjustmentKind): void {
		avatar.adjust(assetId, {
			...adjustment,
			[kind]: { ...defaultAdjustment[kind] },
		})
		if (kind === "scale") isUniformScale = true
	}

	function resetAll(): void {
		avatar.adjust(assetId, defaultAdjustment)
		isUniformScale = true
	}
</script>

{#snippet axisControl(
	kind: AdjustmentKind,
	kindLabel: string,
	axisLabel: string,
	axisLetter: string,
	axes: AdjustmentAxis[],
	step: number,
	fieldFactor: number,
	range: AxisRange,
)}
	{@const value = adjustment[kind][axes[0]]}
	{@const fill = fillOf(value, defaultAdjustment[kind][axes[0]], range)}
	{@const accessibleLabel = `${kindLabel} ${axisLabel.toLowerCase()}`}
	<div class="flex min-h-7 items-center gap-3">
		<span
			class="flex w-16 shrink-0 items-baseline gap-1.5 text-xs"
			aria-hidden="true">
			<span class="text-fg">{axisLabel}</span>
			{#if axisLetter}
				<span class="font-mono text-2xs text-fg-faint">{axisLetter}</span>
			{/if}
		</span>
		<input
			class="range flex-1"
			type="range"
			min={range.min}
			max={range.max}
			{step}
			{value}
			style:--range-fill-start={fill.start}
			style:--range-fill={fill.end}
			aria-label={accessibleLabel}
			aria-valuetext={formatAdjustmentValue(kind, value)}
			aria-keyshortcuts="Shift+ArrowLeft Shift+ArrowRight"
			disabled={avatar.isDraftLocked}
			oninput={(event) =>
				setAxes(kind, axes, event.currentTarget.valueAsNumber, step, range)}
			onkeydown={(event) => stepWithShift(event, kind, axes, step, range)}
			onkeyup={(event) => {
				if (event.key.startsWith("Arrow")) avatar.finishDraftChange()
			}}
			onchange={() => avatar.finishDraftChange()} />
		<input
			class="field h-7 w-16 shrink-0 px-2 text-right font-mono text-xs tabular-nums"
			type="number"
			inputmode="decimal"
			min={fieldValueOf(range.min, fieldFactor)}
			max={fieldValueOf(range.max, fieldFactor)}
			step={fieldValueOf(step, fieldFactor)}
			value={fieldValueOf(value, fieldFactor)}
			aria-label={`${accessibleLabel} value`}
			disabled={avatar.isDraftLocked}
			onchange={(event) =>
				commitField(event.currentTarget, kind, axes, step, fieldFactor, range)}
			onkeydown={(event) => {
				if (event.key === "Enter") {
					event.preventDefault()
					commitField(
						event.currentTarget,
						kind,
						axes,
						step,
						fieldFactor,
						range,
					)
				}
			}} />
	</div>
{/snippet}

<div class="flex flex-col gap-4">
	<div class="grid gap-x-6 gap-y-4 @4xl:grid-cols-3">
		{#each adjustmentKinds as { kind, label, unitLabel, axisLabels, step, fieldFactor } (kind)}
			{@const headingId = `avatar-adjust-${assetId}-${kind}`}
			<section class="flex min-w-0 flex-col gap-1" aria-labelledby={headingId}>
				<div class="flex min-h-6 items-center gap-2">
					<h4
						id={headingId}
						class="flex-1 text-xs font-semibold text-fg-muted">
						{label}
						<span class="font-normal text-fg-faint">{unitLabel}</span>
					</h4>
					<button
						class="button-text"
						type="button"
						aria-label={`Reset ${label.toLowerCase()}`}
						disabled={avatar.isDraftLocked ||
							isDefaultAdjustmentKind(asset.adjustment, kind)}
						onclick={() => resetKind(kind)}>Reset</button>
				</div>
				{#if kind === "scale" && isUniformScale}
					{@render axisControl(
						kind,
						label,
						"Size",
						"",
						adjustmentAxes,
						step,
						fieldFactor,
						uniformScaleRange,
					)}
				{:else}
					{#each adjustmentAxes as axis (axis)}
						{@render axisControl(
							kind,
							label,
							axisLabels[axis],
							axis.toUpperCase(),
							[axis],
							step,
							fieldFactor,
							{
								min: limits[kind].min[axis],
								max: limits[kind].max[axis],
							},
						)}
					{/each}
				{/if}
				{#if kind === "scale"}
					<label class="check-option mt-1">
						<span class="checkbox">
							<input
								type="checkbox"
								checked={isUniformScale}
								disabled={avatar.isDraftLocked}
								onchange={(event) =>
									setUniformScale(event.currentTarget.checked)} />
							<CheckboxMark />
						</span>
						<span><strong>Same size on every axis</strong></span>
					</label>
				{/if}
			</section>
		{/each}
	</div>

	<div class="flex flex-col gap-2 border-t border-line-subtle pt-3">
		<div class="flex flex-wrap items-center gap-2">
			<button
				class="button"
				type="button"
				disabled={isCopiedFromHere}
				onclick={() => avatar.copyAdjustment(assetId, name, adjustment)}>
				<Icon svg={copyIcon} size={16} aria-hidden="true" />
				{isCopiedFromHere ? "Copied" : "Copy placement"}
			</button>
			<button
				class="button"
				type="button"
				disabled={avatar.isDraftLocked || !canPaste}
				onclick={() => avatar.pasteAdjustment(assetId, limits, false)}>
				<Icon svg={clipboardTextIcon} size={16} aria-hidden="true" />
				Paste
			</button>
			<button
				class="button"
				type="button"
				data-tooltip="Paste for the other side of the avatar"
				disabled={avatar.isDraftLocked || !canPaste}
				onclick={() => avatar.pasteAdjustment(assetId, limits, true)}>
				<Icon svg={flipHorizontalIcon} size={16} aria-hidden="true" />
				Paste mirrored
			</button>
			<button
				class="ml-auto button"
				type="button"
				disabled={avatar.isDraftLocked || isDefaultAdjustment(asset.adjustment)}
				onclick={resetAll}>
				<Icon svg={arrowCounterClockwiseIcon} size={16} aria-hidden="true" />
				Reset all
			</button>
		</div>
		<p class="text-xs text-fg-muted" aria-live="polite">
			{#if copied && canPaste}
				Paste uses the placement copied from {copied.sourceName}, within this
				accessory's limits.
			{:else}
				Copy a placement to reuse it. Paste mirrored fits a left accessory to
				its right counterpart.
			{/if}
		</p>
	</div>
</div>

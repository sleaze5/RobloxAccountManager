<script lang="ts">
	import arrowCounterClockwiseIcon from "@phosphor-icons/core/regular/arrow-counter-clockwise.svg?raw"
	import { untrack } from "svelte"
	import type {
		AvatarAccessoryAdjustmentLimits,
		AvatarAsset,
		AvatarAxisLimits,
	} from "../backend/bridge"
	import CheckboxMark from "../shared/CheckboxMark.svelte"
	import Icon from "../shared/Icon.svelte"
	import { adjustmentUnitLabel, formatAdjustmentValue } from "./adjustment-format"
	import {
		adjustmentAxes,
		adjustmentKinds,
		adjustmentOf,
		defaultAdjustment,
		isDefaultAdjustment,
	} from "./avatar-draft"
	import type { AdjustmentAxis, AdjustmentKind } from "./avatar-draft"
	import type { AvatarState } from "./avatar-state.svelte"

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
		adjustment = $derived(adjustmentOf(asset)),
		// The one size range that every scale axis allows.
		uniformScaleLimit = $derived({
			min: Math.max(limits.scale.min.x, limits.scale.min.y, limits.scale.min.z),
			max: Math.min(limits.scale.max.x, limits.scale.max.y, limits.scale.max.z),
		})

	// The size changes on every axis at once while the axes are equal, as the
	// Roblox avatar editor offers it.
	let isUniformScale = $state(
		untrack(
			() =>
				adjustmentOf(asset).scale.x === adjustmentOf(asset).scale.y &&
				adjustmentOf(asset).scale.y === adjustmentOf(asset).scale.z,
		),
	)

	// An undo can restore a size that differs between the axes, which one
	// size slider cannot show.
	$effect(() => {
		const { x, y, z } = adjustment.scale
		if (x !== y || y !== z) untrack(() => (isUniformScale = false))
	})

	function axisLimitOf(
		axisLimits: AvatarAxisLimits,
		axis: AdjustmentAxis,
	): { min: number; max: number } {
		return { min: axisLimits.min[axis], max: axisLimits.max[axis] }
	}

	function fillOf(value: number, limit: { min: number; max: number }): string {
		const share =
			limit.max > limit.min ? (value - limit.min) / (limit.max - limit.min) : 0
		return `${Math.min(Math.max(share, 0), 1) * 100}%`
	}

	// roundedToStep rounds to the step and keeps the value within the limit,
	// because range inputs add floating point noise such as 0.1500000001.
	function roundedToStep(
		rawValue: number,
		step: number,
		limit: { min: number; max: number },
	): number {
		const value = Number((Math.round(rawValue / step) * step).toFixed(4))
		return Math.min(Math.max(value, limit.min), limit.max)
	}

	function setAxes(
		kind: AdjustmentKind,
		axes: AdjustmentAxis[],
		rawValue: number,
		step: number,
		limit: { min: number; max: number },
	): void {
		const value = roundedToStep(rawValue, step, limit),
			vector = { ...adjustment[kind] }
		for (const axis of axes) vector[axis] = value
		avatar.adjust(
			assetId,
			{ ...adjustment, [kind]: vector },
			`adjust:${assetId}:${kind}:${axes.join("")}`,
		)
	}

	function setUniformScale(isUniform: boolean): void {
		isUniformScale = isUniform
		if (!isUniform) return
		// The axes take the size of X, kept within the range of every axis.
		const size = Math.min(
			Math.max(adjustment.scale.x, uniformScaleLimit.min),
			uniformScaleLimit.max,
		)
		avatar.adjust(assetId, { ...adjustment, scale: { x: size, y: size, z: size } })
	}

	function reset(): void {
		avatar.adjust(assetId, defaultAdjustment)
		isUniformScale = true
	}
</script>

{#snippet slider(
	kind: AdjustmentKind,
	kindLabel: string,
	axes: AdjustmentAxis[],
	step: number,
	limit: { min: number; max: number },
)}
	{@const value = adjustment[kind][axes[0]]}
	{@const axisLabel = axes.length === 1 ? axes[0].toUpperCase() : "Size"}
	<div class="flex min-h-7 items-center gap-3">
		<span class="w-8 shrink-0 font-mono text-xs text-fg-muted" aria-hidden="true"
			>{axisLabel}</span>
		<input
			class="range flex-1"
			type="range"
			min={limit.min}
			max={limit.max}
			{step}
			{value}
			style:--range-fill={fillOf(value, limit)}
			aria-label={axes.length === 1 ? `${kindLabel} ${axisLabel}` : axisLabel}
			aria-valuetext={`${formatAdjustmentValue(kind, value)} ${adjustmentUnitLabel(kind)}`}
			disabled={avatar.isDraftLocked}
			oninput={(event) =>
				setAxes(kind, axes, event.currentTarget.valueAsNumber, step, limit)}
			onchange={() => avatar.finishDraftChange()} />
		<span class="w-14 shrink-0 text-right font-mono text-xs text-fg tabular-nums"
			>{formatAdjustmentValue(kind, value)}</span>
	</div>
{/snippet}

<div class="flex flex-col gap-3">
	<div class="grid gap-4 @2xl:grid-cols-3">
		{#each adjustmentKinds as { kind, label, step } (kind)}
			<fieldset class="flex min-w-0 flex-col gap-1">
				<legend class="mb-1 text-xs font-semibold text-fg-muted"
					>{label}
					<span class="font-normal text-fg-faint"
						>{adjustmentUnitLabel(kind)}</span
					></legend>
				{#if kind === "scale" && isUniformScale}
					{@render slider(
						kind,
						label,
						adjustmentAxes,
						step,
						uniformScaleLimit,
					)}
				{:else}
					{#each adjustmentAxes as axis (axis)}
						{@render slider(
							kind,
							label,
							[axis],
							step,
							axisLimitOf(limits[kind], axis),
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
			</fieldset>
		{/each}
	</div>
	<div class="flex items-center justify-end">
		<button
			class="button"
			type="button"
			disabled={avatar.isDraftLocked || isDefaultAdjustment(asset.adjustment)}
			onclick={reset}>
			<Icon svg={arrowCounterClockwiseIcon} size={16} aria-hidden="true" />
			Reset placement
		</button>
	</div>
</div>

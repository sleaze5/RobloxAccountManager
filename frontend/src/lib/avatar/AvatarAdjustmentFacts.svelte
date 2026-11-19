<script lang="ts">
	import arrowRightIcon from "@phosphor-icons/core/regular/arrow-right.svg?raw"
	import type { AvatarAccessoryAdjustment } from "../backend/bridge"
	import Icon from "../shared/Icon.svelte"
	import { formatAdjustmentValue } from "./adjustment-format"
	import { adjustmentAxes, adjustmentKinds, defaultAdjustment } from "./avatar-draft"
	import type { AdjustmentAxis, AdjustmentKind } from "./avatar-draft"

	// AvatarAdjustmentFacts lists only what an adjustment moves: each axis
	// that differs from where the accessory attaches by default, as the
	// default value and the adjusted value. Roblox stores the adjusted values,
	// and the default placement is fixed, so the comparison is exact.
	let { adjustment }: { adjustment: AvatarAccessoryAdjustment } = $props()

	const changedKinds = $derived(
		adjustmentKinds.filter(({ kind }) => changedAxesOf(kind).length > 0),
	)

	function changedAxesOf(kind: AdjustmentKind): AdjustmentAxis[] {
		return adjustmentAxes.filter(
			(axis) => adjustment[kind][axis] !== defaultAdjustment[kind][axis],
		)
	}
</script>

<!-- Each change stays on one line, and the changes of a row wrap as a whole. -->
<dl class="facts">
	{#each changedKinds as { kind, label, unitLabel, axisLabels } (kind)}
		<div>
			<dt>
				<span>{label} <span class="text-fg-faint">{unitLabel}</span></span>
			</dt>
			<dd>
				{#each changedAxesOf(kind) as axis (axis)}
					<span class="flex items-center gap-1.5 whitespace-nowrap">
						<span class="text-fg-muted">{axisLabels[axis]}</span>
						<span class="font-mono text-fg-muted tabular-nums"
							>{formatAdjustmentValue(
								kind,
								defaultAdjustment[kind][axis],
							)}</span>
						<Icon
							svg={arrowRightIcon}
							class="shrink-0 text-fg-faint"
							size={12}
							aria-hidden="true" />
						<span class="sr-only">to</span>
						<span class="font-mono tabular-nums select-text"
							>{formatAdjustmentValue(
								kind,
								adjustment[kind][axis],
							)}</span>
					</span>
				{/each}
			</dd>
		</div>
	{/each}
</dl>

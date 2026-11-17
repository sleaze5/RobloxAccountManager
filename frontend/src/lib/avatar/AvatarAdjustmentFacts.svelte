<script lang="ts">
	import type { AvatarAccessoryAdjustment } from "../backend/bridge"
	import { formatAdjustmentValue } from "./adjustment-format"
	import { adjustmentAxes, adjustmentKinds } from "./avatar-draft"

	let { adjustment }: { adjustment: AvatarAccessoryAdjustment } = $props()
</script>

<!-- Each axis takes an equal column, so that the values of position,
     rotation, and scale line up under each other. -->
<dl class="facts">
	{#each adjustmentKinds as { kind, label, unitLabel, axisLabels } (kind)}
		<div>
			<dt>
				<span>{label} <span class="text-fg-faint">{unitLabel}</span></span>
			</dt>
			<dd>
				<span class="grid w-full grid-cols-3 gap-3">
					{#each adjustmentAxes as axis (axis)}
						<span class="flex min-w-0 items-baseline gap-1.5">
							<span class="text-fg-muted">{axisLabels[axis]}</span>
							<span class="font-mono tabular-nums select-text"
								>{formatAdjustmentValue(
									kind,
									adjustment[kind][axis],
								)}</span>
						</span>
					{/each}
				</span>
			</dd>
		</div>
	{/each}
</dl>

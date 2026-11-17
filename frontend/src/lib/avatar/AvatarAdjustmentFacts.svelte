<script lang="ts">
	import type { AvatarAccessoryAdjustment } from "../backend/bridge"
	import { formatAdjustmentValue } from "./adjustment-format"
	import { adjustmentAxes, adjustmentKinds } from "./avatar-draft"

	let { adjustment }: { adjustment: AvatarAccessoryAdjustment } = $props()
</script>

<dl class="facts">
	{#each adjustmentKinds as { kind, label, unitLabel, axisLabels } (kind)}
		<div>
			<dt>{label}</dt>
			<dd>
				{#each adjustmentAxes as axis (axis)}
					<span class="tabular-nums"
						><span class="text-fg-muted">{axisLabels[axis]}</span>
						<span class="font-mono select-text"
							>{formatAdjustmentValue(kind, adjustment[kind][axis])}</span
						></span>
				{/each}
				<small>{unitLabel}</small>
			</dd>
		</div>
	{/each}
</dl>

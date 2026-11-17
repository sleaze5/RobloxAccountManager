<script lang="ts">
	import type { AvatarAccessoryAdjustment } from "../backend/bridge"
	import { adjustmentUnitLabel, formatAdjustmentValue } from "./adjustment-format"
	import { adjustmentAxes, adjustmentKinds } from "./avatar-draft"

	let { adjustment }: { adjustment: AvatarAccessoryAdjustment } = $props()
</script>

<dl class="facts">
	{#each adjustmentKinds as { kind, label } (kind)}
		<div>
			<dt>{label}</dt>
			<dd>
				{#each adjustmentAxes as axis (axis)}
					<span class="font-mono tabular-nums select-text"
						><span class="text-fg-muted">{axis.toUpperCase()}</span>
						{formatAdjustmentValue(kind, adjustment[kind][axis])}</span>
				{/each}
				<small>{adjustmentUnitLabel(kind)}</small>
			</dd>
		</div>
	{/each}
</dl>

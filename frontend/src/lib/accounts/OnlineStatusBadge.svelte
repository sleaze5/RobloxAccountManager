<script lang="ts">
	import type { OnlineStatus } from "../backend/bridge"
	import {
		onlineStatusDotClass,
		onlineStatusLabel,
		onlineStatusTooltip,
	} from "./account-model"

	let { onlineStatus }: { onlineStatus: OnlineStatus } = $props()
</script>

<span
	class={[
		"absolute -right-0.5 -bottom-0.5 size-3.5 cursor-help rounded-full border-2 border-surface ring-1 ring-line-strong",
		onlineStatusDotClass(onlineStatus),
	]}
	role="img"
	aria-label={onlineStatusLabel(onlineStatus)}
	data-tooltip={onlineStatusTooltip(onlineStatus)}
	data-tooltip-side="top"></span>

<script lang="ts">
	import type { UserPresence } from "../backend/bridge"
	import { presenceDotClass, presenceLabel, presenceTooltip } from "./account-model"

	let { presence }: { presence: UserPresence } = $props()
</script>

<span
	class={[
		"absolute -right-0.5 -bottom-0.5 size-3.5 cursor-help rounded-full border-2 border-surface ring-1 ring-line-strong",
		presenceDotClass(presence),
	]}
	role="img"
	aria-label={presenceLabel(presence)}
	data-tooltip={presenceTooltip(presence)}
	data-tooltip-side="top"></span>

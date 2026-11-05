<script lang="ts">
	import Icon from "../shared/Icon.svelte"
	import type { UserPresence } from "../backend/bridge"
	import {
		presenceBadgeIcon,
		presenceDotClass,
		presenceLabel,
		presenceTooltip,
	} from "./account-model"

	let { presence }: { presence: UserPresence } = $props()
	const icon = $derived(presenceBadgeIcon(presence))
</script>

<span
	class={[
		"absolute -right-0.5 -bottom-0.5 grid size-4 cursor-help place-items-center rounded-full border-2 border-surface text-canvas ring-1 ring-line-strong",
		presenceDotClass(presence),
	]}
	role="img"
	aria-label={presenceLabel(presence)}
	data-tooltip={presenceTooltip(presence)}
	data-tooltip-side="top">
	{#if icon}<Icon svg={icon} size={11} aria-hidden="true" />{/if}
</span>

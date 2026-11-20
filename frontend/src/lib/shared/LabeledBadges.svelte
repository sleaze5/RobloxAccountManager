<script lang="ts">
	import Icon from "./Icon.svelte"
	import type { LabeledBadge } from "./labeled-badge"

	let { badges }: { badges: LabeledBadge[] } = $props()
</script>

{#each badges as badge (badge.label)}
	<span
		class={[
			"inline-flex h-5 max-w-full shrink-0 animate-pop items-center gap-1 rounded-sm border border-line-strong bg-raised px-1 text-2xs font-medium",
			badge.tooltip && "cursor-help",
			badge.isStrong ? "text-accent-strong" : "text-fg-muted",
		]}
		data-tooltip={badge.tooltip}>
		<Icon svg={badge.icon} class="shrink-0" size={12} aria-hidden="true" /><span
			class="truncate">{badge.label}</span>
	</span>
{/each}

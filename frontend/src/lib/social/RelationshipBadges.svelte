<script lang="ts">
	import Icon from "../shared/Icon.svelte"
	import type { RelationshipBadge } from "./relationship"

	let { badges }: { badges: RelationshipBadge[] } = $props()
</script>

{#each badges as badge (badge.label)}
	<span
		class={[
			"inline-flex h-5 shrink-0 animate-pop cursor-help items-center gap-1 rounded-sm border border-line-strong bg-raised px-1 text-2xs font-medium",
			badge.isStrong ? "text-accent-strong" : "text-fg-muted",
		]}
		data-tooltip={badge.tooltip}>
		<Icon svg={badge.icon} size={12} aria-hidden="true" />{badge.label}
	</span>
{/each}

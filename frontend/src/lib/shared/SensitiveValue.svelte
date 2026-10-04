<script lang="ts">
	import type { Snippet } from "svelte"

	let { label, children }: { label: string; children: Snippet } = $props()

	let revealed = $state(false)
</script>

<button
	class="hover-value sensitive-value"
	class:revealed
	type="button"
	aria-pressed={revealed}
	aria-label={revealed ? undefined : `${label} hidden. Click to reveal`}
	data-tooltip={revealed ? "Click to hide" : "Click to reveal"}
	data-tooltip-side="top"
	onclick={() => (revealed = !revealed)}>
	<span>{@render children()}</span>
</button>

<style>
	.sensitive-value {
		max-width: 100%;
		cursor: pointer;
		text-align: left;
		overflow-wrap: anywhere;
	}

	.sensitive-value:not(.revealed) > span {
		filter: blur(var(--blur-sensitive));
		user-select: none;
	}
</style>

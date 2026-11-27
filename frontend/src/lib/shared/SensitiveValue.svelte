<script lang="ts">
	import type { Snippet } from "svelte"

	let { label, children }: { label: string; children: Snippet } = $props()

	let revealed = $state(false)
</script>

<button
	class="max-w-full cursor-pointer rounded-md text-left wrap-anywhere tabular-nums focus-visible:outline-offset-2"
	type="button"
	aria-pressed={revealed}
	aria-label={revealed ? undefined : `${label} hidden. Select to show`}
	data-tooltip={revealed ? "Hide" : "Show"}
	data-tooltip-side="top"
	onclick={() => (revealed = !revealed)}>
	<span class={[!revealed && "blur-sm select-none"]}>{@render children()}</span>
</button>

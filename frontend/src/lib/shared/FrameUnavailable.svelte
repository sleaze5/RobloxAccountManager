<script lang="ts">
	import Icon from "./Icon.svelte"

	// FrameUnavailable fills an image frame with why its content is missing,
	// and a Try again button when onRetry is set.
	let {
		icon,
		message,
		onRetry,
	}: { icon: string; message: string; onRetry?: () => void } = $props()
</script>

<div
	class="relative flex animate-enter flex-col items-center gap-2 p-3 text-center text-xs text-fg-muted">
	<Icon svg={icon} size={28} class="animate-bloom" aria-hidden="true" />
	<p>{message}</p>
	{#if onRetry}
		<button class="button" type="button" onclick={onRetry}>Try again</button>
	{/if}
</div>

<script lang="ts">
	// Skeleton stands in for a line of text while it loads. It is as tall as a
	// line of the text around it, so the layout keeps its size when the text
	// arrives.
	let {
		className = "w-24",
		label = "Loading",
	}: { className?: string; label?: string } = $props()
</script>

<span class={["skeleton-line", className]}><span class="sr-only">{label}</span></span>

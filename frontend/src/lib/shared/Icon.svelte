<script lang="ts">
	import type { SVGAttributes } from "svelte/elements"

	let {
		svg,
		size = 16,
		...rest
	}: { svg: string; size?: number } & SVGAttributes<SVGSVGElement> = $props()

	const markup = $derived(svg.replace(/^\s*<svg[^>]*>|<\/svg>\s*$/g, ""))
</script>

<svg
	xmlns="http://www.w3.org/2000/svg"
	role="img"
	viewBox="0 0 256 256"
	width={size}
	height={size}
	fill="currentColor"
	{...rest}>{@html markup}</svg>

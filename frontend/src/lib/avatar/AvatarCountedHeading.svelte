<script lang="ts">
	import Icon from "../shared/Icon.svelte"

	let {
		id,
		label,
		count,
		icon,
	}: {
		id: string
		label: string
		count: number
		icon?: string
	} = $props()
</script>

<h3 {id} class="flex items-center gap-2 text-xs font-semibold text-fg-muted">
	{#if icon}
		<Icon svg={icon} class="text-fg-faint" size={14} aria-hidden="true" />
	{/if}
	{label}
	<span class="font-mono text-2xs font-normal text-fg-faint">{count}</span>
</h3>

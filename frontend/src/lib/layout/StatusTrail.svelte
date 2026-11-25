<script lang="ts">
	import caretRightIcon from "@phosphor-icons/core/regular/caret-right.svg?raw"
	import Icon from "../shared/Icon.svelte"

	// StatusTrail names what the page shows, after the account switcher of the
	// status strip: such as a person, then the page of their profile that is
	// open. Each part is shorter than the one before it to read at a glance.
	let { parts }: { parts: string[] } = $props()
</script>

{#if parts.length}
	<span class="flex min-w-0 items-center gap-1" aria-live="polite">
		<span class="sr-only">Viewing</span>
		{#each parts as part, index (index)}
			<Icon
				svg={caretRightIcon}
				class="shrink-0 text-fg-faint"
				size={12}
				aria-hidden="true" />
			<span
				class={[
					"min-w-0 truncate",
					index === parts.length - 1 ? "text-fg-muted" : "text-fg-faint",
				]}>{part}</span>
		{/each}
	</span>
{/if}

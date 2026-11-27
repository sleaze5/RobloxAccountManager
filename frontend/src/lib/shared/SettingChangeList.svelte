<script lang="ts" module>
	// SettingChange is one setting that a confirmation lists before it saves,
	// with its value before and after the change.
	export interface SettingChange {
		label: string
		previous: string
		next: string
	}
</script>

<script lang="ts">
	import arrowRightIcon from "@phosphor-icons/core/regular/arrow-right.svg?raw"
	import Icon from "./Icon.svelte"

	let { changes }: { changes: SettingChange[] } = $props()
</script>

<ul
	class="flex flex-col divide-y divide-line-subtle rounded-lg border border-line text-xs"
	aria-label="Changes">
	{#each changes as change (change.label)}
		<li class="flex min-w-0 items-center gap-2 px-3 py-2">
			<span class="w-36 shrink-0 text-fg-muted">{change.label}</span>
			<span class="min-w-0 truncate text-fg-muted line-through"
				>{change.previous}</span>
			<Icon
				svg={arrowRightIcon}
				class="shrink-0 text-fg-muted"
				size={12}
				aria-label="changes to" />
			<span class="min-w-0 truncate font-medium text-fg">{change.next}</span>
		</li>
	{/each}
</ul>

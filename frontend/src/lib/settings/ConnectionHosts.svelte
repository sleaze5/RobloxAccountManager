<script lang="ts">
	import caretRightIcon from "@phosphor-icons/core/regular/caret-right.svg?raw"
	import Icon from "../shared/Icon.svelte"

	let { hosts }: { hosts: string[] } = $props()
</script>

<div class="settings-row">
	<details class="group text-xs text-fg-muted">
		<summary
			class="inline-flex items-center gap-1 rounded-sm transition-colors duration-100 hover:text-fg focus-visible:text-fg focus-visible:outline-offset-2"
			><Icon
				svg={caretRightIcon}
				class="transition-transform duration-100 group-open:rotate-90"
				size={14}
				aria-hidden="true" />Hosts it can connect to</summary>
		<ul class="mt-2 flex animate-enter flex-col gap-0.5">
			{#each hosts as host (host)}
				<li>
					<code
						class="rounded-sm bg-fill px-0.5 font-mono text-2xs text-fg-muted"
						>{host}</code>
				</li>
			{/each}
		</ul>
	</details>
</div>

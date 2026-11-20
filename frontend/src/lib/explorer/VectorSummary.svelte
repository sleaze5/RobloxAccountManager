<script lang="ts">
	import type { ModelProperty } from "../backend/bridge"

	// VectorSummary writes the components of one or more vectors on one line,
	// as Roblox Studio shows a Vector3: 1, 0.8, 4. The separators are quieter
	// than the numbers.
	let { parts }: { parts: NonNullable<ModelProperty["vectorParts"]> } = $props()
</script>

<span class="min-w-0 truncate font-mono tabular-nums select-text"
	>{#each parts as part, partIndex (partIndex)}{#if partIndex > 0}<span
				class="text-fg-muted">&ensp;·&ensp;</span
			>{/if}{#each part.components ?? [] as component, componentIndex (componentIndex)}{#if componentIndex > 0}<span
					class="text-fg-muted">,&nbsp;</span
				>{/if}{component}{/each}{/each}</span>

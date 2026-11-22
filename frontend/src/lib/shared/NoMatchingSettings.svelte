<script lang="ts">
	import magnifyingGlassIcon from "@phosphor-icons/core/regular/magnifying-glass.svg?raw"
	import Icon from "./Icon.svelte"

	let { onClearSearch }: { onClearSearch: () => void } = $props()
</script>

<div class="m-auto empty-state">
	<Icon svg={magnifyingGlassIcon} size={28} aria-hidden="true" />
	<h2>No matching settings</h2>
	<p>Change the search to find a setting.</p>
	<button class="button" type="button" onclick={onClearSearch}>Clear search</button>
</div>

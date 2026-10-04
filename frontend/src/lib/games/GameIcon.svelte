<script lang="ts">
	import GameControllerIcon from "phosphor-svelte/lib/GameControllerIcon"
	let { url, large = false }: { url: string; large?: boolean } = $props()
	let failed = $state("")
</script>

<span class="game-icon" class:large aria-hidden="true">
	{#if url && failed !== url}
		<img
			src={url}
			alt=""
			draggable="false"
			loading="lazy"
			decoding="async"
			referrerpolicy="no-referrer"
			onerror={() => (failed = url)} />
	{:else}
		<GameControllerIcon size={large ? 36 : 18} aria-hidden="true" />
	{/if}
</span>

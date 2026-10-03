<script lang="ts">
	import Gamepad2 from "@lucide/svelte/icons/gamepad-2"
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
		<Gamepad2 size={large ? 24 : 16} />
	{/if}
</span>

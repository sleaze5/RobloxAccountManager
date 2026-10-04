<script lang="ts">
	import User from "@lucide/svelte/icons/user"

	let { url }: { url: string } = $props()
	let failed = $state("")
</script>

{#if url && failed !== url}
	<img
		src={url}
		alt=""
		decoding="async"
		draggable="false"
		referrerpolicy="no-referrer"
		onerror={() => (failed = url)} />
{:else}
	<User class="profile-image-fallback" aria-hidden="true" />
{/if}

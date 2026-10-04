<script lang="ts">
	import UserIcon from "phosphor-svelte/lib/UserIcon"

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
	<UserIcon class="profile-image-fallback" aria-hidden="true" />
{/if}

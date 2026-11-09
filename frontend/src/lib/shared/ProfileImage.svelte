<script lang="ts">
	import userIcon from "@phosphor-icons/core/regular/user.svg?raw"
	import Icon from "./Icon.svelte"

	let { url, fallbackIcon = userIcon }: { url: string; fallbackIcon?: string } =
		$props()
	let failed = $state("")
</script>

{#if url && failed !== url}
	<img
		class="block size-full bg-raised object-cover"
		src={url}
		alt=""
		decoding="async"
		draggable="false"
		referrerpolicy="no-referrer"
		onerror={() => (failed = url)} />
{:else}
	<Icon svg={fallbackIcon} class="size-1/2 text-fg-faint" aria-hidden="true" />
{/if}

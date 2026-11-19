<script lang="ts">
	import imageIcon from "@phosphor-icons/core/regular/image.svg?raw"
	import Icon from "../shared/Icon.svelte"

	// AvatarThumbnailImage shows one avatar image that Roblox renders, such as
	// the headshot, with its name. url is empty while Roblox renders it.
	let { label, url }: { label: string; url: string } = $props()

	let failedUrl = $state("")
</script>

<figure class="flex flex-col items-center gap-1">
	<div
		class="relative grid size-28 place-items-center overflow-hidden rounded-lg border border-line-subtle bg-canvas">
		{#if url && failedUrl !== url}
			{#key url}
				<img
					class="size-full animate-pop object-contain"
					src={url}
					alt={label}
					decoding="async"
					draggable="false"
					referrerpolicy="no-referrer"
					onerror={() => (failedUrl = url)} />
			{/key}
		{:else}
			<Icon
				svg={imageIcon}
				class="text-fg-faint"
				size={28}
				aria-label={`${label} unavailable`} />
		{/if}
	</div>
	<figcaption class="text-2xs text-fg-muted">{label}</figcaption>
</figure>

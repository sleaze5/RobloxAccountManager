<script lang="ts">
	import userIcon from "@phosphor-icons/core/regular/user.svg?raw"
	import Icon from "./Icon.svelte"

	// backgroundUrl is the avatar background that the user equipped, which
	// shows behind the transparent avatar image, as on the Roblox profile.
	// isContained fits a full-body image inside the frame instead of filling it.
	let {
		url,
		fallbackIcon = userIcon,
		backgroundUrl = "",
		isContained = false,
	}: {
		url: string
		fallbackIcon?: string
		backgroundUrl?: string
		isContained?: boolean
	} = $props()
	let failed = $state(""),
		failedBackground = $state("")

	const hasBackground = $derived(
		!!backgroundUrl && failedBackground !== backgroundUrl,
	)
</script>

{#if hasBackground}
	<img
		class="absolute inset-0 size-full animate-enter object-cover"
		src={backgroundUrl}
		alt=""
		decoding="async"
		draggable="false"
		referrerpolicy="no-referrer"
		onerror={() => (failedBackground = backgroundUrl)} />
{/if}
{#if url && failed !== url}
	<img
		class={[
			"relative block size-full",
			isContained ? "object-contain" : "object-cover",
			!hasBackground && "bg-raised",
		]}
		src={url}
		alt=""
		loading="lazy"
		decoding="async"
		draggable="false"
		referrerpolicy="no-referrer"
		onerror={() => (failed = url)} />
{:else}
	<Icon
		svg={fallbackIcon}
		class="relative size-1/2 text-fg-faint"
		aria-hidden="true" />
{/if}

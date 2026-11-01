<script lang="ts">
	import gameControllerIcon from "@phosphor-icons/core/regular/game-controller.svg?raw"
	import Icon from "../shared/Icon.svelte"
	type Size = "xs" | "sm" | "md" | "lg" | "xl"
	let { url, size = "md" }: { url: string; size?: Size } = $props()
	let failed = $state("")

	const sizes: Record<Size, string> = {
		xs: "size-5 rounded-sm",
		sm: "size-6 rounded-sm",
		md: "size-9 rounded-md",
		lg: "size-10 rounded-md",
		xl: "size-24 rounded-md",
	}
</script>

<span
	class={[
		"grid shrink-0 place-items-center overflow-hidden border border-line-strong bg-raised text-fg-faint",
		sizes[size],
	]}
	aria-hidden="true">
	{#if url && failed !== url}
		<img
			class="size-full object-cover"
			src={url}
			alt=""
			draggable="false"
			loading="lazy"
			decoding="async"
			referrerpolicy="no-referrer"
			onerror={() => (failed = url)} />
	{:else}
		<Icon
			svg={gameControllerIcon}
			size={size === "xl" ? 36 : size === "xs" ? 14 : 18}
			aria-hidden="true" />
	{/if}
</span>

<script lang="ts">
	import ProfileImage from "../shared/ProfileImage.svelte"
	import { formatRobux } from "./inventory-prices"

	let {
		name,
		detail,
		imageUrl,
		fallbackIcon,
		priceInRobux,
	}: {
		name: string
		detail: string
		imageUrl: string
		fallbackIcon: string
		priceInRobux: number
	} = $props()
</script>

<div class="flex items-center gap-3 rounded-lg border border-line-subtle bg-fill p-2">
	<span
		class="grid size-12 shrink-0 place-items-center overflow-hidden rounded-md bg-raised"
		aria-hidden="true">
		<ProfileImage url={imageUrl} {fallbackIcon} />
	</span>
	<div class="min-w-0 flex-1">
		<p class="truncate text-sm font-medium text-fg">{name}</p>
		<p class="truncate text-xs text-fg-muted">{detail}</p>
	</div>
	<span class="shrink-0 pr-1 font-mono text-sm font-semibold text-fg"
		>{priceInRobux === 0 ? "Free" : formatRobux(priceInRobux, false)}</span>
</div>

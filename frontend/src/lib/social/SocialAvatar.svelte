<script lang="ts">
	import type { UserPresence } from "../backend/bridge"
	import PresenceBadge from "../accounts/PresenceBadge.svelte"
	import ProfileImage from "../shared/ProfileImage.svelte"

	let {
		imageUrl,
		presence,
		className,
	}: { imageUrl: string; presence?: UserPresence; className: string } = $props()
</script>

<span class={["relative block shrink-0", className]}>
	<span
		class="relative grid size-full place-items-center overflow-hidden rounded-md border border-line-strong bg-raised"
		aria-hidden="true">
		<ProfileImage url={imageUrl} />
	</span>
	{#if presence}
		<PresenceBadge {presence} />
	{/if}
</span>

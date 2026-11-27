<script lang="ts">
	import type { OnlineStatus } from "../backend/bridge"
	import OnlineStatusBadge from "../accounts/OnlineStatusBadge.svelte"
	import ProfileImage from "../shared/ProfileImage.svelte"

	let {
		imageUrl,
		onlineStatus,
		className,
	}: { imageUrl: string; onlineStatus?: OnlineStatus; className: string } = $props()
</script>

<span class={["relative block shrink-0", className]}>
	<span
		class="relative grid size-full place-items-center overflow-hidden rounded-md border border-line-strong bg-raised"
		aria-hidden="true">
		<ProfileImage url={imageUrl} />
	</span>
	{#if onlineStatus}
		<OnlineStatusBadge {onlineStatus} />
	{/if}
</span>

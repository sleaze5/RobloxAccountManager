<script lang="ts">
	import type { UserPresence } from "../backend/bridge"
	import {
		presenceDotClass,
		presenceLabel,
		presenceTooltip,
	} from "../accounts/account-model"
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
		<span
			class={[
				"absolute -right-0.5 -bottom-0.5 size-3.5 cursor-help rounded-full border-2 border-surface ring-1 ring-line-strong",
				presenceDotClass(presence),
			]}
			role="img"
			aria-label={presenceLabel(presence)}
			data-tooltip={presenceTooltip(presence)}
			data-tooltip-side="top"></span>
	{/if}
</span>

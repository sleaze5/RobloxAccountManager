<script lang="ts">
	import ProfileImage from "../shared/ProfileImage.svelte"
	import type { Account } from "./account-model"
	import { presenceDotClass, presenceLabel, presenceTooltip } from "./account-model"

	let {
		account,
		className,
		showPresence = false,
	}: { account: Account; className: string; showPresence?: boolean } = $props()
</script>

<span class={["relative block shrink-0 font-mono font-semibold text-fg", className]}>
	<span
		class="relative grid size-full place-items-center overflow-hidden rounded-md border border-line-strong bg-raised"
		aria-hidden="true">
		<ProfileImage url={account.avatarUrl} />
	</span>
	{#if showPresence && account.id !== 0}
		<span
			class={[
				"absolute -right-0.5 -bottom-0.5 size-3.5 cursor-help rounded-full border-2 border-surface ring-1 ring-line-strong",
				presenceDotClass(account.presence),
			]}
			role="img"
			aria-label={presenceLabel(account.presence)}
			data-tooltip={presenceTooltip(account.presence)}
			data-tooltip-side="top"></span>
	{/if}
</span>

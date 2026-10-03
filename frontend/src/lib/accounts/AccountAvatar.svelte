<script lang="ts">
	import ProfileImage from "../shared/ProfileImage.svelte"
	import type { Account } from "./account-model"
	import { presenceClass, presenceLabel, presenceTooltip } from "./account-model"

	let {
		account,
		className,
		showPresence = false,
	}: { account: Account; className: string; showPresence?: boolean } = $props()
</script>

<span class={className}>
	<span class="avatar-image" aria-hidden="true">
		<ProfileImage url={account.avatarUrl} />
	</span>
	{#if showPresence && account.id !== 0}
		<span
			class={`presence-dot ${presenceClass(account.presence)}`}
			role="img"
			aria-label={presenceLabel(account.presence)}
			data-tooltip={presenceTooltip(account.presence)}
			data-tooltip-side="top"></span>
	{/if}
</span>

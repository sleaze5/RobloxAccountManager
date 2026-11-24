<script lang="ts">
	import ProfileImage from "../shared/ProfileImage.svelte"
	import type { Account } from "./account-model"
	import PresenceBadge from "./PresenceBadge.svelte"

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
		<PresenceBadge presence={account.presence} />
	{/if}
</span>

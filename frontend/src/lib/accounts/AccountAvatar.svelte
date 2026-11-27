<script lang="ts">
	import ProfileImage from "../shared/ProfileImage.svelte"
	import type { Account } from "./account-model"
	import OnlineStatusBadge from "./OnlineStatusBadge.svelte"

	let {
		account,
		className,
		showOnlineStatus = false,
	}: { account: Account; className: string; showOnlineStatus?: boolean } = $props()
</script>

<span class={["relative block shrink-0 font-mono font-semibold text-fg", className]}>
	<span
		class="relative grid size-full place-items-center overflow-hidden rounded-md border border-line-strong bg-raised"
		aria-hidden="true">
		<ProfileImage url={account.avatarUrl} />
	</span>
	{#if showOnlineStatus && account.id !== 0}
		<OnlineStatusBadge onlineStatus={account.onlineStatus} />
	{/if}
</span>

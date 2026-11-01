<script lang="ts">
	import usersIcon from "@phosphor-icons/core/regular/users.svg?raw"
	import Icon from "../shared/Icon.svelte"
	import type { ChatConversationView } from "../backend/bridge"
	import ProfileImage from "../shared/ProfileImage.svelte"

	let {
		conversation,
		className = "size-8",
	}: { conversation: ChatConversationView; className?: string } = $props()
</script>

<span
	class={[
		"relative grid shrink-0 place-items-center overflow-hidden rounded-md border border-line-strong bg-raised",
		className,
	]}
	aria-hidden="true">
	{#if conversation.group}
		<Icon svg={usersIcon} class="size-1/2 text-fg-faint" aria-hidden="true" />
	{:else}
		<ProfileImage url={conversation.imageUrl} />
	{/if}
</span>

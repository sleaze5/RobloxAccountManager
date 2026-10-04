<script lang="ts">
	import UsersIcon from "phosphor-svelte/lib/UsersIcon"
	import type { ChatConversationView } from "../backend/bridge"
	import ProfileImage from "../shared/ProfileImage.svelte"

	let { conversation }: { conversation: ChatConversationView } = $props()
</script>

<span class="chat-avatar avatar-image" aria-hidden="true">
	{#if conversation.group}
		<UsersIcon class="profile-image-fallback" aria-hidden="true" />
	{:else}
		<ProfileImage url={conversation.imageUrl} />
	{/if}
</span>

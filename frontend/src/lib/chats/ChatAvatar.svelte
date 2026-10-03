<script lang="ts">
	import Users from "@lucide/svelte/icons/users"
	import type { ChatConversationView } from "../backend/bridge"
	import ProfileImage from "../shared/ProfileImage.svelte"

	let { conversation }: { conversation: ChatConversationView } = $props()
</script>

<span class="chat-avatar avatar-image" aria-hidden="true">
	{#if conversation.group}
		<Users class="profile-image-fallback" />
	{:else}
		<ProfileImage url={conversation.imageUrl} />
	{/if}
</span>

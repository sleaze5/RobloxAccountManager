<script lang="ts">
	import CircleNotchIcon from "phosphor-svelte/lib/CircleNotchIcon"
	import ChatAvatar from "./ChatAvatar.svelte"
	import type { ChatState } from "./chat-state.svelte"
	import { chatListTime } from "./chat-time"

	let { chat }: { chat: ChatState } = $props()

	function loadMoreWhenVisible(node: HTMLElement, _cursor: string) {
		const observer = new IntersectionObserver(
			(entries) => {
				if (
					entries.some((entry) => entry.isIntersecting) &&
					!chat.loadMoreFailed
				) {
					void chat.loadMore()
				}
			},
			{ root: node.closest(".chat-sidebar"), rootMargin: "0px 0px 160px 0px" },
		)
		observer.observe(node)
		return {
			update() {
				observer.unobserve(node)
				observer.observe(node)
			},
			destroy: () => observer.disconnect(),
		}
	}
</script>

{#if chat.loading && chat.conversations.length === 0}
	<p class="chat-list-status" role="status">
		<CircleNotchIcon class="spinner" size={16} aria-hidden="true" />
		Loading chats
	</p>
{:else if chat.conversations.length === 0 && !chat.error}
	<p class="chat-list-status">No chats yet. Start one with New chat.</p>
{:else}
	<ul class="chat-list" aria-label="Chats">
		{#each chat.conversations as conversation (conversation.id)}
			<li>
				<button
					type="button"
					class="chat-row"
					class:unread={conversation.unreadCount > 0}
					aria-current={chat.openId === conversation.id ? "true" : undefined}
					onclick={() => void chat.open(conversation.id)}>
					<ChatAvatar {conversation} />
					<span class="chat-row-copy">
						<span class="chat-row-line">
							<strong>{conversation.title}</strong>
							<time class="chat-row-time">
								{chatListTime(conversation.updatedAtMs)}
							</time>
						</span>
						<span class="chat-row-line">
							<span class="chat-row-preview">{conversation.preview}</span>
							{#if conversation.unreadCount > 0}
								<span
									class="chat-unread-count"
									aria-label={`${conversation.unreadCount} unread`}>
									{conversation.unreadCount > 99
										? "99+"
										: conversation.unreadCount}
								</span>
							{/if}
						</span>
					</span>
				</button>
			</li>
		{/each}
	</ul>
	{#if chat.conversationsCursor}
		<div
			class="chat-list-footer"
			use:loadMoreWhenVisible={chat.conversationsCursor}>
			{#if chat.loadMoreFailed}
				<button
					type="button"
					class="chat-load-more"
					onclick={() => void chat.loadMore()}>
					Load more chats
				</button>
			{:else}
				<CircleNotchIcon
					class="spinner"
					size={16}
					aria-label="Loading more chats" />
			{/if}
		</div>
	{/if}
{/if}

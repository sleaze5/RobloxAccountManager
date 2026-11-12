<script lang="ts">
	import circleNotchIcon from "@phosphor-icons/core/regular/circle-notch.svg?raw"
	import Icon from "../shared/Icon.svelte"
	import SkeletonMediaRow from "../shared/SkeletonMediaRow.svelte"
	import SkeletonList from "../shared/SkeletonList.svelte"
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
			{
				root: node.closest("[data-chat-sidebar]"),
				rootMargin: "0px 0px 160px 0px",
			},
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
	<div class="-mx-2">
		<SkeletonList label="Loading chats">
			{#snippet row(index)}<SkeletonMediaRow {index} />{/snippet}
		</SkeletonList>
	</div>
{:else if chat.conversations.length === 0 && !chat.error}
	<p class="flex items-center gap-2 p-2 text-xs text-fg-faint">
		No chats yet. Start one with New chat.
	</p>
{:else}
	<ul class="-mx-2 flex flex-col" aria-label="Chats">
		{#each chat.conversations as conversation (conversation.id)}
			<li>
				<button
					type="button"
					class="group/row list-row flex min-h-12 w-full items-center gap-2 border-b border-line-subtle px-3 py-2 text-left"
					data-selected={chat.openId === conversation.id || undefined}
					data-unread={conversation.unreadCount > 0 || undefined}
					aria-current={chat.openId === conversation.id ? "true" : undefined}
					onclick={() => void chat.open(conversation.id)}>
					<ChatAvatar {conversation} selfUserId={chat.selfUserId} />
					<span class="flex min-w-0 flex-1 flex-col">
						<span class="flex min-w-0 items-center gap-2">
							<strong
								class="min-w-0 flex-1 truncate text-sm font-medium text-fg group-data-unread/row:font-semibold group-data-unread/row:text-fg-bright"
								>{conversation.title}</strong>
							<time
								class="shrink-0 text-xs text-fg-faint group-hover/row:text-fg-muted group-data-selected/row:text-fg-muted">
								{chatListTime(conversation.updatedAtMs)}
							</time>
						</span>
						<span class="flex min-w-0 items-center gap-2">
							<span
								class="min-w-0 flex-1 truncate text-xs text-fg-faint group-hover/row:text-fg-muted group-data-selected/row:text-fg-muted group-data-unread/row:text-fg-muted"
								>{conversation.preview}</span>
							{#if conversation.unreadCount > 0}
								<span
									class="min-w-5 animate-bloom rounded-sm bg-solid px-1 text-center text-2xs leading-4 font-bold text-on-solid"
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
			class="flex min-h-7 items-center justify-center text-fg-faint"
			use:loadMoreWhenVisible={chat.conversationsCursor}>
			{#if chat.loadMoreFailed}
				<button
					type="button"
					class="button self-center"
					onclick={() => void chat.loadMore()}>
					Load more chats
				</button>
			{:else}
				<Icon
					svg={circleNotchIcon}
					class="spinner"
					size={16}
					aria-label="Loading more chats" />
			{/if}
		</div>
	{/if}
{/if}

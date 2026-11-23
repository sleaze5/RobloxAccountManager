<script lang="ts">
	import LoadMore from "../shared/LoadMore.svelte"
	import Icon from "../shared/Icon.svelte"
	import SkeletonMediaRow from "../shared/SkeletonMediaRow.svelte"
	import SkeletonList from "../shared/SkeletonList.svelte"
	import { moveListFocus } from "../shared/move-shortcut"
	import ChatAvatar from "./ChatAvatar.svelte"
	import type { ChatState } from "./chat-state.svelte"
	import { chatListTime } from "./chat-time"

	let { chat }: { chat: ChatState } = $props()
	const tabStopConversationId = $derived(
		(
			chat.conversations.find(
				(conversation) => conversation.id === chat.openId,
			) ?? chat.conversations[0]
		)?.id,
	)
</script>

{#if chat.loading && chat.conversations.length === 0}
	<div class="-mx-2">
		<SkeletonList label="Loading chats…">
			{#snippet row(index)}<SkeletonMediaRow {index} />{/snippet}
		</SkeletonList>
	</div>
{:else if chat.conversations.length === 0 && !chat.listError}
	<p class="flex items-center gap-2 p-2 text-xs text-fg-faint">
		No chats yet. Start one with New chat.
	</p>
{:else}
	<ul class="-mx-2 flex flex-col" aria-label="Chats" data-list>
		{#each chat.conversations as conversation (conversation.id)}
			<li>
				<button
					type="button"
					class="group/row list-row flex min-h-12 w-full items-center gap-2 border-b border-line-subtle px-3 py-2 text-left"
					data-selected={chat.openId === conversation.id || undefined}
					data-unread={conversation.unreadCount > 0 || undefined}
					aria-current={chat.openId === conversation.id ? "true" : undefined}
					data-list-item
					tabindex={conversation.id === tabStopConversationId ? 0 : -1}
					onclick={() => void chat.open(conversation.id)}
					onkeydown={moveListFocus}>
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
		<LoadMore
			pageKey={chat.conversationsCursor}
			label="Load more chats"
			loadingLabel="Loading more chats…"
			isLoadingMore={chat.loadingMore}
			hasFailed={chat.loadMoreFailed}
			onLoadMore={() => void chat.loadMore()} />
	{/if}
{/if}

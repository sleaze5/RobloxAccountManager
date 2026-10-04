<script lang="ts">
	import EyeIcon from "phosphor-svelte/lib/EyeIcon"
	import CircleNotchIcon from "phosphor-svelte/lib/CircleNotchIcon"
	import PaperPlaneRightIcon from "phosphor-svelte/lib/PaperPlaneRightIcon"
	import { tick } from "svelte"
	import type { ChatConversationView } from "../backend/bridge"
	import ChatAvatar from "./ChatAvatar.svelte"
	import type { ChatState, PendingChatMessage } from "./chat-state.svelte"
	import { chatMessageTime, chatTimestampBreakMs } from "./chat-time"

	type TimelineItem =
		| { kind: "time"; key: string; ms: number }
		| { kind: "system"; key: string; text: string }
		| {
				kind: "message"
				key: string
				mine: boolean
				sender: string
				text: string
				moderated: boolean
				pending?: PendingChatMessage
		  }

	let {
		chat,
		conversation,
		selfUserId,
	}: {
		chat: ChatState
		conversation: ChatConversationView
		selfUserId: number
	} = $props()

	let draft = $state(""),
		scroller = $state<HTMLElement | undefined>(undefined)

	const names = $derived(
			new Map(
				(conversation.participants ?? []).map((participant) => [
					participant.userId,
					participant.displayName || participant.username,
				]),
			),
		),
		timeline = $derived(buildTimeline()),
		lastKey = $derived(timeline.at(-1)?.key ?? "")

	function buildTimeline(): TimelineItem[] {
		const items: TimelineItem[] = []
		let previousMs = 0,
			previousSender: number | null = null
		const entries = [
			...chat.messages.map((message) => ({
				key: message.id,
				ms: message.createdAtMs,
				sender: message.senderUserId,
				text: message.text,
				system: message.system,
				moderated: message.moderated,
				pending: undefined,
			})),
			...chat.pending
				.filter((pending) => pending.conversationId === conversation.id)
				.map((pending) => ({
					key: `pending-${pending.key}`,
					ms: pending.createdAtMs,
					sender: selfUserId,
					text: pending.text,
					system: false,
					moderated: false,
					pending,
				})),
		]
		for (const entry of entries) {
			if (
				entry.ms &&
				(!previousMs || entry.ms - previousMs > chatTimestampBreakMs)
			) {
				items.push({ kind: "time", key: `time-${entry.key}`, ms: entry.ms })
				previousSender = null
			}
			previousMs = entry.ms || previousMs
			if (entry.system) {
				items.push({ kind: "system", key: entry.key, text: entry.text })
				previousSender = null
				continue
			}
			const mine = entry.sender === selfUserId
			items.push({
				kind: "message",
				key: entry.key,
				mine,
				sender:
					conversation.group && !mine && previousSender !== entry.sender
						? (names.get(entry.sender) ?? "Unknown user")
						: "",
				text: entry.text,
				moderated: entry.moderated,
				pending: entry.pending,
			})
			previousSender = entry.sender
		}
		return items
	}

	$effect(() => {
		void lastKey
		void tick().then(() => scroller?.scrollTo({ top: scroller.scrollHeight }))
	})

	async function loadOlder(): Promise<void> {
		if (!scroller) return
		const fromBottom = scroller.scrollHeight - scroller.scrollTop
		await chat.loadOlder()
		await tick()
		scroller.scrollTop = scroller.scrollHeight - fromBottom
	}

	function send(): void {
		if (!draft.trim()) return
		chat.send(draft)
		draft = ""
	}

	function handleKeydown(event: KeyboardEvent): void {
		if (event.key === "Enter" && !event.shiftKey && !event.isComposing) {
			event.preventDefault()
			send()
		}
	}
</script>

<section class="chat-thread" aria-labelledby="chat-thread-title">
	<header class="chat-thread-header">
		<ChatAvatar {conversation} />
		<h2 id="chat-thread-title">{conversation.title}</h2>
		{#if conversation.group}
			<span class="chat-thread-meta">
				{conversation.participants?.length ?? 0} members
			</span>
		{/if}
	</header>

	<div class="chat-timeline" bind:this={scroller}>
		{#if chat.messagesCursor && !chat.messagesLoading}
			<button
				type="button"
				class="chat-load-more"
				disabled={chat.olderLoading}
				aria-busy={chat.olderLoading}
				onclick={() => void loadOlder()}>
				{#if chat.olderLoading}
					<CircleNotchIcon class="spinner" size={15} aria-hidden="true" />
				{/if}
				Load older messages
			</button>
		{/if}
		{#if chat.messagesError}
			<p class="settings-inline-error" role="alert">
				{chat.messagesError}
				<button type="button" onclick={() => void chat.reloadMessages()}>
					Retry
				</button>
			</p>
		{/if}
		{#if chat.messagesLoading && chat.messages.length === 0}
			<p class="chat-list-status" role="status">
				<CircleNotchIcon class="spinner" size={16} aria-hidden="true" />
				Loading messages
			</p>
		{:else if timeline.length === 0 && !chat.messagesError}
			<p class="chat-list-status">No messages yet. Say hello.</p>
		{/if}
		<ol class="chat-messages" aria-label="Messages">
			{#each timeline as item (item.key)}
				{#if item.kind === "time"}
					<li class="chat-time-break">{chatMessageTime(item.ms)}</li>
				{:else if item.kind === "system"}
					<li class="chat-system-message">{item.text}</li>
				{:else}
					<li class="chat-message" class:mine={item.mine}>
						{#if item.sender}<span class="chat-sender">{item.sender}</span
							>{/if}
						<p
							class="chat-bubble"
							class:pending={item.pending && !item.pending.failed}
							class:failed={item.pending?.failed || item.moderated}>
							{item.text}
						</p>
						{#if item.moderated}
							<span class="chat-message-note failed">
								Not delivered: Roblox moderated this message.
							</span>
						{:else if item.pending?.failed}
							{@const pending = item.pending}
							<span class="chat-message-note failed" role="alert">
								{pending.error}
								<button
									type="button"
									onclick={() => chat.retry(pending.key)}>
									Retry
								</button>
								<button
									type="button"
									onclick={() => chat.discard(pending.key)}>
									Discard
								</button>
							</span>
						{:else if item.pending}
							<span class="chat-message-note">Sending</span>
						{/if}
					</li>
				{/if}
			{/each}
		</ol>
	</div>

	{#if chat.actionError}
		<p class="settings-inline-error chat-action-error" role="alert">
			{chat.actionError}
		</p>
	{/if}
	<form
		novalidate
		class="chat-bar"
		onsubmit={(event) => {
			event.preventDefault()
			send()
		}}>
		<textarea
			bind:value={draft}
			rows="1"
			aria-label={`Message ${conversation.title}`}
			placeholder="Write a message"
			onkeydown={handleKeydown}></textarea>
		<button
			type="button"
			class="icon-action"
			aria-label="Mark as read"
			data-tooltip="Mark as read"
			aria-busy={chat.markingRead}
			disabled={chat.markingRead}
			onclick={() => void chat.markRead()}>
			{#if chat.markingRead}
				<CircleNotchIcon class="spinner" size={17} aria-hidden="true" />
			{:else}
				<EyeIcon size={17} aria-hidden="true" />
			{/if}
		</button>
		<button
			type="submit"
			class="icon-action"
			aria-label="Send message"
			data-tooltip="Send message"
			disabled={!draft.trim()}>
			<PaperPlaneRightIcon size={17} aria-hidden="true" />
		</button>
	</form>
</section>

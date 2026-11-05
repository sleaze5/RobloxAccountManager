<script lang="ts">
	import circleNotchIcon from "@phosphor-icons/core/regular/circle-notch.svg?raw"
	import eyeIcon from "@phosphor-icons/core/regular/eye.svg?raw"
	import paperPlaneRightIcon from "@phosphor-icons/core/regular/paper-plane-right.svg?raw"
	import Icon from "../shared/Icon.svelte"
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
	const noteClass = "flex flex-wrap items-center justify-end gap-2 px-1 text-2xs",
		linkButtonClass =
			"rounded-sm text-2xs font-medium text-fg underline underline-offset-2"
</script>

<section
	class="flex min-h-0 animate-enter flex-col"
	aria-labelledby="chat-thread-title">
	<header
		class="flex min-w-0 shrink-0 items-center gap-2 border-b border-line-subtle px-3 py-2">
		<ChatAvatar {conversation} className="size-6" />
		<h2
			id="chat-thread-title"
			class="min-w-0 truncate text-sm font-semibold text-fg">
			{conversation.title}
		</h2>
		{#if conversation.group}
			<span class="shrink-0 text-xs text-fg-muted">
				{conversation.participants?.length ?? 0} members
			</span>
		{/if}
	</header>

	<div
		class="flex min-h-0 flex-1 flex-col gap-2 overflow-y-auto p-3"
		bind:this={scroller}>
		{#if chat.messagesCursor && !chat.messagesLoading}
			<button
				type="button"
				class="btn self-center"
				disabled={chat.olderLoading}
				aria-busy={chat.olderLoading}
				onclick={() => void loadOlder()}>
				{#if chat.olderLoading}
					<Icon
						svg={circleNotchIcon}
						class="spinner"
						size={16}
						aria-hidden="true" />
				{/if}
				Load older messages
			</button>
		{/if}
		{#if chat.messagesError}
			<p
				class="flex items-center justify-between gap-2 notice-error"
				role="alert">
				{chat.messagesError}
				<button
					class={linkButtonClass}
					type="button"
					onclick={() => void chat.reloadMessages()}>
					Retry
				</button>
			</p>
		{/if}
		{#if chat.messagesLoading && chat.messages.length === 0}
			<p class="flex items-center gap-2 p-2 text-xs text-fg-faint" role="status">
				<Icon
					svg={circleNotchIcon}
					class="spinner"
					size={16}
					aria-hidden="true" />
				Loading messages
			</p>
		{:else if timeline.length === 0 && !chat.messagesError}
			<p class="flex items-center gap-2 p-2 text-xs text-fg-faint">
				No messages yet. Say hello.
			</p>
		{/if}
		<ol class="flex flex-col gap-0.5" aria-label="Messages">
			{#each timeline as item (item.key)}
				{#if item.kind === "time"}
					<li
						class="self-center pt-2 pb-1 text-center text-2xs text-fg-faint">
						{chatMessageTime(item.ms)}
					</li>
				{:else if item.kind === "system"}
					<li
						class="max-w-4/5 self-center pt-2 pb-1 text-center text-2xs text-fg-muted italic">
						{item.text}
					</li>
				{:else}
					<li
						class={[
							"flex max-w-3/4 flex-col gap-0.5",
							item.mine ? "items-end self-end" : "items-start self-start",
						]}>
						{#if item.sender}<span
								class="px-1 pt-1 text-2xs font-medium text-fg-muted"
								>{item.sender}</span
							>{/if}
						<p
							class={[
								"rounded-xl border px-3 py-2 text-sm leading-snug wrap-anywhere whitespace-pre-wrap text-fg select-text",
								item.pending && !item.pending.failed && "opacity-60",
								item.mine && (item.pending?.failed || item.moderated)
									? "border-danger/40 bg-danger/5"
									: item.mine
										? "border-line-strong bg-raised"
										: "border-line bg-fill",
							]}>
							{item.text}
						</p>
						{#if item.moderated}
							<span class={[noteClass, "text-danger-muted"]}>
								Not delivered: Roblox moderated this message.
							</span>
						{:else if item.pending?.failed}
							{@const pending = item.pending}
							<span class={[noteClass, "text-danger-muted"]} role="alert">
								{pending.error}
								<button
									class={linkButtonClass}
									type="button"
									onclick={() => chat.retry(pending.key)}>
									Retry
								</button>
								<button
									class={linkButtonClass}
									type="button"
									onclick={() => chat.discard(pending.key)}>
									Discard
								</button>
							</span>
						{:else if item.pending}
							<span class={[noteClass, "text-fg-faint"]}>Sending…</span>
						{/if}
					</li>
				{/if}
			{/each}
		</ol>
	</div>

	{#if chat.actionError}
		<p class="mx-3 mb-2 notice-error" role="alert">
			{chat.actionError}
		</p>
	{/if}
	<form
		novalidate
		class="flex shrink-0 items-end gap-1 border-t border-line-subtle bg-surface px-3 py-2"
		onsubmit={(event) => {
			event.preventDefault()
			send()
		}}>
		<textarea
			bind:value={draft}
			class="field-sizing-content max-h-24 min-h-8 min-w-0 flex-1 resize-none rounded-md border border-line bg-transparent px-3 py-2 text-sm leading-snug text-fg transition-colors duration-100 placeholder:text-fg-faint focus-visible:border-line-strong focus-visible:outline-0"
			rows="1"
			aria-label={`Message ${conversation.title}`}
			placeholder="Write a message"
			onkeydown={handleKeydown}></textarea>
		<button
			type="button"
			class="btn-ghost size-8"
			aria-label="Mark as read"
			data-tooltip="Mark as read"
			aria-busy={chat.markingRead}
			disabled={chat.markingRead}
			onclick={() => void chat.markRead()}>
			{#if chat.markingRead}
				<Icon
					svg={circleNotchIcon}
					class="spinner"
					size={16}
					aria-hidden="true" />
			{:else}
				<Icon svg={eyeIcon} size={16} aria-hidden="true" />
			{/if}
		</button>
		<button
			type="submit"
			class="btn-ghost size-8"
			aria-label="Send message"
			data-tooltip="Send message"
			disabled={!draft.trim()}>
			<Icon svg={paperPlaneRightIcon} size={16} aria-hidden="true" />
		</button>
	</form>
</section>

<script lang="ts">
	import circleNotchIcon from "@phosphor-icons/core/regular/circle-notch.svg?raw"
	import eyeIcon from "@phosphor-icons/core/regular/eye.svg?raw"
	import paperPlaneRightIcon from "@phosphor-icons/core/regular/paper-plane-right.svg?raw"
	import pencilSimpleIcon from "@phosphor-icons/core/regular/pencil-simple.svg?raw"
	import userPlusIcon from "@phosphor-icons/core/regular/user-plus.svg?raw"
	import usersThreeIcon from "@phosphor-icons/core/regular/users-three.svg?raw"
	import xIcon from "@phosphor-icons/core/regular/x.svg?raw"
	import SocialAvatar from "../social/SocialAvatar.svelte"
	import ProfileImage from "../shared/ProfileImage.svelte"
	import Icon from "../shared/Icon.svelte"
	import { tick } from "svelte"
	import type { ChatConversationView } from "../backend/bridge"
	import ChatAvatar from "./ChatAvatar.svelte"
	import type { ChatState, PendingChatMessage } from "./chat-state.svelte"
	import { chatMessageTime, chatTimestampBreakMs } from "./chat-time"

	type TimelineItem =
		| { kind: "time"; key: string; createdAtMs: number }
		| { kind: "system"; key: string; text: string }
		| {
				kind: "message"
				key: string
				mine: boolean
				sender: string
				senderImageUrl: string
				startsRun: boolean
				text: string
				moderated: boolean
				pending?: PendingChatMessage
		  }

	let {
		chat,
		conversation,
		selfUserId,
		onAddMembers,
	}: {
		chat: ChatState
		conversation: ChatConversationView
		selfUserId: number
		onAddMembers: () => void
	} = $props()

	let draft = $state(""),
		scroller = $state<HTMLElement | undefined>(undefined),
		membersOpen = $state(false),
		renamingOpen = $state(false),
		groupName = $state(""),
		renameInput = $state<HTMLInputElement | undefined>(undefined)

	const participants = $derived(conversation.participants ?? []),
		participantsById = $derived(
			new Map(
				participants.map((participant) => [participant.userId, participant]),
			),
		),
		memberSlots = $derived(
			Math.max(0, chat.groupMemberLimit - participants.length),
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
				createdAtMs: message.createdAtMs,
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
					createdAtMs: pending.createdAtMs,
					sender: selfUserId,
					text: pending.text,
					system: false,
					moderated: false,
					pending,
				})),
		]
		for (const entry of entries) {
			if (
				entry.createdAtMs &&
				(!previousMs || entry.createdAtMs - previousMs > chatTimestampBreakMs)
			) {
				items.push({
					kind: "time",
					key: `time-${entry.key}`,
					createdAtMs: entry.createdAtMs,
				})
				previousSender = null
			}
			previousMs = entry.createdAtMs || previousMs
			if (entry.system) {
				items.push({ kind: "system", key: entry.key, text: entry.text })
				previousSender = null
				continue
			}
			items.push(messageItem(entry, previousSender !== entry.sender))
			previousSender = entry.sender
		}
		return items
	}

	function messageItem(
		entry: {
			key: string
			sender: number
			text: string
			moderated: boolean
			pending?: PendingChatMessage
		},
		startsRun: boolean,
	): TimelineItem {
		const mine = entry.sender === selfUserId,
			sender = participantsById.get(entry.sender),
			senderName = sender?.displayName || sender?.username || "Unknown user"
		return {
			kind: "message",
			key: entry.key,
			mine,
			sender: conversation.group && !mine && startsRun ? senderName : "",
			senderImageUrl: sender?.imageUrl ?? "",
			startsRun,
			text: entry.text,
			moderated: entry.moderated,
			pending: entry.pending,
		}
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

	async function startRename(): Promise<void> {
		groupName = conversation.title
		chat.renameError = ""
		renamingOpen = true
		await tick()
		renameInput?.select()
	}

	async function saveRename(): Promise<void> {
		if (groupName.trim() === conversation.title) {
			renamingOpen = false
			return
		}
		if (await chat.rename(groupName)) renamingOpen = false
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

<div class="flex min-h-0 min-w-0">
	<section
		class="flex min-h-0 min-w-0 flex-1 animate-enter flex-col"
		aria-labelledby="chat-thread-title">
		<header
			class="flex h-12 min-w-0 shrink-0 items-center gap-2 border-b border-line-subtle px-3">
			<ChatAvatar {conversation} {selfUserId} className="size-8" />
			{#if renamingOpen}
				<form
					novalidate
					class="flex min-w-0 flex-1 animate-enter items-center gap-1"
					onsubmit={(event) => {
						event.preventDefault()
						void saveRename()
					}}>
					<input
						bind:this={renameInput}
						bind:value={groupName}
						class="field h-7 min-w-0 flex-1"
						aria-label="Group name"
						aria-invalid={chat.renameError ? "true" : undefined}
						maxlength={chat.groupNameMaxLength || undefined}
						disabled={chat.renaming}
						onkeydown={(event) => {
							if (event.key === "Escape") {
								event.preventDefault()
								event.stopPropagation()
								renamingOpen = false
							}
						}} />
					<button
						class="button-primary"
						type="submit"
						aria-busy={chat.renaming}
						disabled={chat.renaming || !groupName.trim()}>
						{chat.renaming ? "Renaming…" : "Rename"}
					</button>
					<button
						class="button-ghost-sm"
						type="button"
						aria-label="Cancel renaming"
						disabled={chat.renaming}
						onclick={() => (renamingOpen = false)}>
						<Icon svg={xIcon} size={14} aria-hidden="true" />
					</button>
				</form>
			{:else}
				<div class="flex min-w-0 flex-1 flex-col leading-snug">
					<h2
						id="chat-thread-title"
						class="min-w-0 truncate text-sm font-semibold text-fg">
						{conversation.title}
					</h2>
					{#if conversation.group}
						<span class="truncate text-2xs text-fg-faint">
							{participants.length} members
						</span>
					{:else if participants[0]?.username}
						<span class="truncate text-2xs text-fg-faint"
							>@{participants[0].username}</span>
					{/if}
				</div>
				{#if conversation.group}
					<button
						class="button-ghost"
						type="button"
						aria-label="Rename group"
						data-tooltip="Rename group"
						onclick={() => void startRename()}>
						<Icon svg={pencilSimpleIcon} size={16} aria-hidden="true" />
					</button>
					<button
						class="button-ghost"
						type="button"
						aria-label="Add members"
						aria-haspopup="dialog"
						data-tooltip={memberSlots > 0
							? "Add members"
							: "This group is full"}
						disabled={memberSlots === 0}
						onclick={onAddMembers}>
						<Icon svg={userPlusIcon} size={16} aria-hidden="true" />
					</button>
					<button
						class="button-ghost"
						type="button"
						aria-label="Members"
						aria-expanded={membersOpen}
						aria-controls="chat-members"
						data-open={membersOpen || undefined}
						data-tooltip={membersOpen ? "Hide members" : "Show members"}
						onclick={() => (membersOpen = !membersOpen)}>
						<Icon svg={usersThreeIcon} size={16} aria-hidden="true" />
					</button>
				{/if}
			{/if}
		</header>
		{#if chat.renameError && renamingOpen}
			<p class="mx-3 mt-2 notice-error" role="alert">{chat.renameError}</p>
		{/if}

		<div
			class="flex min-h-0 flex-1 flex-col gap-2 overflow-y-auto p-3"
			bind:this={scroller}>
			{#if chat.messagesCursor && !chat.messagesLoading}
				<button
					type="button"
					class="button self-center"
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
				<p
					class="flex items-center gap-2 p-2 text-xs text-fg-faint"
					role="status">
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
				{#each timeline as entry (entry.key)}
					{#if entry.kind === "time"}
						<li
							class="self-center pt-2 pb-1 text-center text-2xs text-fg-faint">
							{chatMessageTime(entry.createdAtMs)}
						</li>
					{:else if entry.kind === "system"}
						<li
							class="max-w-4/5 self-center pt-2 pb-1 text-center text-2xs text-fg-muted italic">
							{entry.text}
						</li>
					{:else}
						<li
							class={[
								"flex max-w-3/4 gap-2",
								entry.mine ? "self-end" : "self-start",
							]}>
							{#if conversation.group && !entry.mine}
								<span class="mt-5 size-6 shrink-0" aria-hidden="true">
									{#if entry.startsRun}<span
											class="relative grid size-full animate-pop place-items-center overflow-hidden rounded-md border border-line-strong bg-raised"
											><ProfileImage
												url={entry.senderImageUrl} /></span
										>{/if}
								</span>
							{/if}
							<div
								class={[
									"flex min-w-0 flex-col gap-0.5",
									entry.mine ? "items-end" : "items-start",
								]}>
								{#if entry.sender}<span
										class="px-1 pt-1 text-2xs font-medium text-fg-muted"
										>{entry.sender}</span
									>{/if}
								<p
									class={[
										"rounded-xl border px-3 py-2 text-sm leading-snug wrap-anywhere whitespace-pre-wrap text-fg select-text",
										entry.pending &&
											!entry.pending.failed &&
											"opacity-60",
										entry.mine &&
										(entry.pending?.failed || entry.moderated)
											? "border-danger/40 bg-danger/5"
											: entry.mine
												? "border-line-strong bg-raised"
												: "border-line bg-fill",
									]}>
									{entry.text}
								</p>
								{#if entry.moderated}
									<span class={[noteClass, "text-danger-muted"]}>
										Not delivered: Roblox moderated this message.
									</span>
								{:else if entry.pending?.failed}
									{@const pending = entry.pending}
									<span
										class={[noteClass, "text-danger-muted"]}
										role="alert">
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
								{:else if entry.pending}
									<span class={[noteClass, "text-fg-faint"]}
										>Sending…</span>
								{/if}
							</div>
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
				class="button-ghost size-8"
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
				class="button-ghost size-8"
				aria-label="Send message"
				data-tooltip="Send message"
				disabled={!draft.trim()}>
				<Icon svg={paperPlaneRightIcon} size={16} aria-hidden="true" />
			</button>
		</form>
	</section>
	{#if conversation.group && membersOpen}
		<aside
			id="chat-members"
			class="flex w-52 shrink-0 animate-enter flex-col border-l border-line-subtle bg-surface"
			aria-labelledby="chat-members-title">
			<div
				class="flex h-12 shrink-0 items-center justify-between gap-2 border-b border-line-subtle px-3">
				<h3 id="chat-members-title" class="text-xs font-semibold text-fg-muted">
					Members
					<span class="font-mono font-normal text-fg-faint"
						>{participants.length}/{chat.groupMemberLimit}</span>
				</h3>
			</div>
			<ul class="min-h-0 flex-1 overflow-y-auto" aria-label="Members">
				{#each participants as participant (participant.userId)}
					<li
						class="flex h-12 items-center gap-2 border-b border-line-subtle px-3">
						<SocialAvatar
							imageUrl={participant.imageUrl}
							className="size-8" />
						<span class="flex min-w-0 flex-1 flex-col leading-snug">
							<span class="truncate text-sm font-medium text-fg"
								>{participant.displayName ||
									participant.username}</span>
							<span class="truncate text-2xs text-fg-faint"
								>@{participant.username}</span>
						</span>
						{#if participant.userId === selfUserId}
							<span class="tag-pill shrink-0">You</span>
						{/if}
					</li>
				{/each}
			</ul>
			<div class="shrink-0 border-t border-line-subtle p-2">
				<button
					class="button w-full"
					type="button"
					aria-haspopup="dialog"
					disabled={memberSlots === 0}
					onclick={onAddMembers}>
					<Icon svg={userPlusIcon} size={14} aria-hidden="true" />
					{memberSlots > 0 ? "Add members" : "Group is full"}
				</button>
			</div>
		</aside>
	{/if}
</div>

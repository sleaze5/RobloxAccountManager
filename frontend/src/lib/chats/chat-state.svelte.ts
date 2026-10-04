import { accountBackend } from "../backend/bridge"
import type { ChatConversationView, ChatMessageView } from "../backend/bridge"
import type { AccountStore } from "../accounts/account-store.svelte"

export interface PendingChatMessage {
	key: number
	conversationId: string
	text: string
	createdAtMs: number
	failed: boolean
	error: string
}

export class ChatState {
	conversations = $state<ChatConversationView[]>([])
	conversationsCursor = $state("")
	loading = $state(false)
	loadingMore = $state(false)
	loadMoreFailed = $state(false)
	error = $state("")
	openId = $state("")
	messages = $state<ChatMessageView[]>([])
	messagesCursor = $state("")
	messagesLoading = $state(false)
	olderLoading = $state(false)
	messagesError = $state("")
	pending = $state<PendingChatMessage[]>([])
	markingRead = $state(false)
	markingAll = $state(false)
	creating = $state(false)
	createError = $state("")
	actionError = $state("")
	#active = true
	#listGeneration = 0
	#messagesGeneration = 0
	#nextPendingKey = 0
	readonly #accountId: number
	readonly #store: AccountStore

	constructor(accountId: number, store: AccountStore) {
		this.#accountId = accountId
		this.#store = store
	}

	get openConversation(): ChatConversationView | undefined {
		return this.conversations.find((item) => item.id === this.openId)
	}

	#current(): boolean {
		return (
			this.#active &&
			this.#store.vault.unlocked &&
			this.#store.selectedAccount.id === this.#accountId
		)
	}

	async refresh(): Promise<void> {
		if (!this.#current() || this.loading) return
		const generation = ++this.#listGeneration
		this.loading = true
		this.error = ""
		this.loadMoreFailed = false
		try {
			const page = await accountBackend.GetChatConversations(this.#accountId, "")
			if (!this.#current() || generation !== this.#listGeneration) return
			const conversations = page.conversations ?? [],
				open = this.openConversation
			if (open && !conversations.some((item) => item.id === open.id)) {
				conversations.unshift(open)
			}
			this.conversations = conversations
			this.conversationsCursor = page.nextCursor
		} catch (error) {
			if (this.#current() && generation === this.#listGeneration) {
				this.error = message(error, "Could not load chats. Try again.")
			}
		} finally {
			if (generation === this.#listGeneration) this.loading = false
		}
		if (this.openId) await this.#loadMessages(this.openId)
	}

	async loadMore(): Promise<void> {
		if (
			!this.#current() ||
			this.loading ||
			this.loadingMore ||
			!this.conversationsCursor
		) {
			return
		}
		const generation = this.#listGeneration
		this.loadingMore = true
		this.loadMoreFailed = false
		this.error = ""
		try {
			const page = await accountBackend.GetChatConversations(
				this.#accountId,
				this.conversationsCursor,
			)
			if (!this.#current() || generation !== this.#listGeneration) return
			const incoming = page.conversations ?? [],
				ids = new Set(incoming.map((item) => item.id))
			this.conversations = [
				...this.conversations.filter((item) => !ids.has(item.id)),
				...incoming,
			]
			this.conversationsCursor = page.nextCursor
		} catch (error) {
			if (this.#current() && generation === this.#listGeneration) {
				this.error = message(error, "Could not load more chats. Try again.")
				this.loadMoreFailed = true
			}
		} finally {
			this.loadingMore = false
		}
	}

	async open(conversationId: string): Promise<void> {
		if (!this.#current() || this.openId === conversationId) return
		this.openId = conversationId
		this.messages = []
		this.messagesCursor = ""
		this.actionError = ""
		await this.#loadMessages(conversationId)
	}

	async #loadMessages(conversationId: string): Promise<void> {
		const generation = ++this.#messagesGeneration
		this.messagesLoading = true
		this.olderLoading = false
		this.messagesError = ""
		try {
			const page = await accountBackend.GetChatMessages(
				this.#accountId,
				conversationId,
				"",
			)
			if (!this.#current() || generation !== this.#messagesGeneration) return
			this.messages = page.messages ?? []
			this.messagesCursor = page.nextCursor
		} catch (error) {
			if (this.#current() && generation === this.#messagesGeneration) {
				this.messagesError = message(
					error,
					"Could not load this chat. Try again.",
				)
			}
		} finally {
			if (generation === this.#messagesGeneration) this.messagesLoading = false
		}
	}

	async reloadMessages(): Promise<void> {
		if (this.#current() && this.openId && !this.messagesLoading) {
			await this.#loadMessages(this.openId)
		}
	}

	async loadOlder(): Promise<void> {
		if (
			!this.#current() ||
			!this.openId ||
			!this.messagesCursor ||
			this.messagesLoading ||
			this.olderLoading
		) {
			return
		}
		const generation = this.#messagesGeneration
		this.olderLoading = true
		this.messagesError = ""
		try {
			const page = await accountBackend.GetChatMessages(
				this.#accountId,
				this.openId,
				this.messagesCursor,
			)
			if (!this.#current() || generation !== this.#messagesGeneration) return
			const known = new Set(this.messages.map((item) => item.id))
			this.messages = [
				...(page.messages ?? []).filter((item) => !known.has(item.id)),
				...this.messages,
			]
			this.messagesCursor = page.nextCursor
		} catch (error) {
			if (this.#current() && generation === this.#messagesGeneration) {
				this.messagesError = message(error, "Could not load older messages.")
			}
		} finally {
			if (generation === this.#messagesGeneration) this.olderLoading = false
		}
	}

	async markRead(): Promise<void> {
		const conversationId = this.openId
		if (!this.#current() || !conversationId || this.markingRead) return
		this.markingRead = true
		this.actionError = ""
		try {
			await accountBackend.MarkChatConversationRead(
				this.#accountId,
				conversationId,
			)
			if (this.#current()) this.#clearUnread(new Set([conversationId]))
		} catch (error) {
			if (this.#current()) {
				this.actionError = message(error, "Could not mark this chat as read.")
			}
		} finally {
			this.markingRead = false
		}
	}

	async markAllRead(): Promise<void> {
		if (!this.#current() || this.markingAll) return
		this.markingAll = true
		this.error = ""
		try {
			await accountBackend.MarkAllChatConversationsRead(this.#accountId)
			if (!this.#current()) return
			this.#clearUnread(null)
		} catch (error) {
			if (this.#current()) {
				this.error = message(error, "Could not mark chats as read. Try again.")
			}
		} finally {
			this.markingAll = false
		}
	}

	#clearUnread(ids: Set<string> | null): void {
		this.conversations = this.conversations.map((item) =>
			!ids || ids.has(item.id) ? { ...item, unreadCount: 0 } : item,
		)
	}

	async create(username: string): Promise<boolean> {
		if (!this.#current() || this.creating) return false
		this.creating = true
		this.createError = ""
		try {
			const conversation = await accountBackend.CreateChatConversation(
				this.#accountId,
				username,
			)
			if (!this.#current()) return false
			this.conversations = [
				conversation,
				...this.conversations.filter((item) => item.id !== conversation.id),
			]
			await this.open(conversation.id)
			return true
		} catch (error) {
			if (this.#current()) {
				this.createError = message(
					error,
					"Could not start this chat. Try again.",
				)
			}
			return false
		} finally {
			this.creating = false
		}
	}

	send(text: string): void {
		const conversationId = this.openId
		if (!this.#current() || !conversationId || !text.trim()) return
		const pending: PendingChatMessage = {
			key: ++this.#nextPendingKey,
			conversationId,
			text,
			createdAtMs: Date.now(),
			failed: false,
			error: "",
		}
		this.pending = [...this.pending, pending]
		void this.#deliver(pending)
	}

	retry(key: number): void {
		const pending = this.pending.find((item) => item.key === key)
		if (!pending?.failed || !this.#current()) return
		this.#updatePending(key, { failed: false, error: "" })
		void this.#deliver(pending)
	}

	discard(key: number): void {
		this.pending = this.pending.filter((item) => item.key !== key)
	}

	async #deliver(pending: PendingChatMessage): Promise<void> {
		try {
			const sent = await accountBackend.SendChatMessage(
				this.#accountId,
				pending.conversationId,
				pending.text,
			)
			if (!this.#current()) return
			this.discard(pending.key)
			if (this.openId === pending.conversationId) {
				this.messages = [...this.messages, sent]
			}
			this.conversations = this.conversations.map((item) =>
				item.id === pending.conversationId
					? { ...item, preview: sent.text, updatedAtMs: sent.createdAtMs }
					: item,
			)
		} catch (error) {
			if (this.#current()) {
				this.#updatePending(pending.key, {
					failed: true,
					error: message(error, "Message not sent."),
				})
			}
		}
	}

	#updatePending(key: number, change: Partial<PendingChatMessage>): void {
		this.pending = this.pending.map((item) =>
			item.key === key ? { ...item, ...change } : item,
		)
	}

	dispose(): void {
		this.#active = false
		this.#listGeneration++
		this.#messagesGeneration++
	}
}

function message(error: unknown, fallback: string): string {
	return error instanceof Error && error.message.trim() ? error.message : fallback
}

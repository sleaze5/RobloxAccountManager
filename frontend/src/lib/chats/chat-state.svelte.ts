import { accountBackend, SocialList } from "../backend/bridge"
import type {
	ChatConversationView,
	ChatMessageView,
	SocialUser,
} from "../backend/bridge"
import type { AccountStore } from "../accounts/account-store.svelte"
import { AccountScope } from "../accounts/account-scope"
import { errorMessage } from "../shared/errors"

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
	renaming = $state(false)
	renameError = $state("")
	friends = $state<SocialUser[]>([])
	friendsLoading = $state(false)
	friendsLoaded = $state(false)
	friendsError = $state("")
	groupMemberLimit = $state(0)
	groupNameMaxLength = $state(0)
	actionError = $state("")
	#listGeneration = 0
	#messagesGeneration = 0
	#deliveredWhileLoading: ChatMessageView[] = []
	#nextPendingKey = 0
	readonly #scope: AccountScope
	readonly selfUserId: number

	constructor(accountId: number, selfUserId: number, store: AccountStore) {
		this.selfUserId = selfUserId
		this.#scope = new AccountScope(accountId, store)
	}

	get openConversation(): ChatConversationView | undefined {
		return this.conversations.find(
			(conversation) => conversation.id === this.openId,
		)
	}

	async refresh(): Promise<void> {
		if (!this.#scope.isCurrent || this.loading) return
		const generation = ++this.#listGeneration
		this.loading = true
		this.error = ""
		this.loadMoreFailed = false
		try {
			const page = await this.#scope.track(
				accountBackend.GetChatConversations(this.#scope.accountId, ""),
			)
			if (!this.#scope.isCurrent || generation !== this.#listGeneration) return
			const conversations = page.conversations ?? [],
				open = this.openConversation
			if (
				open &&
				!conversations.some((conversation) => conversation.id === open.id)
			) {
				conversations.unshift(open)
			}
			this.conversations = conversations
			this.conversationsCursor = page.nextCursor
			this.groupMemberLimit = page.groupMemberLimit
			this.groupNameMaxLength = page.groupNameMaxLength
		} catch (error) {
			if (this.#scope.isCurrent && generation === this.#listGeneration) {
				this.error = errorMessage(error, "Could not load chats. Try again.")
			}
		} finally {
			if (generation === this.#listGeneration) this.loading = false
		}
		if (this.#scope.isCurrent && this.openId) await this.#loadMessages(this.openId)
	}

	async loadMore(): Promise<void> {
		if (
			!this.#scope.isCurrent ||
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
			const page = await this.#scope.track(
				accountBackend.GetChatConversations(
					this.#scope.accountId,
					this.conversationsCursor,
				),
			)
			if (!this.#scope.isCurrent || generation !== this.#listGeneration) return
			const incoming = page.conversations ?? [],
				ids = new Set(incoming.map((conversation) => conversation.id))
			this.conversations = [
				...this.conversations.filter(
					(conversation) => !ids.has(conversation.id),
				),
				...incoming,
			]
			this.conversationsCursor = page.nextCursor
		} catch (error) {
			if (this.#scope.isCurrent && generation === this.#listGeneration) {
				this.error = errorMessage(
					error,
					"Could not load more chats. Try again.",
				)
				this.loadMoreFailed = true
			}
		} finally {
			this.loadingMore = false
		}
	}

	async open(conversationId: string): Promise<void> {
		if (!this.#scope.isCurrent || this.openId === conversationId) return
		this.openId = conversationId
		this.messages = []
		this.messagesCursor = ""
		this.actionError = ""
		await this.#loadMessages(conversationId)
	}

	async #loadMessages(conversationId: string): Promise<void> {
		const generation = ++this.#messagesGeneration
		this.#deliveredWhileLoading = []
		this.messagesLoading = true
		this.olderLoading = false
		this.messagesError = ""
		try {
			const page = await this.#scope.track(
				accountBackend.GetChatMessages(
					this.#scope.accountId,
					conversationId,
					"",
				),
			)
			if (!this.#scope.isCurrent || generation !== this.#messagesGeneration)
				return
			const loaded = page.messages ?? [],
				loadedIds = new Set(loaded.map((message) => message.id))
			this.messages = [
				...loaded,
				...this.#deliveredWhileLoading.filter(
					(message) => !loadedIds.has(message.id),
				),
			]
			this.messagesCursor = page.nextCursor
		} catch (error) {
			if (this.#scope.isCurrent && generation === this.#messagesGeneration) {
				this.messagesError = errorMessage(
					error,
					"Could not load this chat. Try again.",
				)
			}
		} finally {
			if (generation === this.#messagesGeneration) this.messagesLoading = false
		}
	}

	async reloadMessages(): Promise<void> {
		if (this.#scope.isCurrent && this.openId && !this.messagesLoading) {
			await this.#loadMessages(this.openId)
		}
	}

	async loadOlder(): Promise<void> {
		if (
			!this.#scope.isCurrent ||
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
			const page = await this.#scope.track(
				accountBackend.GetChatMessages(
					this.#scope.accountId,
					this.openId,
					this.messagesCursor,
				),
			)
			if (!this.#scope.isCurrent || generation !== this.#messagesGeneration)
				return
			const known = new Set(this.messages.map((message) => message.id))
			this.messages = [
				...(page.messages ?? []).filter((message) => !known.has(message.id)),
				...this.messages,
			]
			this.messagesCursor = page.nextCursor
		} catch (error) {
			if (this.#scope.isCurrent && generation === this.#messagesGeneration) {
				this.messagesError = errorMessage(
					error,
					"Could not load older messages.",
				)
			}
		} finally {
			if (generation === this.#messagesGeneration) this.olderLoading = false
		}
	}

	async markRead(): Promise<void> {
		const conversationId = this.openId
		if (!this.#scope.isCurrent || !conversationId || this.markingRead) return
		this.markingRead = true
		this.actionError = ""
		try {
			await accountBackend.MarkChatConversationRead(
				this.#scope.accountId,
				conversationId,
			)
			if (this.#scope.isCurrent) this.#clearUnread(new Set([conversationId]))
		} catch (error) {
			if (this.#scope.isCurrent) {
				this.actionError = errorMessage(
					error,
					"Could not mark this chat as read.",
				)
			}
		} finally {
			this.markingRead = false
		}
	}

	async markAllRead(): Promise<void> {
		if (!this.#scope.isCurrent || this.markingAll) return
		this.markingAll = true
		this.error = ""
		try {
			await accountBackend.MarkAllChatConversationsRead(this.#scope.accountId)
			if (!this.#scope.isCurrent) return
			this.#clearUnread(null)
		} catch (error) {
			if (this.#scope.isCurrent) {
				this.error = errorMessage(
					error,
					"Could not mark chats as read. Try again.",
				)
			}
		} finally {
			this.markingAll = false
		}
	}

	#clearUnread(ids: Set<string> | null): void {
		this.conversations = this.conversations.map((conversation) =>
			!ids || ids.has(conversation.id)
				? { ...conversation, unreadCount: 0 }
				: conversation,
		)
	}

	async create(userIds: number[], name: string): Promise<boolean> {
		if (!this.#scope.isCurrent || this.creating) return false
		this.creating = true
		this.createError = ""
		try {
			const conversation = await this.#scope.track(
				accountBackend.CreateChatConversation(
					this.#scope.accountId,
					userIds,
					name,
				),
			)
			if (!this.#scope.isCurrent) return false
			this.#install(conversation)
			await this.open(conversation.id)
			return true
		} catch (error) {
			if (this.#scope.isCurrent) {
				this.createError = errorMessage(
					error,
					"Could not start this chat. Try again.",
				)
			}
			return false
		} finally {
			this.creating = false
		}
	}

	async addMembers(userIds: number[]): Promise<boolean> {
		const conversationId = this.openId
		if (!this.#scope.isCurrent || !conversationId || this.creating) return false
		this.creating = true
		this.createError = ""
		try {
			const conversation = await this.#scope.track(
				accountBackend.AddChatMembers(
					this.#scope.accountId,
					conversationId,
					userIds,
				),
			)
			if (!this.#scope.isCurrent) return false
			this.#install(conversation)
			await this.reloadMessages()
			return true
		} catch (error) {
			if (this.#scope.isCurrent) {
				this.createError = errorMessage(
					error,
					"Could not add these members. Try again.",
				)
			}
			return false
		} finally {
			this.creating = false
		}
	}

	async rename(name: string): Promise<boolean> {
		const conversationId = this.openId
		if (!this.#scope.isCurrent || !conversationId || this.renaming) return false
		this.renaming = true
		this.renameError = ""
		try {
			const conversation = await this.#scope.track(
				accountBackend.RenameChatConversation(
					this.#scope.accountId,
					conversationId,
					name,
				),
			)
			if (!this.#scope.isCurrent) return false
			this.#install(conversation)
			return true
		} catch (error) {
			if (this.#scope.isCurrent) {
				this.renameError = errorMessage(
					error,
					"Could not rename this chat. Try again.",
				)
			}
			return false
		} finally {
			this.renaming = false
		}
	}

	// The friends list feeds the people picker. It loads once per page.
	async loadFriends(): Promise<void> {
		if (!this.#scope.isCurrent || this.friendsLoading || this.friendsLoaded) return
		this.friendsLoading = true
		this.friendsError = ""
		try {
			const page = await this.#scope.track(
				accountBackend.GetSocialPage(
					this.#scope.accountId,
					this.selfUserId,
					SocialList.SocialFriends,
					"",
				),
			)
			if (!this.#scope.isCurrent) return
			this.friends = page.people ?? []
			this.friendsLoaded = true
		} catch (error) {
			if (this.#scope.isCurrent) {
				this.friendsError = errorMessage(error, "Could not load friends.")
			}
		} finally {
			this.friendsLoading = false
		}
	}

	async findUser(username: string): Promise<SocialUser | null> {
		return this.#scope.track(accountBackend.FindPlayerByUsername(username))
	}

	#install(conversation: ChatConversationView): void {
		this.conversations = [
			conversation,
			...this.conversations.filter((listed) => listed.id !== conversation.id),
		]
	}

	send(text: string): void {
		const conversationId = this.openId
		if (!this.#scope.isCurrent || !conversationId || !text.trim()) return
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
		const pending = this.pending.find((message) => message.key === key)
		if (!pending?.failed || !this.#scope.isCurrent) return
		this.#updatePending(key, { failed: false, error: "" })
		void this.#deliver(pending)
	}

	discard(key: number): void {
		this.pending = this.pending.filter((message) => message.key !== key)
	}

	async #deliver(pending: PendingChatMessage): Promise<void> {
		try {
			const sent = await accountBackend.SendChatMessage(
				this.#scope.accountId,
				pending.conversationId,
				pending.text,
			)
			if (!this.#scope.isCurrent) return
			this.discard(pending.key)
			if (this.openId === pending.conversationId) {
				this.messages = [...this.messages, sent]
				if (this.messagesLoading) this.#deliveredWhileLoading.push(sent)
			}
			this.conversations = this.conversations.map((conversation) =>
				conversation.id === pending.conversationId
					? {
							...conversation,
							preview: sent.text,
							updatedAtMs: sent.createdAtMs,
						}
					: conversation,
			)
		} catch (error) {
			if (this.#scope.isCurrent) {
				this.#updatePending(pending.key, {
					failed: true,
					error: errorMessage(error, "Message not sent."),
				})
			}
		}
	}

	#updatePending(key: number, change: Partial<PendingChatMessage>): void {
		this.pending = this.pending.map((message) =>
			message.key === key ? { ...message, ...change } : message,
		)
	}

	dispose(): void {
		this.#scope.dispose()
		this.#listGeneration++
		this.#messagesGeneration++
	}
}

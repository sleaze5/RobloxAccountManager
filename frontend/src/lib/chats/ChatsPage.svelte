<script lang="ts">
	import { notifyErrors } from "../notifications/error-notifications.svelte"
	import checksIcon from "@phosphor-icons/core/regular/checks.svg?raw"
	import circleNotchIcon from "@phosphor-icons/core/regular/circle-notch.svg?raw"
	import chatsCircleIcon from "@phosphor-icons/core/regular/chats-circle.svg?raw"
	import notePencilIcon from "@phosphor-icons/core/regular/note-pencil.svg?raw"
	import warningIcon from "@phosphor-icons/core/regular/warning.svg?raw"
	import Icon from "../shared/Icon.svelte"
	import ProfilePageHeader from "../shared/ProfilePageHeader.svelte"
	import RefreshIcon from "../shared/RefreshIcon.svelte"
	import { onMount, untrack } from "svelte"
	import type { AccountStore } from "../accounts/account-store.svelte"
	import SidebarResizer from "../layout/SidebarResizer.svelte"
	import type { WorkspaceState } from "../layout/workspace-state.svelte"
	import ChatList from "./ChatList.svelte"
	import ChatPeopleDialog from "./ChatPeopleDialog.svelte"
	import ChatThread from "./ChatThread.svelte"
	import { ChatState } from "./chat-state.svelte"

	let { store, workspace }: { store: AccountStore; workspace: WorkspaceState } =
		$props()

	const account = untrack(() => store.selectedAccount),
		selfUserId = Number(account.userId),
		chat = untrack(() => new ChatState(account.id, selfUserId, store))

	let peopleDialog = $state<"create" | "add" | null>(null),
		openConversation = $derived(chat.openConversation)

	function openPeopleDialog(mode: "create" | "add"): void {
		chat.createError = ""
		peopleDialog = mode
	}

	async function loadChats(userIdToOpen: number): Promise<void> {
		if (!chat.isFresh) await chat.refresh()
		if (userIdToOpen && !(await chat.create([userIdToOpen], ""))) {
			chat.error = chat.createError
			chat.createError = ""
		}
	}

	onMount(() => {
		void loadChats(workspace.chatUserIdToOpen)
		workspace.chatUserIdToOpen = 0
		return () => chat.dispose()
	})

	notifyErrors("Chats not loaded", () => chat.listError)
	notifyErrors("Chat action not completed", () => chat.error)
</script>

<section class="settings-page" aria-labelledby="chats-title">
	<ProfilePageHeader
		titleId="chats-title"
		title="Chats"
		subtitle={`@${account.username}`}>
		<button
			class="button-ghost"
			type="button"
			aria-label="New chat"
			aria-haspopup="dialog"
			data-tooltip="New chat or group chat"
			onclick={() => openPeopleDialog("create")}>
			<Icon svg={notePencilIcon} size={16} aria-hidden="true" />
		</button>
		<button
			class="button-ghost"
			type="button"
			aria-label="Mark all as read"
			data-tooltip="Mark all as read"
			aria-busy={chat.markingAll}
			disabled={chat.markingAll || chat.loading}
			onclick={() => void chat.markAllRead()}>
			{#if chat.markingAll}
				<Icon
					svg={circleNotchIcon}
					class="spinner"
					size={16}
					aria-hidden="true" />
			{:else}
				<Icon svg={checksIcon} size={16} aria-hidden="true" />
			{/if}
		</button>
		<button
			class="button-ghost"
			type="button"
			aria-label="Refresh chats"
			data-tooltip="Refresh chats"
			aria-busy={chat.loading}
			disabled={chat.loading}
			onclick={() => void chat.refresh()}>
			<RefreshIcon isRefreshing={chat.loading} />
		</button>
	</ProfilePageHeader>

	<div
		class="settings-layout"
		style={`--sidebar-width: ${workspace.accountSettingsSidebarWidth}px;`}>
		<div class="flex settings-sidebar flex-col gap-2 p-2" data-load-more-root>
			<ChatList {chat} />
		</div>
		<SidebarResizer
			bind:width={workspace.accountSettingsSidebarWidth}
			name="chats"
			minimumWidth={192} />

		<section
			class="grid h-full min-h-0 min-w-0 flex-1 grid-rows-1"
			aria-label="Chat">
			{#if openConversation}
				{#key openConversation.id}
					<ChatThread
						{chat}
						conversation={openConversation}
						{selfUserId}
						onAddMembers={() => openPeopleDialog("add")} />
				{/key}
			{:else if chat.listError && chat.conversations.length === 0}
				<div class="empty-state h-full">
					<Icon svg={warningIcon} size={28} aria-hidden="true" />
					<h2>Chats unavailable</h2>
					<button
						class="button"
						type="button"
						disabled={chat.loading}
						onclick={() => void chat.refresh()}>Try again</button>
				</div>
			{:else}
				<div class="empty-state h-full">
					<Icon svg={chatsCircleIcon} size={28} aria-hidden="true" />
					<h2>Select a chat</h2>
					<p>Select a chat, or start a new one.</p>
					<button
						class="button-primary"
						type="button"
						aria-haspopup="dialog"
						onclick={() => openPeopleDialog("create")}>
						<Icon svg={notePencilIcon} size={14} aria-hidden="true" />
						New chat
					</button>
				</div>
			{/if}
		</section>
	</div>
</section>

{#if peopleDialog}
	<ChatPeopleDialog
		{chat}
		mode={peopleDialog}
		existingUserIds={peopleDialog === "add"
			? (openConversation?.participants ?? []).map(
					(participant) => participant.userId,
				)
			: []}
		onClose={() => (peopleDialog = null)} />
{/if}

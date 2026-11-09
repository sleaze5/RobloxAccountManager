<script lang="ts">
	import caretLeftIcon from "@phosphor-icons/core/regular/caret-left.svg?raw"
	import checksIcon from "@phosphor-icons/core/regular/checks.svg?raw"
	import circleNotchIcon from "@phosphor-icons/core/regular/circle-notch.svg?raw"
	import chatsCircleIcon from "@phosphor-icons/core/regular/chats-circle.svg?raw"
	import notePencilIcon from "@phosphor-icons/core/regular/note-pencil.svg?raw"
	import Icon from "../shared/Icon.svelte"
	import RefreshIcon from "../shared/RefreshIcon.svelte"
	import { onMount, untrack } from "svelte"
	import type { AccountStore } from "../accounts/account-store.svelte"
	import SidebarResizer from "../layout/SidebarResizer.svelte"
	import type { WorkspaceState } from "../layout/workspace-state.svelte"
	import ChatList from "./ChatList.svelte"
	import ChatPeopleDialog from "./ChatPeopleDialog.svelte"
	import ChatThread from "./ChatThread.svelte"
	import { ChatState } from "./chat-state.svelte"

	let {
		store,
		workspace,
		onBack,
	}: { store: AccountStore; workspace: WorkspaceState; onBack: () => void } = $props()

	const account = untrack(() => store.selectedAccount),
		selfUserId = Number(account.userId),
		chat = untrack(() => new ChatState(account.id, selfUserId, store))

	let heading = $state<HTMLHeadingElement | undefined>(undefined),
		peopleDialog = $state<"create" | "add" | null>(null),
		openConversation = $derived(chat.openConversation)

	function openPeopleDialog(mode: "create" | "add"): void {
		chat.createError = ""
		peopleDialog = mode
	}

	onMount(() => {
		heading?.closest("[data-workspace-scroll]")?.scrollTo({ top: 0 })
		heading?.focus({ preventScroll: true })
		void chat.refresh()
		return () => chat.dispose()
	})
</script>

<section class="settings-page" aria-labelledby="chats-title">
	<header class="settings-header">
		<button
			class="button-ghost"
			type="button"
			aria-label="Back to profile"
			data-tooltip="Back to profile"
			onclick={onBack}>
			<Icon svg={caretLeftIcon} data-nudge="left" size={16} aria-hidden="true" />
		</button>
		<h1 id="chats-title" tabindex="-1" bind:this={heading}>Chats</h1>
		<span class="mr-auto min-w-0 text-xs wrap-anywhere text-fg-muted"
			>@{account.username}</span>
		<button
			class="button-ghost"
			type="button"
			aria-label="New chat"
			aria-haspopup="dialog"
			data-tooltip="New chat or group"
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
	</header>

	<div
		class="settings-layout"
		style={`--sidebar-width: ${workspace.accountSettingsSidebarWidth}px;`}>
		<aside class="flex settings-sidebar flex-col gap-2 p-2" data-chat-sidebar>
			{#if chat.error}
				<p class="notice-error" role="alert">
					{chat.error}
				</p>
			{/if}
			<ChatList {chat} />
		</aside>
		<SidebarResizer
			bind:width={workspace.accountSettingsSidebarWidth}
			name="chats"
			minimumWidth={192} />

		<main class="grid h-full min-h-0 min-w-0 flex-1 grid-rows-1" aria-label="Chat">
			{#if openConversation}
				{#key openConversation.id}
					<ChatThread
						{chat}
						conversation={openConversation}
						{selfUserId}
						onAddMembers={() => openPeopleDialog("add")} />
				{/key}
			{:else}
				<div class="empty-state h-full">
					<Icon svg={chatsCircleIcon} size={28} aria-hidden="true" />
					<h2>Select a chat</h2>
					<p>Choose a chat to read it, or start a new one with friends.</p>
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
		</main>
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

<script lang="ts">
	import caretLeftIcon from "@phosphor-icons/core/regular/caret-left.svg?raw"
	import checksIcon from "@phosphor-icons/core/regular/checks.svg?raw"
	import circleNotchIcon from "@phosphor-icons/core/regular/circle-notch.svg?raw"
	import notePencilIcon from "@phosphor-icons/core/regular/note-pencil.svg?raw"
	import Icon from "../shared/Icon.svelte"
	import RefreshIcon from "../shared/RefreshIcon.svelte"
	import { onMount, tick, untrack } from "svelte"
	import type { AccountStore } from "../accounts/account-store.svelte"
	import SidebarResizer from "../layout/SidebarResizer.svelte"
	import type { WorkspaceState } from "../layout/workspace-state.svelte"
	import ChatList from "./ChatList.svelte"
	import ChatThread from "./ChatThread.svelte"
	import { ChatState } from "./chat-state.svelte"

	let {
		store,
		workspace,
		onBack,
	}: { store: AccountStore; workspace: WorkspaceState; onBack: () => void } = $props()

	const account = untrack(() => store.selectedAccount),
		chat = untrack(() => new ChatState(account.id, store))

	let heading = $state<HTMLHeadingElement | undefined>(undefined),
		usernameInput = $state<HTMLInputElement | undefined>(undefined),
		creatingOpen = $state(false),
		username = $state(""),
		openConversation = $derived(chat.openConversation)

	async function toggleNewChat(): Promise<void> {
		creatingOpen = !creatingOpen
		chat.createError = ""
		if (!creatingOpen) return
		await tick()
		usernameInput?.focus()
	}

	async function startChat(): Promise<void> {
		if (await chat.create(username)) {
			username = ""
			creatingOpen = false
		}
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
			aria-expanded={creatingOpen}
			data-tooltip="New chat"
			onclick={() => void toggleNewChat()}>
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
			{#if creatingOpen}
				<form
					novalidate
					class="flex animate-enter flex-wrap gap-2"
					onsubmit={(event) => {
						event.preventDefault()
						void startChat()
					}}>
					<div class="search-field">
						<input
							bind:this={usernameInput}
							bind:value={username}
							aria-label="Username"
							aria-invalid={chat.createError ? "true" : undefined}
							placeholder="Username"
							autocomplete="off"
							spellcheck="false"
							disabled={chat.creating}
							onkeydown={(event) => {
								if (event.key === "Escape") void toggleNewChat()
							}} />
					</div>
					<button
						type="submit"
						class="button-primary h-8"
						aria-busy={chat.creating}
						disabled={chat.creating || !username.trim()}>
						{#if chat.creating}
							<Icon
								svg={circleNotchIcon}
								class="spinner"
								size={16}
								aria-hidden="true" />
						{/if}
						Start
					</button>
					{#if chat.createError}
						<p class="w-full text-xs text-danger-muted" role="alert">
							{chat.createError}
						</p>
					{/if}
				</form>
			{/if}
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
						selfUserId={Number(account.userId)} />
				{/key}
			{:else}
				<p class="m-auto text-xs text-fg-faint">Select a chat to read it.</p>
			{/if}
		</main>
	</div>
</section>

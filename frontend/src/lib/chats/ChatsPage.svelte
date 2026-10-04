<script lang="ts">
	import ChecksIcon from "phosphor-svelte/lib/ChecksIcon"
	import CaretLeftIcon from "phosphor-svelte/lib/CaretLeftIcon"
	import CircleNotchIcon from "phosphor-svelte/lib/CircleNotchIcon"
	import ArrowsClockwiseIcon from "phosphor-svelte/lib/ArrowsClockwiseIcon"
	import NotePencilIcon from "phosphor-svelte/lib/NotePencilIcon"
	import { onMount, tick, untrack } from "svelte"
	import type { AccountStore } from "../accounts/account-store.svelte"
	import ChatList from "./ChatList.svelte"
	import ChatThread from "./ChatThread.svelte"
	import { ChatState } from "./chat-state.svelte"

	let { store, onBack }: { store: AccountStore; onBack: () => void } = $props()

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
		heading?.closest(".workspace-scroll")?.scrollTo({ top: 0 })
		heading?.focus({ preventScroll: true })
		void chat.refresh()
		return () => chat.dispose()
	})
</script>

<section class="settings-page" aria-labelledby="chats-title">
	<header class="settings-header">
		<button
			type="button"
			aria-label="Back to profile"
			data-tooltip="Back to profile"
			onclick={onBack}>
			<CaretLeftIcon size={18} aria-hidden="true" />
		</button>
		<h1 id="chats-title" tabindex="-1" bind:this={heading}>Chats</h1>
		<span class="account-settings-identity">@{account.username}</span>
		<button
			type="button"
			aria-label="New chat"
			aria-expanded={creatingOpen}
			data-tooltip="New chat"
			onclick={() => void toggleNewChat()}>
			<NotePencilIcon size={17} aria-hidden="true" />
		</button>
		<button
			type="button"
			aria-label="Mark all as read"
			data-tooltip="Mark all as read"
			aria-busy={chat.markingAll}
			disabled={chat.markingAll || chat.loading}
			onclick={() => void chat.markAllRead()}>
			{#if chat.markingAll}
				<CircleNotchIcon class="spinner" size={17} aria-hidden="true" />
			{:else}
				<ChecksIcon size={17} aria-hidden="true" />
			{/if}
		</button>
		<button
			type="button"
			aria-label="Refresh chats"
			data-tooltip="Refresh chats"
			aria-busy={chat.loading}
			disabled={chat.loading}
			onclick={() => void chat.refresh()}>
			{#if chat.loading}
				<CircleNotchIcon class="spinner" size={17} aria-hidden="true" />
			{:else}
				<ArrowsClockwiseIcon size={17} aria-hidden="true" />
			{/if}
		</button>
	</header>

	<div class="settings-layout account-settings-layout">
		<aside class="settings-sidebar chat-sidebar">
			{#if creatingOpen}
				<form
					novalidate
					class="chat-new"
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
						class="primary-action"
						aria-busy={chat.creating}
						disabled={chat.creating || !username.trim()}>
						{#if chat.creating}
							<CircleNotchIcon
								class="spinner"
								size={15}
								aria-hidden="true" />
						{/if}
						Start
					</button>
					{#if chat.createError}
						<p class="chat-new-error" role="alert">{chat.createError}</p>
					{/if}
				</form>
			{/if}
			{#if chat.error}
				<p class="settings-inline-error chat-list-error" role="alert">
					{chat.error}
				</p>
			{/if}
			<ChatList {chat} />
		</aside>

		<main class="chat-main" aria-label="Chat">
			{#if openConversation}
				{#key openConversation.id}
					<ChatThread
						{chat}
						conversation={openConversation}
						selfUserId={Number(account.userId)} />
				{/key}
			{:else}
				<p class="settings-content-empty">Select a chat to read it.</p>
			{/if}
		</main>
	</div>
</section>

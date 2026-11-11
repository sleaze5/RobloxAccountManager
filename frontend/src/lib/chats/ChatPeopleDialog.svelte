<script lang="ts">
	import circleNotchIcon from "@phosphor-icons/core/regular/circle-notch.svg?raw"
	import magnifyingGlassIcon from "@phosphor-icons/core/regular/magnifying-glass.svg?raw"
	import userCirclePlusIcon from "@phosphor-icons/core/regular/user-circle-plus.svg?raw"
	import usersIcon from "@phosphor-icons/core/regular/users.svg?raw"
	import xIcon from "@phosphor-icons/core/regular/x.svg?raw"
	import { onDestroy, onMount } from "svelte"
	import Icon from "../shared/Icon.svelte"
	import CheckboxMark from "../shared/CheckboxMark.svelte"
	import Dialog from "../shared/Dialog.svelte"
	import ProfileImage from "../shared/ProfileImage.svelte"
	import type { SocialUser } from "../backend/bridge"
	import { presenceLabel } from "../accounts/account-model"
	import SocialAvatar from "../social/SocialAvatar.svelte"
	import { errorMessage } from "../shared/errors"
	import { matchesSearch, searchTerms } from "../shared/search"
	import type { ChatState } from "./chat-state.svelte"

	let {
		chat,
		mode,
		existingUserIds = [],
		onClose,
	}: {
		chat: ChatState
		mode: "create" | "add"
		existingUserIds?: number[]
		onClose: () => void
	} = $props()

	// The lookup waits for a pause in typing, so that each keystroke does not
	// send a Roblox request.
	const lookupDelayMs = 700,
		minimumLookupLength = 3

	let query = $state(""),
		groupName = $state(""),
		selected = $state<SocialUser[]>([]),
		lookupStatus = $state<
			"idle" | "waiting" | "loading" | "found" | "missing" | "failed"
		>("idle"),
		lookupPerson = $state<SocialUser | null>(null),
		lookupError = $state(""),
		lookupTimer = 0,
		lookupRequest = 0

	const memberLimit = $derived(Math.max(chat.groupMemberLimit, 2)),
		selectionLimit = $derived(
			mode === "create"
				? memberLimit - 1
				: Math.max(0, memberLimit - existingUserIds.length),
		),
		selectedIds = $derived(selected.map((person) => person.userId)),
		term = $derived(query.trim().replace(/^@/, "").toLowerCase()),
		candidates = $derived(
			chat.friends.filter((friend) => !existingUserIds.includes(friend.userId)),
		),
		matches = $derived.by(() => {
			const terms = searchTerms(term)
			return candidates.filter((friend) =>
				matchesSearch(
					terms,
					`${friend.username} ${friend.displayName} ${friend.nickname}`,
				),
			)
		}),
		exactFriend = $derived(
			candidates.some((friend) => friend.username.toLowerCase() === term),
		),
		isGroup = $derived(mode === "add" || selected.length > 1),
		full = $derived(selected.length >= selectionLimit),
		submitting = $derived(chat.creating),
		title = $derived(mode === "create" ? "New chat" : "Add members"),
		submitLabel = $derived.by(() => {
			if (mode === "add") {
				if (submitting) return "Adding…"
				return selected.length > 1
					? `Add ${selected.length} members`
					: "Add member"
			}
			if (submitting) return isGroup ? "Creating…" : "Starting…"
			return isGroup ? "Create group" : "Start chat"
		})

	$effect(() => {
		const text = term,
			skip = text.length < minimumLookupLength || exactFriend
		window.clearTimeout(lookupTimer)
		const request = ++lookupRequest
		lookupPerson = null
		lookupError = ""
		lookupStatus = skip ? "idle" : "waiting"
		if (skip) return
		lookupTimer = window.setTimeout(() => void lookup(text, request), lookupDelayMs)
	})

	async function lookup(username: string, request: number): Promise<void> {
		lookupStatus = "loading"
		try {
			const person = await chat.findUser(username)
			if (request !== lookupRequest) return
			if (person && existingUserIds.includes(person.userId)) {
				lookupStatus = "missing"
				lookupError = "This person is already in the chat."
				return
			}
			lookupPerson = person
			lookupStatus = person ? "found" : "missing"
		} catch (error) {
			if (request !== lookupRequest) return
			lookupStatus = "failed"
			lookupError = errorMessage(error, "Could not look up this username.")
		}
	}

	function toggle(person: SocialUser): void {
		if (selectedIds.includes(person.userId)) {
			selected = selected.filter((chosen) => chosen.userId !== person.userId)
			return
		}
		if (full) return
		selected = [...selected, person]
		chat.createError = ""
	}

	function addFromSearch(): void {
		const person = lookupPerson ?? (matches.length === 1 ? matches[0] : null)
		if (!person || selectedIds.includes(person.userId) || full) return
		toggle(person)
		query = ""
	}

	async function submit(): Promise<void> {
		if (selected.length === 0 || submitting) return
		const done =
			mode === "create"
				? await chat.create(selectedIds, isGroup ? groupName : "")
				: await chat.addMembers(selectedIds)
		if (done) onClose()
	}

	function close(): void {
		if (submitting) return
		chat.createError = ""
		onClose()
	}

	onMount(() => void chat.loadFriends())
	onDestroy(() => {
		window.clearTimeout(lookupTimer)
		lookupRequest++
	})
</script>

<svelte:window
	onkeydown={(event) => {
		if (event.key === "Escape") {
			event.preventDefault()
			close()
		}
	}} />

{#snippet personRow(person: SocialUser, note: string)}
	{@const checked = selectedIds.includes(person.userId)}
	<label
		class={[
			"group/row list-row flex h-12 w-full cursor-pointer items-center gap-2 border-b border-line-subtle px-3 last:border-b-0",
			!checked && full && "cursor-not-allowed opacity-45",
		]}
		data-selected={checked || undefined}>
		<span class="checkbox shrink-0">
			<input
				type="checkbox"
				aria-label={`${checked ? "Remove" : "Add"} ${person.displayName} (@${person.username})`}
				{checked}
				disabled={submitting || (!checked && full)}
				onchange={() => toggle(person)} />
			<CheckboxMark />
		</span>
		<SocialAvatar
			imageUrl={person.imageUrl}
			presence={person.presence ?? undefined}
			className="size-8" />
		<span class="flex min-w-0 flex-1 flex-col leading-snug">
			<span class="truncate text-sm font-medium text-fg"
				>{person.nickname || person.displayName}</span>
			<span
				class="truncate text-2xs text-fg-faint group-hover/row:text-fg-muted group-data-selected/row:text-fg-muted"
				>@{person.username}</span>
		</span>
		<span class="shrink-0 text-2xs text-fg-faint">{note}</span>
	</label>
{/snippet}

<Dialog
	class="flex max-h-full max-w-md flex-col text-left"
	aria-labelledby="chat-people-title"
	aria-describedby="chat-people-description"
	onBackdropClick={close}>
	<div class="dialog-heading">
		<div>
			<h2 id="chat-people-title">{title}</h2>
			<p id="chat-people-description">
				{mode === "create"
					? `Choose one friend for a direct chat, or up to ${selectionLimit} for a group.`
					: selectionLimit > 0
						? `This group has room for ${selectionLimit} more ${selectionLimit === 1 ? "person" : "people"}.`
						: "This group is full."}
			</p>
		</div>
		<button
			type="button"
			aria-label={`Close ${title.toLowerCase()} dialog`}
			disabled={submitting}
			onclick={close}>
			<Icon svg={xIcon} size={14} aria-hidden="true" />
		</button>
	</div>

	<form
		novalidate
		class="mt-3 flex min-h-0 flex-col gap-3"
		onsubmit={(event) => {
			event.preventDefault()
			void submit()
		}}>
		<div class="flex min-h-7 flex-wrap items-center gap-1" aria-live="polite">
			{#each selected as person (person.userId)}
				<span
					class="inline-flex h-7 animate-enter items-center gap-1.5 rounded-md border border-line-strong bg-raised pr-0.5 pl-1 text-xs font-medium text-fg">
					<span
						class="relative grid size-5 shrink-0 place-items-center overflow-hidden rounded-sm border border-line bg-raised"
						aria-hidden="true"><ProfileImage url={person.imageUrl} /></span>
					<span class="max-w-28 truncate">{person.displayName}</span>
					<button
						class="button-ghost-sm"
						type="button"
						aria-label={`Remove ${person.displayName}`}
						disabled={submitting}
						onclick={() => toggle(person)}>
						<Icon svg={xIcon} size={12} aria-hidden="true" />
					</button>
				</span>
			{:else}
				<span class="text-xs text-fg-faint">Nobody selected yet.</span>
			{/each}
			<span class="ml-auto font-mono text-2xs text-fg-faint"
				>{selected.length}/{selectionLimit}</span>
		</div>

		<div class="search-field">
			<Icon svg={magnifyingGlassIcon} size={16} aria-hidden="true" /><input
				bind:value={query}
				aria-label="Search friends or enter a username"
				placeholder="Search friends or enter a username"
				autocomplete="off"
				spellcheck="false"
				maxlength={50}
				disabled={submitting}
				onkeydown={(event) => {
					if (event.key === "Enter") {
						event.preventDefault()
						addFromSearch()
					}
				}} />{#if query}<button
					type="button"
					aria-label="Clear search"
					onclick={() => (query = "")}
					><Icon svg={xIcon} size={14} aria-hidden="true" /></button
				>{/if}
		</div>

		<div
			class="overflow-hidden rounded-md border border-line bg-fill"
			aria-busy={chat.friendsLoading || lookupStatus === "loading"}>
			<div class="max-h-64 min-h-24 overflow-y-auto">
				{#if lookupStatus === "waiting" || lookupStatus === "loading"}
					<p
						class="flex h-12 items-center gap-2 border-b border-line-subtle px-3 text-xs text-fg-faint">
						<Icon
							svg={circleNotchIcon}
							class="spinner"
							size={16}
							aria-hidden="true" />
						Looking up @{term}…
					</p>
				{:else if lookupStatus === "found" && lookupPerson}
					<div class="animate-enter" role="group" aria-label="Roblox user">
						{@render personRow(lookupPerson, "Roblox user")}
					</div>
				{:else if lookupStatus === "missing" || lookupStatus === "failed"}
					<p
						class="flex h-12 items-center gap-2 border-b border-line-subtle px-3 text-xs text-fg-muted"
						role={lookupStatus === "failed" ? "alert" : undefined}>
						<Icon svg={userCirclePlusIcon} size={16} aria-hidden="true" />
						{lookupError || `No Roblox user is named @${term}.`}
					</p>
				{/if}
				{#if chat.friendsLoading && chat.friends.length === 0}
					<p
						class="flex h-12 items-center gap-2 px-3 text-xs text-fg-faint"
						role="status">
						<Icon
							svg={circleNotchIcon}
							class="spinner"
							size={16}
							aria-hidden="true" />
						Loading friends…
					</p>
				{:else if chat.friendsError}
					<p
						class="flex h-12 items-center justify-between gap-2 px-3 text-xs text-danger-muted"
						role="alert">
						{chat.friendsError}
						<button
							class="button"
							type="button"
							onclick={() => void chat.loadFriends()}>Try again</button>
					</p>
				{:else if matches.length === 0 && lookupStatus === "idle"}
					<div
						class="flex h-24 flex-col items-center justify-center gap-1 px-3 text-center text-xs text-fg-faint">
						<Icon svg={usersIcon} size={20} aria-hidden="true" />
						{term
							? "No friends match. Type a full username to look it up."
							: "No friends to show. Type a username to find someone."}
					</div>
				{:else if matches.length > 0}
					<div role="group" aria-label="Friends">
						{#each matches as friend (friend.userId)}
							{@render personRow(
								friend,
								friend.presence ? presenceLabel(friend.presence) : "",
							)}
						{/each}
					</div>
				{/if}
			</div>
		</div>

		{#if mode === "create" && isGroup}
			<label class="dialog-field mt-0 animate-enter">
				<span>Group name (optional)</span>
				<input
					class="field"
					bind:value={groupName}
					maxlength={chat.groupNameMaxLength || undefined}
					placeholder="Named after its members"
					autocomplete="off"
					disabled={submitting} />
			</label>
		{/if}

		{#if chat.createError}
			<div class="notice-error" role="alert">{chat.createError}</div>
		{/if}

		<div class="dialog-actions mt-0">
			<button class="button" type="button" disabled={submitting} onclick={close}
				>Cancel</button>
			<button
				class="button-primary"
				type="submit"
				aria-busy={submitting}
				disabled={selected.length === 0 || submitting}>
				{#if submitting}<Icon
						svg={circleNotchIcon}
						class="spinner"
						size={14}
						aria-hidden="true" />{/if}
				{submitLabel}
			</button>
		</div>
	</form>
</Dialog>

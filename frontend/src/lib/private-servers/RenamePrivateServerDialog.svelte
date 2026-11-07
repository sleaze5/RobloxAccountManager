<script lang="ts">
	import { tick, untrack } from "svelte"
	import checkCircleIcon from "@phosphor-icons/core/regular/check-circle.svg?raw"
	import xIcon from "@phosphor-icons/core/regular/x.svg?raw"
	import Icon from "../shared/Icon.svelte"
	import { accountBackend, TextModeration } from "../backend/bridge"
	import type { TextFilter } from "../backend/bridge"
	import type { NotificationCenter } from "../notifications/notification-center.svelte"
	import { replayValidationNudge } from "../shared/presence"
	import { errorMessage } from "../shared/errors"
	import Dialog from "../shared/Dialog.svelte"
	import type { PrivateServerState } from "./private-server-state.svelte"

	let {
		server,
		username,
		notifications,
		onClose,
	}: {
		server: PrivateServerState
		username: string
		notifications: NotificationCenter
		onClose: () => void
	} = $props()

	const id = $props.id(),
		maximumLength = 50,
		current = untrack(() => server.settings?.name ?? "")
	let value = $state(current),
		validating = $state<"validate" | "submit" | null>(null),
		validated = $state<{ text: string; filter: TextFilter | null } | null>(null),
		error = $state(""),
		card = $state<HTMLDivElement>(),
		field = $state<HTMLInputElement>()
	const text = $derived(value.trim()),
		changed = $derived(text !== "" && text !== current),
		result = $derived(validated?.text === text ? validated : null),
		clean = $derived(
			result?.filter?.moderation === TextModeration.TextModerationNone,
		),
		tagged = $derived(
			!!result?.filter &&
				result.filter.moderation !== TextModeration.TextModerationNone,
		),
		preview = $derived(tagged ? result?.filter?.filteredText : undefined),
		renaming = $derived(server.pending === "rename"),
		busy = $derived(validating !== null || renaming)

	function close(): void {
		if (!busy) onClose()
	}

	async function validate(mode: "validate" | "submit"): Promise<boolean> {
		const validatedText = text
		validating = mode
		error = ""
		try {
			const filter = await accountBackend.ValidatePrivateServerName(
				server.accountId,
				validatedText,
			)
			validated = { text: validatedText, filter }
			return filter.moderation === TextModeration.TextModerationNone
		} catch (cause) {
			validated = { text: validatedText, filter: null }
			error = errorMessage(cause, "Could not validate the name. Try again.")
			return false
		} finally {
			validating = null
		}
	}

	async function edit(): Promise<void> {
		validated = null
		await tick()
		field?.focus()
	}

	async function rename(): Promise<void> {
		error = ""
		if (await server.rename(text)) {
			notifications.show({
				id: `private-server-rename-${server.serverId}`,
				title: "Private server renamed",
				tone: "success",
				message: `@${username}`,
			})
			onClose()
			return
		}
		error = server.errorFor("rename")
		replayValidationNudge(card)
	}

	async function submit(): Promise<void> {
		if (!changed || busy) {
			replayValidationNudge(card)
		} else if (result || (await validate("submit"))) {
			await rename()
		}
	}
</script>

<svelte:window
	onkeydown={(event) => {
		if (event.key === "Escape") close()
	}} />

<Dialog class="max-w-sm text-left" aria-labelledby={`${id}-title`} bind:card>
	<form
		novalidate
		class="contents"
		onsubmit={(event) => {
			event.preventDefault()
			void submit()
		}}>
		<div class="dialog-heading">
			<div class="min-w-0">
				<h2 id={`${id}-title`}>Rename private server</h2>
				<p>@{username}</p>
			</div>
			<button
				type="button"
				aria-label="Close rename private server"
				disabled={busy}
				onclick={close}>
				<Icon svg={xIcon} size={14} aria-hidden="true" />
			</button>
		</div>
		<div class="dialog-field mt-0">
			<label for={`${id}-name`}>Name</label>
			{#if preview === undefined}
				<input
					id={`${id}-name`}
					class="field"
					type="text"
					autocomplete="off"
					maxlength={maximumLength}
					aria-describedby={`${id}-count`}
					disabled={busy}
					bind:this={field}
					bind:value />
			{:else}
				<input
					id={`${id}-name`}
					class="field"
					type="text"
					aria-describedby={`${id}-count`}
					value={preview}
					readonly
					disabled={busy} />
			{/if}
			<small
				id={`${id}-count`}
				class="flex justify-end font-mono text-2xs text-fg-muted tabular-nums"
				>{Array.from(value).length}/{maximumLength}</small>
		</div>
		<div aria-live="polite">
			{#if clean}
				<span class="sr-only">Roblox's filter won't tag this name.</span>
			{:else if result?.filter?.moderation === TextModeration.TextModerationPartial}
				<span class="sr-only">Roblox will tag part of this name.</span>
			{:else if result?.filter?.moderation === TextModeration.TextModerationFull}
				<span class="sr-only">Roblox will tag all of this name.</span>
			{/if}
		</div>
		{#if error}
			<div class="mt-3 notice-error" role="alert">{error}</div>
		{/if}
		<div class="dialog-actions">
			<button class="button" type="button" disabled={busy} onclick={close}
				>Cancel</button>
			<button
				class="button"
				type="button"
				aria-busy={validating === "validate"}
				disabled={busy ||
					(preview === undefined && (text === "" || result !== null))}
				onclick={() =>
					void (preview === undefined ? validate("validate") : edit())}>
				{#if validating === "validate"}
					Validating…
				{:else if preview !== undefined}
					Edit
				{:else}
					{#if clean}<Icon
							svg={checkCircleIcon}
							class="text-success"
							size={14}
							aria-hidden="true" />{/if}
					Validate
				{/if}
			</button>
			<button
				class={tagged ? "button-warning" : "button-primary"}
				type="submit"
				aria-busy={validating === "submit" || renaming}
				disabled={!changed || busy}>
				{#if validating === "submit"}
					Validating…
				{:else if renaming}
					Renaming…
				{:else if result && !result.filter}
					Rename without validating
				{:else if tagged}
					Rename anyway
				{:else}
					Rename server
				{/if}
			</button>
		</div>
	</form>
</Dialog>

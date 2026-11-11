<script lang="ts">
	import { tick, untrack } from "svelte"
	import xIcon from "@phosphor-icons/core/regular/x.svg?raw"
	import Icon from "../shared/Icon.svelte"
	import { accountBackend } from "../backend/bridge"
	import type { NotificationCenter } from "../notifications/notification-center.svelte"
	import { replayValidationNudge } from "../shared/presence"
	import { errorMessage } from "../shared/errors"
	import { TextFilterCheck } from "../shared/text-filter-check.svelte"
	import TextFilterStatus from "../shared/TextFilterStatus.svelte"
	import TextFilterValidateButton from "../shared/TextFilterValidateButton.svelte"
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
		error = $state(""),
		card = $state<HTMLDivElement>(),
		field = $state<HTMLInputElement>()
	const text = $derived(value.trim()),
		changed = $derived(text !== "" && text !== current),
		renaming = $derived(server.pending === "rename"),
		busy = $derived(validating !== null || renaming),
		check = new TextFilterCheck(() => text)

	function close(): void {
		if (!busy) onClose()
	}

	async function validate(mode: "validate" | "submit"): Promise<boolean> {
		const validatedText = text
		validating = mode
		error = ""
		try {
			return await check.run(validatedText, () =>
				accountBackend.ValidatePrivateServerName(
					server.accountId,
					validatedText,
				),
			)
		} catch (cause) {
			error = errorMessage(cause, "Could not validate the name. Try again.")
			return false
		} finally {
			validating = null
		}
	}

	async function edit(): Promise<void> {
		check.clear()
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
		} else if (check.current || (await validate("submit"))) {
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
			{#if check.filteredPreview === undefined}
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
					value={check.filteredPreview}
					readonly
					disabled={busy} />
			{/if}
			<small
				id={`${id}-count`}
				class="flex justify-end font-mono text-2xs text-fg-muted tabular-nums"
				>{Array.from(value).length}/{maximumLength}</small>
		</div>
		<div aria-live="polite">
			<TextFilterStatus {check} noun="name" />
		</div>
		{#if error}
			<div class="mt-3 notice-error" role="alert">{error}</div>
		{/if}
		<div class="dialog-actions">
			<button class="button" type="button" disabled={busy} onclick={close}
				>Cancel</button>
			<TextFilterValidateButton
				{check}
				{text}
				isValidating={validating === "validate"}
				disabled={busy}
				onValidate={() => void validate("validate")}
				onEdit={() => void edit()} />
			<button
				class={check.isTagged ? "button-warning" : "button-primary"}
				type="submit"
				aria-busy={validating === "submit" || renaming}
				disabled={!changed || busy}>
				{#if validating === "submit"}
					Validating…
				{:else if renaming}
					Renaming…
				{:else if check.current && !check.current.filter}
					Rename without validating
				{:else if check.isTagged}
					Rename anyway
				{:else}
					Rename server
				{/if}
			</button>
		</div>
	</form>
</Dialog>

<script lang="ts">
	import { untrack } from "svelte"
	import xIcon from "@phosphor-icons/core/regular/x.svg?raw"
	import Icon from "../shared/Icon.svelte"
	import { accountBackend, TextModeration } from "../backend/bridge"
	import type { TextFilter } from "../backend/bridge"
	import {
		modalBackdropIn,
		modalBackdropOut,
		modalCardIn,
		modalCardOut,
		modalFocus,
		replayValidationNudge,
	} from "../shared/presence"
	import { errorMessage } from "../shared/errors"
	import type { PrivateServerState } from "./private-server-state.svelte"

	let {
		server,
		username,
		onClose,
	}: {
		server: PrivateServerState
		username: string
		onClose: () => void
	} = $props()

	const id = $props.id(),
		maximumLength = 50,
		current = untrack(() => server.settings?.name ?? "")
	let value = $state(current),
		checking = $state(false),
		checked = $state<{ text: string; filter: TextFilter | null } | null>(null),
		error = $state(""),
		card = $state<HTMLDivElement>()
	const text = $derived(value.trim()),
		changed = $derived(text !== "" && text !== current),
		result = $derived(checked?.text === text ? checked : null),
		censored = $derived(
			!!result?.filter &&
				result.filter.moderation !== TextModeration.TextModerationNone,
		),
		renaming = $derived(server.pending === "rename"),
		busy = $derived(checking || renaming)

	function close(): void {
		if (!busy) onClose()
	}

	async function check(): Promise<boolean> {
		const checkedText = text
		checking = true
		error = ""
		try {
			const filter = await accountBackend.ValidatePrivateServerName(
				server.accountId,
				checkedText,
			)
			checked = { text: checkedText, filter }
			return filter.moderation === TextModeration.TextModerationNone
		} catch (cause) {
			checked = { text: checkedText, filter: null }
			error = errorMessage(
				cause,
				"Roblox's filter could not check the name. Try again, or rename without checking.",
			)
			return false
		} finally {
			checking = false
		}
	}

	async function submit(): Promise<void> {
		if (!changed || busy) {
			replayValidationNudge(card)
			return
		}
		if (!result && !(await check())) {
			replayValidationNudge(card)
			return
		}
		if (await server.rename(text)) {
			onClose()
			return
		}
		error = server.errorFor("rename")
		replayValidationNudge(card)
	}
</script>

<svelte:window
	onkeydown={(event) => {
		if (event.key === "Escape") close()
	}} />

<div class="dialog-backdrop" in:modalBackdropIn out:modalBackdropOut>
	<div
		class="dialog-card max-w-sm text-left"
		role={censored ? "alertdialog" : "dialog"}
		aria-modal="true"
		aria-labelledby={`${id}-title`}
		aria-describedby={`${id}-description`}
		bind:this={card}
		use:modalFocus
		in:modalCardIn
		out:modalCardOut>
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
					<p id={`${id}-description`}>
						Saved to Roblox for @{username}. Everyone who can join sees the
						new name.
					</p>
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
				<input
					id={`${id}-name`}
					class="field"
					type="text"
					autocomplete="off"
					maxlength={maximumLength}
					aria-describedby={`${id}-count`}
					disabled={busy}
					bind:value />
				<small
					id={`${id}-count`}
					class="flex justify-end font-mono text-2xs text-fg-muted tabular-nums"
					>{Array.from(value).length}/{maximumLength}</small>
			</div>
			<div aria-live="polite">
				{#if censored && result?.filter}
					<div class="mt-3 flex flex-col gap-1.5 notice-warning">
						<span
							>{result.filter.moderation ===
							TextModeration.TextModerationFull
								? "Roblox will censor this name."
								: "Roblox will censor part of this name."} Players will see:</span>
						<span class="font-mono text-2xs wrap-anywhere text-fg"
							>{result.filter.filteredText}</span>
					</div>
				{/if}
			</div>
			{#if error}
				<div class="mt-3 notice-error" role="alert">{error}</div>
			{/if}
			<div class="dialog-actions">
				<button class="btn" type="button" disabled={busy} onclick={close}
					>Cancel</button>
				<button
					class={censored ? "btn-warning" : "btn-primary"}
					type="submit"
					aria-busy={busy}
					disabled={!changed || busy}>
					{#if checking}
						Checking…
					{:else if renaming}
						Renaming…
					{:else if result && !result.filter}
						Rename without checking
					{:else if censored}
						Rename anyway
					{:else}
						Rename server
					{/if}
				</button>
			</div>
		</form>
	</div>
</div>

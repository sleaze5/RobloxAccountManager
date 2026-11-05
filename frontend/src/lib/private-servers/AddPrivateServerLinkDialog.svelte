<script lang="ts">
	import xIcon from "@phosphor-icons/core/regular/x.svg?raw"
	import Icon from "../shared/Icon.svelte"
	import type { SavedServers } from "../games/game-servers-state.svelte"
	import {
		modalBackdropIn,
		modalBackdropOut,
		modalCardIn,
		modalCardOut,
		modalFocus,
		replayValidationNudge,
	} from "../shared/presence"

	let {
		saved,
		gameName,
		onClose,
	}: {
		saved: SavedServers
		gameName: string
		onClose: () => void
	} = $props()

	const id = $props.id()
	let name = $state(""),
		link = $state(""),
		card = $state<HTMLDivElement>()

	function close(): void {
		if (saved.saving) return
		saved.actionError = ""
		onClose()
	}

	async function save(): Promise<void> {
		if (!link.trim()) {
			replayValidationNudge(card)
			return
		}
		if (await saved.save(name, link)) onClose()
		else replayValidationNudge(card)
	}
</script>

<svelte:window
	onkeydown={(event) => {
		if (event.key === "Escape") close()
	}} />

<div class="dialog-backdrop" in:modalBackdropIn out:modalBackdropOut>
	<div
		class="dialog-card max-w-md text-left"
		role="dialog"
		aria-modal="true"
		aria-labelledby={`${id}-title`}
		aria-describedby={`${id}-description`}
		bind:this={card}
		use:modalFocus
		in:modalCardIn
		out:modalCardOut>
		<div class="dialog-heading">
			<div>
				<h2 id={`${id}-title`}>Add private server link</h2>
				<p id={`${id}-description`}>
					Saved in the vault for {gameName}. Any account can launch it.
				</p>
			</div>
			<button
				type="button"
				aria-label="Close add private server link"
				disabled={saved.saving}
				onclick={close}>
				<Icon svg={xIcon} size={14} aria-hidden="true" />
			</button>
		</div>
		<form
			novalidate
			class="contents"
			onsubmit={(event) => {
				event.preventDefault()
				void save()
			}}>
			<div class="dialog-field mt-0">
				<label for={`${id}-link`}>Link</label>
				<input
					id={`${id}-link`}
					class="field font-mono text-2xs"
					type="text"
					autocomplete="off"
					spellcheck="false"
					placeholder="https://www.roblox.com/share?code=…&type=Server"
					aria-describedby={`${id}-link-help`}
					disabled={saved.saving}
					bind:value={link} />
				<small id={`${id}-link-help`} class="text-xs text-fg-muted"
					>A private server link, a server share link, or a link with an
					access code.</small>
			</div>
			<div class="dialog-field">
				<label for={`${id}-name`}>Name (optional)</label>
				<input
					id={`${id}-name`}
					class="field"
					type="text"
					autocomplete="off"
					maxlength={50}
					placeholder="Saved server"
					disabled={saved.saving}
					bind:value={name} />
			</div>
			{#if saved.actionError}
				<div class="mt-3 notice-error" role="alert">{saved.actionError}</div>
			{/if}
			<div class="dialog-actions">
				<button
					class="btn"
					type="button"
					disabled={saved.saving}
					onclick={close}>Cancel</button>
				<button
					class="btn-primary"
					type="submit"
					disabled={saved.saving || !link.trim()}>
					{saved.saving ? "Saving…" : "Save link"}
				</button>
			</div>
		</form>
	</div>
</div>

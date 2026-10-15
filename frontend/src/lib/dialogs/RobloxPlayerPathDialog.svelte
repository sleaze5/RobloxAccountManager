<script lang="ts">
	import ArrowsClockwiseIcon from "phosphor-svelte/lib/ArrowsClockwiseIcon"
	import CheckCircleIcon from "phosphor-svelte/lib/CheckCircleIcon"
	import CircleNotchIcon from "phosphor-svelte/lib/CircleNotchIcon"
	import TrashIcon from "phosphor-svelte/lib/TrashIcon"
	import WarningCircleIcon from "phosphor-svelte/lib/WarningCircleIcon"
	import XIcon from "phosphor-svelte/lib/XIcon"
	import { onMount } from "svelte"
	import type { AccountStore } from "../accounts/account-store.svelte"
	import {
		accountBackend,
		type RobloxPlayerPathCheck,
		type RobloxPlayerVersion,
	} from "../backend/bridge"
	import {
		modalBackdropIn,
		modalBackdropOut,
		modalCardIn,
		modalCardOut,
		modalFocus,
	} from "../shared/presence"
	import { errorMessage } from "../shared/errors"

	let {
		store,
		onClose,
	}: {
		store: AccountStore
		onClose: () => void
	} = $props()

	const automatic = "",
		custom = "custom",
		radioName = $props.id(),
		customPathID = `${radioName}-custom`
	let versions = $state<RobloxPlayerVersion[] | null>(null),
		selection = $state(automatic),
		customPath = $state(""),
		savedCustomPath = "",
		loading = $state(false),
		saving = $state(false),
		validating = $state(false),
		validated = $state(false),
		check = $state<{ text: string; result: RobloxPlayerPathCheck } | null>(null),
		checkSequence = 0,
		error = $state("")

	const newest = $derived(versions?.[0]),
		choices = $derived(versions && versions.length > 1 ? versions : []),
		busy = $derived(saving || validating),
		checked = $derived(check?.text === customPath ? check.result : null),
		parses = $derived(customPath.trim() === "" || checked?.problem === "")

	async function load(): Promise<void> {
		loading = true
		error = ""
		try {
			const result = await accountBackend.GetRobloxPlayers(),
				loaded = result.versions ?? []
			if (versions === null) initialize(loaded, result.customPath)
			else if (
				selection !== automatic &&
				selection !== custom &&
				selectionFor(selection, loaded) !== selection
			) {
				selection = automatic
			}
			versions = loaded
		} catch {
			error = "Roblox versions could not be loaded. Try again."
		} finally {
			loading = false
		}
	}

	function initialize(loaded: RobloxPlayerVersion[], savedPath: string): void {
		const current = store.launchInput.playerPath
		savedCustomPath = savedPath
		customPath = savedPath
		selection = selectionFor(current, loaded)
		if (selection === custom) customPath = current
		void checkPath()
	}

	async function checkPath(): Promise<void> {
		const text = customPath,
			sequence = ++checkSequence
		if (text.trim() === "") {
			check = null
			return
		}
		try {
			const result = await accountBackend.CheckRobloxPlayerPath(text)
			if (sequence === checkSequence) check = { text, result }
		} catch {
			if (sequence === checkSequence) {
				check = {
					text,
					result: {
						path: "",
						found: false,
						problem: "The custom path could not be checked. Try again.",
					},
				}
			}
		}
	}

	async function validate(): Promise<void> {
		const text = customPath
		checkSequence++
		validating = true
		error = ""
		try {
			const result = await accountBackend.CheckRobloxPlayerPath(text)
			if (result.problem !== "") {
				check = { text, result }
				return
			}
			await accountBackend.SetRobloxPlayerPath(result.path)
			customPath = result.path
			savedCustomPath = result.path
			selection = custom
			check = { text: result.path, result }
			validated = true
		} catch (cause) {
			error = errorMessage(
				cause,
				"The custom Roblox Player path could not be saved. Try again.",
			)
		} finally {
			validating = false
		}
	}

	function selectionFor(path: string, loaded: RobloxPlayerVersion[]): string {
		if (path === "") return automatic
		if (!loaded.some((version) => version.path === path)) return custom
		return loaded.length > 1 ? path : automatic
	}

	async function save(): Promise<void> {
		const path = customPath.trim()
		saving = true
		error = ""
		try {
			if (path !== savedCustomPath) {
				await accountBackend.SetRobloxPlayerPath(path)
			}
			store.launchInput.playerPath = selection === custom ? path : selection
			onClose()
		} catch (cause) {
			error = errorMessage(
				cause,
				"The custom Roblox Player path could not be saved. Try again.",
			)
		} finally {
			saving = false
		}
	}

	onMount(() => void load())
</script>

<svelte:window
	onkeydown={(event) => {
		if (event.key === "Escape") {
			event.preventDefault()
			onClose()
		}
	}} />

<div
	class="modal-backdrop"
	in:modalBackdropIn
	out:modalBackdropOut
	onclick={(event) => {
		if (event.target === event.currentTarget) onClose()
	}}
	role="presentation">
	<div
		class="modal-card launch-data-card"
		role="dialog"
		aria-modal="true"
		use:modalFocus
		in:modalCardIn
		out:modalCardOut
		aria-labelledby="player-path-title"
		aria-busy={loading || busy}>
		<form
			novalidate
			onsubmit={(event) => {
				event.preventDefault()
				void save()
			}}>
			<div class="modal-heading compact-heading">
				<div>
					<h2 id="player-path-title">Roblox Player location</h2>
					<p>Choose the RobloxPlayerBeta.exe that direct launch starts.</p>
				</div>
				<div class="player-path-actions">
					<button
						class="icon-action"
						type="button"
						aria-label="Reload Roblox versions"
						data-tooltip="Reload Roblox versions"
						disabled={loading || busy}
						onclick={() => void load()}>
						{#if loading}<CircleNotchIcon
								class="spinner"
								size={16}
								aria-hidden="true" />
						{:else}<ArrowsClockwiseIcon size={16} aria-hidden="true" />{/if}
					</button>
					<button
						class="icon-action"
						type="button"
						aria-label="Close Roblox Player location dialog"
						onclick={onClose}>
						<XIcon size={17} aria-hidden="true" />
					</button>
				</div>
			</div>
			<div
				class="settings-radio-options player-path-options"
				role="radiogroup"
				aria-labelledby="player-path-title">
				<label
					class="settings-radio-option"
					class:disabled={busy}
					data-tooltip={newest?.displayPath}>
					<span class="settings-radio">
						<input
							type="radio"
							name={radioName}
							bind:group={selection}
							value={automatic}
							disabled={busy} />
						<span aria-hidden="true"></span>
					</span>
					<span class="settings-radio-label">Automatic</span>
					<small>
						{#if versions === null}
							{loading ? "Finding Roblox versions" : ""}
						{:else if newest}
							{newest.name}
						{:else}
							No Roblox versions were found
						{/if}
					</small>
				</label>
				{#each choices as version (version.path)}
					<label
						class="settings-radio-option"
						class:disabled={busy}
						data-tooltip={version.displayPath}>
						<span class="settings-radio">
							<input
								type="radio"
								name={radioName}
								bind:group={selection}
								value={version.path}
								disabled={busy} />
							<span aria-hidden="true"></span>
						</span>
						<span class="settings-radio-label player-path-version"
							>{version.name}</span>
					</label>
				{/each}
				<label class="settings-radio-option" class:disabled={busy}>
					<span class="settings-radio">
						<input
							type="radio"
							name={radioName}
							bind:group={selection}
							value={custom}
							disabled={busy} />
						<span aria-hidden="true"></span>
					</span>
					<span class="settings-radio-label">Use a custom path</span>
				</label>
			</div>
			<div class="modal-field">
				<span><label for={customPathID}>Custom path</label></span>
				<div class="player-path-custom">
					<input
						id={customPathID}
						bind:value={customPath}
						type="text"
						placeholder="Path to binary"
						spellcheck="false"
						autocomplete="off"
						aria-invalid={checked?.problem ? true : undefined}
						disabled={busy}
						oninput={() => {
							selection = custom
							validated = false
							void checkPath()
						}} />
					<button
						class="player-path-validate"
						type="button"
						disabled={busy || customPath.trim() === ""}
						onclick={() => void validate()}>
						{#if validating}<CircleNotchIcon
								class="spinner"
								size={14}
								aria-hidden="true" />{/if}
						Validate
					</button>
					<button
						class="icon-action"
						type="button"
						aria-label="Clear custom path"
						data-tooltip="Clear custom path"
						disabled={busy || customPath.length === 0}
						onclick={() => {
							customPath = ""
							check = null
							validated = false
							checkSequence++
							if (selection === custom) selection = automatic
						}}>
						<TrashIcon size={17} aria-hidden="true" />
					</button>
				</div>
				<div class="player-path-status" aria-live="polite">
					{#if checked?.problem}
						<span class="problem"
							><WarningCircleIcon
								size={14}
								aria-hidden="true" />{checked.problem}</span>
					{:else if validated && checked?.found}
						<span class="found"
							><CheckCircleIcon size={14} aria-hidden="true" />Found.
							Custom path saved.</span>
					{:else if validated && checked}
						<span class="missing"
							><WarningCircleIcon size={14} aria-hidden="true" />Saved,
							but the binary was not found.</span>
					{/if}
				</div>
			</div>
			{#if error}
				<div class="modal-error" role="alert">{error}</div>
			{/if}
			<div class="modal-actions">
				<button type="button" disabled={busy} onclick={onClose}>Cancel</button>
				<button class="primary-action" type="submit" disabled={busy || !parses}>
					Save
				</button>
			</div>
		</form>
	</div>
</div>

<style>
	.player-path-actions {
		display: flex;
		gap: var(--space-1);
	}

	.player-path-options small {
		margin-left: auto;
		color: var(--color-text-faint);
		font-family: var(--font-mono);
		font-size: var(--text-11);
	}

	.player-path-custom {
		display: flex;
		align-items: center;
		gap: var(--space-2);
	}

	.player-path-custom input {
		flex: 1;
		min-width: 0;
		font-family: var(--font-mono);
		font-size: var(--text-11);
	}

	.player-path-validate {
		display: inline-flex;
		min-height: var(--control-height-lg);
		align-items: center;
		gap: var(--space-1);
		padding: 0 var(--space-3);
		border: 1px solid var(--color-border);
		border-radius: var(--radius-control);
		background: transparent;
		color: var(--color-text-muted);
		font-size: var(--text-12);
		font-weight: var(--font-semibold);
		cursor: pointer;
		transition:
			border-color 80ms var(--ease-standard),
			background-color 80ms var(--ease-standard);
	}

	.player-path-validate:hover:not(:disabled) {
		border-color: var(--color-border-focus);
		background: var(--color-surface-raised-hover);
	}

	.player-path-validate:disabled {
		color: var(--color-text-faint);
		cursor: not-allowed;
	}

	.player-path-status span {
		display: flex;
		align-items: center;
		gap: var(--space-1);
		margin-top: var(--space-2);
		font-size: var(--text-12);
	}

	.player-path-status .problem {
		color: var(--color-danger-muted);
	}

	.player-path-status .found {
		color: var(--color-success);
	}

	.player-path-status .missing {
		color: var(--color-warning);
	}

	.player-path-version {
		font-family: var(--font-mono);
		font-size: var(--text-11);
	}
</style>

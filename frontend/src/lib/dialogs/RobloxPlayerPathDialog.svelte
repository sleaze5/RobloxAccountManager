<script lang="ts">
	import arrowsClockwiseIcon from "@phosphor-icons/core/regular/arrows-clockwise.svg?raw"
	import checkCircleIcon from "@phosphor-icons/core/regular/check-circle.svg?raw"
	import circleNotchIcon from "@phosphor-icons/core/regular/circle-notch.svg?raw"
	import warningCircleIcon from "@phosphor-icons/core/regular/warning-circle.svg?raw"
	import xIcon from "@phosphor-icons/core/regular/x.svg?raw"
	import Icon from "../shared/Icon.svelte"
	import { onMount } from "svelte"
	import type { AccountStore } from "../accounts/account-store.svelte"
	import {
		accountBackend,
		type RobloxPlayerPathCheck,
		type RobloxPlayerVersion,
	} from "../backend/bridge"
	import { errorMessage } from "../shared/errors"
	import Dialog from "../shared/Dialog.svelte"

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

<Dialog
	class="max-w-md"
	aria-labelledby="player-path-title"
	aria-busy={loading || busy}
	onBackdropClick={onClose}>
	<form
		novalidate
		onsubmit={(event) => {
			event.preventDefault()
			void save()
		}}>
		<div class="dialog-heading mb-2">
			<div>
				<h2 id="player-path-title">Roblox Player location</h2>
				<p>Choose the RobloxPlayerBeta.exe that direct launch starts.</p>
			</div>
			<div class="flex gap-1">
				<button
					class="button-ghost"
					type="button"
					aria-label="Reload Roblox versions"
					data-tooltip="Reload Roblox versions"
					disabled={loading || busy}
					onclick={() => void load()}>
					{#if loading}<Icon
							svg={circleNotchIcon}
							class="spinner"
							size={16}
							aria-hidden="true" />
					{:else}<Icon
							svg={arrowsClockwiseIcon}
							data-turn
							size={16}
							aria-hidden="true" />{/if}
				</button>
				<button
					class="button-ghost-sm"
					type="button"
					aria-label="Close Roblox Player location dialog"
					onclick={onClose}>
					<Icon svg={xIcon} size={14} aria-hidden="true" />
				</button>
			</div>
		</div>
		<div
			class="flex flex-col gap-0.5 pl-1"
			role="radiogroup"
			aria-labelledby="player-path-title">
			<label
				class="radio-option"
				data-disabled={busy || undefined}
				data-tooltip={newest?.displayPath}>
				<span class="radio">
					<input
						type="radio"
						name={radioName}
						bind:group={selection}
						value={automatic}
						disabled={busy} />
					<span aria-hidden="true"></span>
				</span>
				<span class="text-xs text-fg">Automatic</span>
				<small class="ml-auto font-mono text-2xs text-fg-faint">
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
					class="radio-option"
					data-disabled={busy || undefined}
					data-tooltip={version.displayPath}>
					<span class="radio">
						<input
							type="radio"
							name={radioName}
							bind:group={selection}
							value={version.path}
							disabled={busy} />
						<span aria-hidden="true"></span>
					</span>
					<span class="font-mono text-2xs text-fg">{version.name}</span>
				</label>
			{/each}
			<label class="radio-option" data-disabled={busy || undefined}>
				<span class="radio">
					<input
						type="radio"
						name={radioName}
						bind:group={selection}
						value={custom}
						disabled={busy} />
					<span aria-hidden="true"></span>
				</span>
				<span class="text-xs text-fg">Use a custom path</span>
			</label>
		</div>
		<div class="dialog-field">
			<span><label for={customPathID}>Custom path</label></span>
			<div class="flex items-center gap-2">
				<input
					class="min-w-0 flex-1 font-mono text-2xs"
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
					class="button h-8 px-3 font-semibold"
					type="button"
					disabled={busy || customPath.trim() === ""}
					onclick={() => void validate()}>
					{#if validating}<Icon
							svg={circleNotchIcon}
							class="spinner"
							size={14}
							aria-hidden="true" />{/if}
					Validate
				</button>
				<button
					class="button-ghost"
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
					<Icon svg={xIcon} size={16} aria-hidden="true" />
				</button>
			</div>
			<div
				class="*:mt-2 *:flex *:items-center *:gap-1 *:text-xs"
				aria-live="polite">
				{#if checked?.problem}
					<span class="text-danger-muted"
						><Icon
							svg={warningCircleIcon}
							size={14}
							aria-hidden="true" />{checked.problem}</span>
				{:else if validated && checked?.found}
					<span class="text-success"
						><Icon
							svg={checkCircleIcon}
							size={14}
							aria-hidden="true" />Found. Custom path saved.</span>
				{:else if validated && checked}
					<span class="text-warning"
						><Icon
							svg={warningCircleIcon}
							size={14}
							aria-hidden="true" />Saved, but the binary was not found.</span>
				{/if}
			</div>
		</div>
		{#if error}
			<div class="mt-3 notice-error" role="alert">{error}</div>
		{/if}
		<div class="dialog-actions">
			<button class="button" type="button" disabled={busy} onclick={onClose}
				>Cancel</button>
			<button class="button-primary" type="submit" disabled={busy || !parses}>
				Save
			</button>
		</div>
	</form>
</Dialog>

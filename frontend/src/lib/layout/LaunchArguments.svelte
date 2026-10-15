<script lang="ts">
	import ArrowRightIcon from "phosphor-svelte/lib/ArrowRightIcon"
	import CheckIcon from "phosphor-svelte/lib/CheckIcon"
	import SlidersIcon from "phosphor-svelte/lib/SlidersIcon"
	import { onMount } from "svelte"
	import type { AccountStore } from "../accounts/account-store.svelte"
	import { accountBackend, type RobloxClientState } from "../backend/bridge"
	import LaunchDataDialog from "../dialogs/LaunchDataDialog.svelte"
	import LinuxClientDialog from "../dialogs/LinuxClientDialog.svelte"
	import RobloxPlayerPathDialog from "../dialogs/RobloxPlayerPathDialog.svelte"
	import { linuxClientOptions } from "../settings/linux-clients"
	import { formatCompactBytes } from "../shared/bytes"
	import { menuIn, menuOut } from "../shared/presence"

	let { store }: { store: AccountStore } = $props()
	let clients = $state<RobloxClientState | null>(null)
	const directLaunch = $derived(clients?.directLaunch ?? true),
		clientLabel = $derived(
			clients
				? (linuxClientOptions(clients).find(
						(option) => option.value === clients?.selected,
					)?.label ?? "Automatic")
				: "",
		)
	const launchDataSize = $derived(
			formatCompactBytes(
				new TextEncoder().encode(store.launchInput.launchData).length,
			),
		),
		activeOptions = $derived(
			[
				directLaunch && store.launchInput.directLaunch && "Direct launch",
				store.launchInput.teleport && "Teleport",
				store.launchDataEnabled && `Custom launch data (${launchDataSize})`,
			].filter((option) => typeof option === "string"),
		),
		triggerLabel = $derived(
			activeOptions.length > 0
				? `Launch options: ${activeOptions.join(", ")}`
				: "Additional launch options",
		)
	async function loadClients(): Promise<void> {
		clients = await accountBackend.GetRobloxClients()
		if (!clients.directLaunch) store.launchInput.directLaunch = false
	}
	onMount(() => void loadClients())
	let open = $state(false),
		pathOpen = $state(false),
		clientOpen = $state(false),
		launchDataOpen = $state(false),
		root = $state<HTMLDivElement>(),
		trigger = $state<HTMLButtonElement>()
	const panelID = $props.id()
	$effect(() => {
		if (store.launching || !store.vault.unlocked) open = false
	})
	$effect(() => {
		if (!store.vault.unlocked) {
			pathOpen = false
			clientOpen = false
			launchDataOpen = false
		}
	})
</script>

<svelte:window
	onclick={(event) => {
		if (event.target instanceof Node && !root?.contains(event.target)) open = false
	}}
	onkeydown={(event) => {
		if (open && event.key === "Escape") {
			event.preventDefault()
			open = false
			trigger?.focus()
		}
	}} />

<div class="launch-arguments" bind:this={root}>
	<button
		class="icon-action"
		class:active={activeOptions.length > 0}
		type="button"
		bind:this={trigger}
		aria-label={triggerLabel}
		aria-expanded={open}
		aria-controls={open ? panelID : undefined}
		data-tooltip={triggerLabel}
		data-tooltip-side="top"
		disabled={store.launching}
		onclick={() => {
			open = !open
			if (open) void loadClients()
		}}>
		<SlidersIcon size={17} aria-hidden="true" />
	</button>
	{#if open}
		<section
			class="launch-arguments-menu"
			id={panelID}
			aria-label="Additional launch options"
			in:menuIn
			out:menuOut>
			<strong>Additional launch options</strong>
			{#if directLaunch}
				<div class="launch-option-group">
					<label class="launch-option">
						<span class="cookie-selection">
							<input
								type="checkbox"
								bind:checked={store.launchInput.directLaunch}
								disabled={store.launching} />
							<span aria-hidden="true"
								><CheckIcon
									size={11}
									weight="bold"
									aria-hidden="true" /></span>
						</span>
						<span
							><strong>Launch binary directly</strong><small
								>Find and launch RobloxPlayerBeta.exe directly instead
								of using the registered Roblox launcher.</small
							></span>
					</label>
					<button
						class="launch-option-action"
						type="button"
						aria-haspopup="dialog"
						disabled={store.launching}
						onclick={() => {
							trigger?.focus()
							open = false
							pathOpen = true
						}}>
						Set location
						<ArrowRightIcon size={12} aria-hidden="true" />
					</button>
				</div>
			{:else if clients?.choiceSupported}
				<div class="launch-option-group">
					<span class="launch-option-copy"
						><strong>Linux client</strong><small>{clientLabel}</small
						></span>
					<button
						class="launch-option-action"
						type="button"
						aria-haspopup="dialog"
						disabled={store.launching}
						onclick={() => {
							trigger?.focus()
							open = false
							clientOpen = true
						}}>
						Choose client
						<ArrowRightIcon size={12} aria-hidden="true" />
					</button>
				</div>
			{/if}
			<label class="launch-option">
				<span class="cookie-selection">
					<input
						type="checkbox"
						bind:checked={store.launchInput.teleport}
						disabled={store.launching} />
					<span aria-hidden="true"
						><CheckIcon size={11} weight="bold" aria-hidden="true" /></span>
				</span>
				<span
					><strong>Mark launch as a teleport</strong><small
						>Set the isTeleport flag to launch the game as an in-game
						teleport.</small
					></span>
			</label>
			<div class="launch-option-group">
				<label class="launch-option">
					<span class="cookie-selection">
						<input
							type="checkbox"
							bind:checked={store.launchDataEnabled}
							disabled={store.launching} />
						<span aria-hidden="true"
							><CheckIcon
								size={11}
								weight="bold"
								aria-hidden="true" /></span>
					</span>
					<span
						><strong>Use custom launch data</strong><small
							>Set the launchData flag to pass custom data to the game.</small
						></span>
				</label>
				<button
					class="launch-option-action"
					type="button"
					aria-haspopup="dialog"
					disabled={store.launching}
					onclick={() => {
						trigger?.focus()
						open = false
						launchDataOpen = true
					}}>
					Set data
					<ArrowRightIcon size={12} aria-hidden="true" />
				</button>
			</div>
		</section>
	{/if}
</div>
{#if pathOpen}
	<RobloxPlayerPathDialog {store} onClose={() => (pathOpen = false)} />
{/if}
{#if launchDataOpen}
	<LaunchDataDialog {store} onClose={() => (launchDataOpen = false)} />
{/if}
{#if clientOpen && clients}
	<LinuxClientDialog
		{clients}
		onSaved={(state) => (clients = state)}
		onClose={() => (clientOpen = false)} />
{/if}

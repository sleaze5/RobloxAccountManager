<script lang="ts">
	import CheckboxMark from "../shared/CheckboxMark.svelte"
	import arrowRightIcon from "@phosphor-icons/core/regular/arrow-right.svg?raw"
	import slidersIcon from "@phosphor-icons/core/regular/sliders.svg?raw"
	import Icon from "../shared/Icon.svelte"
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

{#snippet dialogLink(label: string, isIndented: boolean, openDialog: () => void)}
	<button
		class={[
			"inline-flex items-center gap-1 text-xs font-medium text-fg transition-colors duration-100 enabled:hover:text-fg-bright",
			isIndented && "ml-5",
		]}
		type="button"
		aria-haspopup="dialog"
		disabled={store.launching}
		onclick={() => {
			trigger?.focus()
			open = false
			openDialog()
		}}>
		{label}
		<Icon svg={arrowRightIcon} data-nudge="right" size={12} aria-hidden="true" />
	</button>
{/snippet}

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

<div class="relative" bind:this={root}>
	<button
		class={["button-ghost", activeOptions.length > 0 && "text-accent-strong"]}
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
		<Icon svg={slidersIcon} size={16} aria-hidden="true" />
	</button>
	{#if open}
		<section
			class="absolute right-0 bottom-full z-10 mb-2 flex w-80 flex-col gap-3 rounded-xl border border-line-strong bg-overlay p-3 shadow-menu"
			style="--menu-origin: bottom right"
			id={panelID}
			aria-label="Additional launch options"
			in:menuIn
			out:menuOut>
			<strong class="text-xs font-semibold text-fg"
				>Additional launch options</strong>
			{#if directLaunch}
				<div class="flex flex-col items-start gap-2">
					<label class="check-option">
						<span class="checkbox">
							<input
								type="checkbox"
								bind:checked={store.launchInput.directLaunch}
								disabled={store.launching} />
							<CheckboxMark />
						</span>
						<span
							><strong>Launch binary directly</strong><small
								>Find and launch RobloxPlayerBeta.exe directly instead
								of using the registered Roblox launcher.</small
							></span>
					</label>
					{@render dialogLink("Set location", true, () => (pathOpen = true))}
				</div>
			{:else if clients?.choiceSupported}
				<div class="flex flex-col items-start gap-2">
					<span class="flex flex-col gap-1"
						><strong class="text-xs font-semibold text-fg"
							>Linux client</strong
						><small class="text-2xs leading-normal text-fg-muted"
							>{clientLabel}</small
						></span>
					{@render dialogLink(
						"Choose client",
						false,
						() => (clientOpen = true),
					)}
				</div>
			{/if}
			<label class="check-option">
				<span class="checkbox">
					<input
						type="checkbox"
						bind:checked={store.launchInput.teleport}
						disabled={store.launching} />
					<CheckboxMark />
				</span>
				<span
					><strong>Mark launch as a teleport</strong><small
						>Set the isTeleport flag to launch the game as an in-game
						teleport.</small
					></span>
			</label>
			<div class="flex flex-col items-start gap-2">
				<label class="check-option">
					<span class="checkbox">
						<input
							type="checkbox"
							bind:checked={store.launchDataEnabled}
							disabled={store.launching} />
						<CheckboxMark />
					</span>
					<span
						><strong>Use custom launch data</strong><small
							>Set the launchData flag to pass custom data to the game.</small
						></span>
				</label>
				{@render dialogLink("Set data", true, () => (launchDataOpen = true))}
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

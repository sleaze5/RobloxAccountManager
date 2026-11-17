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
	import LaunchPanelPopover from "./LaunchPanelPopover.svelte"

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
	let pathOpen = $state(false),
		clientOpen = $state(false),
		launchDataOpen = $state(false)
	$effect(() => {
		if (!store.vault.unlocked) {
			pathOpen = false
			clientOpen = false
			launchDataOpen = false
		}
	})
</script>

{#snippet dialogLink(
	label: string,
	isIndented: boolean,
	closePanel: () => void,
	openDialog: () => void,
)}
	<button
		class={[
			"inline-flex items-center gap-1 text-xs font-medium text-fg transition-colors duration-100 enabled:hover:text-fg-bright",
			isIndented && "ml-5",
		]}
		type="button"
		aria-haspopup="dialog"
		disabled={store.launching}
		onclick={() => {
			closePanel()
			openDialog()
		}}>
		{label}
		<Icon svg={arrowRightIcon} data-nudge="right" size={12} aria-hidden="true" />
	</button>
{/snippet}

<LaunchPanelPopover
	title="Additional launch options"
	{triggerLabel}
	triggerIcon={slidersIcon}
	isTriggerCompact={false}
	isTriggerHighlighted={activeOptions.length > 0}
	isDisabled={store.launching || !store.vault.unlocked}
	onOpen={() => void loadClients()}>
	{#snippet children(closePanel)}
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
							>Find RobloxPlayerBeta.exe and start it directly, without
							the registered Roblox launcher.</small
						></span>
				</label>
				{@render dialogLink(
					"Set location",
					true,
					closePanel,
					() => (pathOpen = true),
				)}
			</div>
		{:else if clients?.choiceSupported}
			<div class="flex flex-col items-start gap-2">
				<span class="flex flex-col gap-1"
					><strong class="text-xs font-semibold text-fg">Linux client</strong
					><small class="text-2xs leading-normal text-fg-muted"
						>{clientLabel}</small
					></span>
				{@render dialogLink(
					"Select client",
					false,
					closePanel,
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
					>Set the isTeleport flag to launch the game as an in-game teleport.</small
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
			{@render dialogLink(
				"Set data",
				true,
				closePanel,
				() => (launchDataOpen = true),
			)}
		</div>
	{/snippet}
</LaunchPanelPopover>
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

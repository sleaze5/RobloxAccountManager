<script lang="ts">
	import Settings2 from "@lucide/svelte/icons/settings-2"
	import { onMount } from "svelte"
	import type { AccountStore } from "../accounts/account-store.svelte"
	import { accountBackend } from "../backend/bridge"
	import { menuIn, menuOut } from "../shared/presence"

	let { store }: { store: AccountStore } = $props()
	let directLaunch = $state(true)
	onMount(() => {
		void (async () => {
			const state = await accountBackend.GetRobloxClients()
			directLaunch = state.directLaunch
			if (!state.directLaunch) store.launchInput.directLaunch = false
		})()
	})
	let open = $state(false),
		root = $state<HTMLDivElement>(),
		trigger = $state<HTMLButtonElement>()
	const panelID = $props.id()
	$effect(() => {
		if (store.launching || !store.vault.unlocked) open = false
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
		type="button"
		bind:this={trigger}
		aria-label="Additional launch options"
		aria-expanded={open}
		aria-controls={open ? panelID : undefined}
		data-tooltip="Additional launch options"
		data-tooltip-side="top"
		disabled={store.launching}
		onclick={() => (open = !open)}>
		<Settings2 size={15} aria-hidden="true" />
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
				<label class="launch-option">
					<input
						type="checkbox"
						bind:checked={store.launchInput.directLaunch}
						disabled={store.launching} />
					<span
						><strong>Launch binary directly</strong><small
							>Find and launch RobloxPlayerBeta.exe directly instead of
							using the registered Roblox launcher.</small
						></span>
				</label>
			{/if}
			<label class="launch-option">
				<input
					type="checkbox"
					bind:checked={store.launchInput.teleport}
					disabled={store.launching} />
				<span
					><strong>Mark launch as a teleport</strong><small
						>Set the isTeleport flag to launch the game as an in-game
						teleport.</small
					></span>
			</label>
		</section>
	{/if}
</div>

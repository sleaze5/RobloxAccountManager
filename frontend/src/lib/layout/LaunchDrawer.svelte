<script lang="ts">
	import ProhibitIcon from "phosphor-svelte/lib/ProhibitIcon"
	import CaretDownIcon from "phosphor-svelte/lib/CaretDownIcon"
	import FileArrowUpIcon from "phosphor-svelte/lib/FileArrowUpIcon"
	import GameControllerIcon from "phosphor-svelte/lib/GameControllerIcon"
	import TrashIcon from "phosphor-svelte/lib/TrashIcon"
	import type { AccountStore } from "../accounts/account-store.svelte"
	import { LaunchMethod } from "../backend/bridge"
	import Select from "../shared/Select.svelte"
	import LaunchDataDialog from "../dialogs/LaunchDataDialog.svelte"
	import RobloxProcessesDialog from "../dialogs/RobloxProcessesDialog.svelte"
	import { appSettings } from "../settings/settings-store.svelte"
	import LaunchArguments from "./LaunchArguments.svelte"
	import LaunchButton from "./LaunchButton.svelte"
	import NearestServerButton from "./NearestServerButton.svelte"
	import PlaceIdInput from "./PlaceIdInput.svelte"
	import type { WorkspaceState } from "./workspace-state.svelte"

	let { store, workspace }: { store: AccountStore; workspace: WorkspaceState } =
		$props()
	let drawer = $state<HTMLElement>()
	let launchDataOpen = $state(false)
	let processesOpen = $state(false)
	let measuredMethod: LaunchMethod | undefined
	const hasJoinTarget = $derived.by(() => {
		const input = store.launchInput
		return (
			input.method !== LaunchMethod.MethodPlace ||
			input.resolveCurrentGame ||
			[input.placeId, input.jobId, input.link, input.user].some(
				(value) => value.trim().length > 0,
			)
		)
	})
	const methods = [
		{
			value: LaunchMethod.MethodPlace,
			label: "Place ID",
			description: "Join a game by its Place ID and, optionally, its Job ID.",
		},
		{
			value: LaunchMethod.MethodLink,
			label: "Link",
			description:
				"Join using a Roblox game link, private server link, experience invite, or server-share code.",
		},
		{
			value: LaunchMethod.MethodUser,
			label: "User",
			description: "Join the game that a specific user is currently playing.",
		},
	]

	$effect(() => {
		if (!workspace.launchOpen || !drawer) return
		const method = store.launchInput.method,
			panel = drawer,
			body = panel.querySelector<HTMLElement>(".launch-body"),
			content = panel.querySelector<HTMLElement>(".launch-fields")
		if (!body || !content) return
		const observer = new ResizeObserver(() => {
			const style = getComputedStyle(body),
				minimumHeight =
					panel.offsetHeight -
					body.offsetHeight +
					content.offsetHeight +
					Number.parseFloat(style.paddingTop) +
					Number.parseFloat(style.paddingBottom)
			const shouldFit = measuredMethod !== method
			measuredMethod = method
			workspace.setLaunchMinimumHeight(Math.ceil(minimumHeight), shouldFit)
		})
		observer.observe(panel)
		observer.observe(content)
		return () => observer.disconnect()
	})

	$effect(() => {
		if (!workspace.launchOpen || !store.vault.unlocked) launchDataOpen = false
		if (!store.vault.unlocked) processesOpen = false
	})
</script>

<section
	bind:this={drawer}
	class:collapsed={!workspace.launchOpen}
	class="launch-drawer"
	aria-labelledby="launch-title">
	{#if workspace.launchOpen}
		<button
			class="launch-resizer"
			type="button"
			aria-label="Resize launch panel"
			data-tooltip="Drag to resize the launch panel"
			data-tooltip-side="top"
			onkeydown={(event) => {
				if (event.key === "ArrowUp" || event.key === "ArrowDown") {
					event.preventDefault()
					workspace.resizeLaunch(
						workspace.launchHeight + (event.key === "ArrowUp" ? 16 : -16),
					)
				}
			}}
			onpointerdown={(event) => workspace.startLaunchResize(event)}></button>
	{/if}
	<div class="drawer-header">
		<span>
			<GameControllerIcon size={18} aria-hidden="true" />
			<strong id="launch-title">Launch options</strong>
		</span>
		<div class="drawer-header-actions">
			<button
				class="icon-action close-roblox-action"
				type="button"
				aria-label="Kill Roblox processes"
				aria-haspopup="dialog"
				aria-expanded={processesOpen}
				data-tooltip="Kill Roblox processes"
				data-tooltip-side="top"
				disabled={!store.vault.unlocked}
				onclick={() => {
					workspace.closeAccountMenus()
					processesOpen = true
				}}>
				<ProhibitIcon size={16} aria-hidden="true" />
			</button>
			<button
				class="drawer-toggle"
				type="button"
				aria-label={workspace.launchOpen
					? "Collapse the launch panel"
					: "Expand the launch panel"}
				aria-expanded={workspace.launchOpen}
				data-tooltip={workspace.launchOpen
					? "Collapse the launch panel"
					: "Expand the launch panel"}
				data-tooltip-side="top-end"
				onclick={() => (workspace.launchOpen = !workspace.launchOpen)}>
				<CaretDownIcon
					class={workspace.launchOpen ? "open" : undefined}
					size={18}
					aria-hidden="true" />
			</button>
		</div>
	</div>
	{#if workspace.launchOpen}
		<form
			class="drawer-content"
			novalidate
			aria-busy={store.launching}
			onsubmit={(event) => {
				event.preventDefault()
				void store.launchSelectedAccounts()
			}}>
			<div class="launch-body">
				<div
					class="launch-fields"
					class:launch-user={store.launchInput.method ===
						LaunchMethod.MethodUser}
					class:launch-place={store.launchInput.method ===
						LaunchMethod.MethodPlace}>
					{#if store.launchInput.method === LaunchMethod.MethodPlace}
						<PlaceIdInput {store} />
						<div class="launch-job">
							<label
								><span>Job ID <small>(optional)</small></span>
								<input
									bind:value={store.launchInput.jobId}
									type="text"
									placeholder="Server UUID"
									disabled={store.launching} />
							</label>
							{#if appSettings.roValraEnabled}
								<NearestServerButton {store} />
							{/if}
						</div>
					{:else if store.launchInput.method === LaunchMethod.MethodLink}
						<label
							><span>Roblox links</span>
							<input
								bind:value={store.launchInput.link}
								type="text"
								placeholder="Paste link here"
								spellcheck="false"
								autocomplete="off"
								required
								disabled={store.launching} />
						</label>
					{:else}
						<label
							><span>Username / User ID</span>
							<input
								bind:value={store.launchInput.user}
								type="text"
								placeholder="Username or id=123"
								required
								disabled={store.launching} />
						</label>
						<label class="launch-option launch-resolve"
							><input
								bind:checked={store.launchInput.resolveCurrentGame}
								type="checkbox"
								disabled={store.launching} /><span
								>Resolve current game</span
							></label>
					{/if}
				</div>
			</div>
			<div class="launch-footer">
				<div class="launch-method">
					<span>Method:</span>
					<Select
						label="Launch method"
						value={store.launchInput.method}
						options={methods}
						disabled={store.launching}
						onChange={(method) => (store.launchInput.method = method)} />
					<button
						class="icon-action"
						type="button"
						aria-label="Clear join fields"
						data-tooltip="Clear join fields and reset to Place ID"
						data-tooltip-side="top"
						disabled={store.launching || !hasJoinTarget}
						onclick={() => store.clearJoinTarget()}>
						<TrashIcon size={17} aria-hidden="true" />
					</button>
				</div>
				<button
					class="icon-action"
					class:active={store.launchInput.launchData.trim().length > 0}
					type="button"
					aria-label="Launch data"
					aria-haspopup="dialog"
					aria-expanded={launchDataOpen}
					data-tooltip="Launch data"
					data-tooltip-side="top"
					disabled={store.launching}
					onclick={() => (launchDataOpen = true)}>
					<FileArrowUpIcon size={17} aria-hidden="true" />
				</button>
				<LaunchArguments {store} />
				<LaunchButton {store} {workspace} />
			</div>
		</form>
	{/if}
	{#if launchDataOpen}
		<LaunchDataDialog {store} onClose={() => (launchDataOpen = false)} />
	{/if}
	{#if processesOpen}
		<RobloxProcessesDialog onClose={() => (processesOpen = false)} />
	{/if}
</section>

<style>
	.drawer-header-actions {
		display: flex;
		align-items: center;
		gap: var(--space-1);
	}

	.close-roblox-action {
		width: var(--control-height-sm);
		height: var(--control-height-sm);
		color: var(--color-danger);
	}

	.close-roblox-action:hover {
		background: var(--color-danger-wash);
		color: var(--color-danger-muted);
	}
</style>

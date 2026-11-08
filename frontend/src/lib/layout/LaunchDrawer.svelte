<script lang="ts">
	import CheckboxMark from "../shared/CheckboxMark.svelte"
	import caretDownIcon from "@phosphor-icons/core/regular/caret-down.svg?raw"
	import eraserIcon from "@phosphor-icons/core/regular/eraser.svg?raw"
	import gameControllerIcon from "@phosphor-icons/core/regular/game-controller.svg?raw"
	import prohibitIcon from "@phosphor-icons/core/regular/prohibit.svg?raw"
	import Icon from "../shared/Icon.svelte"
	import type { AccountStore } from "../accounts/account-store.svelte"
	import { LaunchMethod } from "../backend/bridge"
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
	let processesOpen = $state(false)
	let measuredMethod: LaunchMethod | undefined
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
			body = panel.querySelector<HTMLElement>("[data-launch-body]"),
			content = panel.querySelector<HTMLElement>("[data-launch-fields]")
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
		if (!store.vault.unlocked) processesOpen = false
	})
</script>

<svelte:window bind:innerHeight={workspace.viewportHeight} />

<section
	bind:this={drawer}
	class={[
		"shrink-0 border-t border-line-subtle bg-surface",
		workspace.launchOpen ? "flex h-(--launch-height) flex-col" : "h-9",
	]}
	aria-labelledby="launch-title">
	{#if workspace.launchOpen}
		<div
			class="relative z-10 h-0.5 w-full shrink-0 cursor-row-resize touch-none bg-surface before:absolute before:inset-x-0 before:-top-1.5 before:bottom-0"
			role="slider"
			tabindex="0"
			aria-label="Resize launch panel"
			aria-orientation="horizontal"
			aria-valuenow={workspace.launchHeight}
			aria-valuemin={workspace.launchMinimumHeight}
			aria-valuemax={workspace.launchMaximumHeight}
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
			onpointerdown={(event) => workspace.startLaunchResize(event)}>
		</div>
	{/if}
	<div
		class={[
			"flex h-9 w-full shrink-0 items-center justify-between px-3 text-fg-muted select-none",
			workspace.launchOpen && "border-b border-line-subtle",
		]}>
		<span class="flex items-center gap-2">
			<Icon svg={gameControllerIcon} size={16} aria-hidden="true" />
			<strong id="launch-title" class="text-2xs font-semibold text-fg"
				>Launch options</strong>
		</span>
		<div class="flex items-center gap-1">
			<button
				class="button-ghost-sm hover:bg-danger/5 hover:text-danger aria-expanded:bg-danger/5 aria-expanded:text-danger"
				type="button"
				aria-label="Close Roblox processes"
				aria-haspopup="dialog"
				aria-expanded={processesOpen}
				data-tooltip="Close Roblox processes"
				data-tooltip-side="top"
				disabled={!store.vault.unlocked}
				onclick={() => {
					workspace.closeAccountMenus()
					processesOpen = true
				}}>
				<Icon svg={prohibitIcon} size={16} aria-hidden="true" />
			</button>
			<button
				class="button-ghost-sm"
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
				<Icon
					svg={caretDownIcon}
					class={[
						"transition-transform duration-100",
						workspace.launchOpen && "rotate-180",
					]}
					size={14}
					aria-hidden="true" />
			</button>
		</div>
	</div>
	{#if workspace.launchOpen}
		<form
			class="flex min-h-0 flex-1 flex-col"
			novalidate
			aria-busy={store.launching}
			onsubmit={(event) => {
				event.preventDefault()
				void store.launchSelectedAccounts()
			}}>
			<div
				class="flex min-h-0 flex-1 flex-col gap-2 overflow-auto p-3 *:shrink-0"
				data-launch-body>
				<div
					data-launch-fields
					class={[
						"flex w-full min-w-0",
						store.launchInput.method === LaunchMethod.MethodLink
							? "max-w-md flex-col gap-2"
							: "max-w-lg items-end gap-2",
					]}>
					{#if store.launchInput.method === LaunchMethod.MethodPlace}
						<PlaceIdInput {store} />
						<div class="flex min-w-0 flex-1 items-end gap-1">
							<label class="min-w-0 flex-1"
								><span class="mb-1 block text-xs text-fg-muted"
									>Job ID <small class="text-xs text-fg-faint"
										>(optional)</small
									></span>
								<input
									class="field font-mono text-2xs"
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
						<label class="min-w-0"
							><span class="mb-1 block text-xs text-fg-muted"
								>Roblox links</span>
							<input
								class="field font-mono text-2xs"
								data-launch-link
								bind:value={store.launchInput.link}
								type="text"
								placeholder="Paste link here"
								spellcheck="false"
								autocomplete="off"
								required
								disabled={store.launching} />
						</label>
					{:else}
						<label class="min-w-0 flex-1"
							><span class="mb-1 block text-xs text-fg-muted"
								>Username / User ID</span>
							<input
								class="field font-mono text-2xs"
								bind:value={store.launchInput.user}
								type="text"
								placeholder="Username or id=123"
								required
								disabled={store.launching} />
						</label>
						<label class="check-option h-8 items-center whitespace-nowrap">
							<span class="checkbox">
								<input
									bind:checked={store.launchInput.resolveCurrentGame}
									type="checkbox"
									disabled={store.launching} />
								<CheckboxMark />
							</span>
							<span>Resolve current game</span>
						</label>
					{/if}
				</div>
			</div>
			<div
				class="flex h-10 shrink-0 items-center justify-end gap-2 border-t border-line-subtle px-3 py-1">
				<div
					class="mr-auto flex shrink-0 items-center gap-2 text-xs text-fg-muted">
					<span id="launch-method-label">Method:</span>
					<div
						class="segmented"
						role="group"
						aria-labelledby="launch-method-label">
						{#each methods as method (method.value)}
							<button
								type="button"
								aria-pressed={store.launchInput.method === method.value}
								data-tooltip={method.description}
								data-tooltip-side="top"
								disabled={store.launching}
								onclick={() =>
									(store.launchInput.method = method.value)}>
								{method.label}
							</button>
						{/each}
					</div>
					<button
						class="button-ghost"
						type="button"
						aria-label="Clear join fields"
						data-tooltip="Clear join fields and reset to Place ID"
						data-tooltip-side="top"
						disabled={store.launching}
						onclick={() => store.clearJoinTarget()}>
						<Icon svg={eraserIcon} size={16} aria-hidden="true" />
					</button>
				</div>
				<LaunchArguments {store} />
				<LaunchButton {store} {workspace} />
			</div>
		</form>
	{/if}
	{#if processesOpen}
		<RobloxProcessesDialog onClose={() => (processesOpen = false)} />
	{/if}
</section>

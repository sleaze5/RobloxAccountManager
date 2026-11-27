<script lang="ts">
	import CheckboxMark from "../shared/CheckboxMark.svelte"
	import caretDownIcon from "@phosphor-icons/core/regular/caret-down.svg?raw"
	import eraserIcon from "@phosphor-icons/core/regular/eraser.svg?raw"
	import gameControllerIcon from "@phosphor-icons/core/regular/game-controller.svg?raw"
	import prohibitIcon from "@phosphor-icons/core/regular/prohibit.svg?raw"
	import { untrack } from "svelte"
	import Icon from "../shared/Icon.svelte"
	import type { AccountStore } from "../accounts/account-store.svelte"
	import type { NotificationCenter } from "../notifications/notification-center.svelte"
	import { LaunchMethod, RobloxCleanScope } from "../backend/bridge"
	import type { AccountJoinCheck } from "../backend/bridge"
	import RobloxCleanDialog from "../dialogs/RobloxCleanDialog.svelte"
	import RobloxProcessesDialog from "../dialogs/RobloxProcessesDialog.svelte"
	import { appSettings } from "../settings/settings-store.svelte"
	import ClientOptions from "./ClientOptions.svelte"
	import LaunchArguments from "./LaunchArguments.svelte"
	import LaunchButton from "./LaunchButton.svelte"
	import { LaunchJoinCheck } from "./launch-join-check.svelte"
	import LaunchJoinWarningDialog from "./LaunchJoinWarningDialog.svelte"
	import NearestServerButton from "./NearestServerButton.svelte"
	import PlaceIdInput from "./PlaceIdInput.svelte"
	import type { WorkspaceState } from "./workspace-state.svelte"

	let {
		store,
		workspace,
		notifications,
	}: {
		store: AccountStore
		workspace: WorkspaceState
		notifications: NotificationCenter
	} = $props()
	let processesOpen = $state(false),
		cleanOpen = $state(false)
	const methods = [
		{
			value: LaunchMethod.MethodPlace,
			label: "Place ID",
			description:
				"Join a game by place ID and an optional server ID (UUID). Leave both empty to open Roblox without joining a game.",
		},
		{
			value: LaunchMethod.MethodLink,
			label: "Link",
			description:
				"Join with a game link, private server link, game invite, or 32-character server share code. Leave it empty to open Roblox without joining a game.",
		},
		{
			value: LaunchMethod.MethodUser,
			label: "User",
			description:
				"Join the game that a user is playing. Leave it empty to open Roblox without joining a game.",
		},
	]

	$effect(() => {
		if (!store.vault.unlocked) {
			processesOpen = false
			cleanOpen = false
		}
	})

	const joinCheck = new LaunchJoinCheck()

	// The check reruns when the join target or the selected accounts change,
	// and not for launch options such as Teleport, which do not change it.
	$effect(() => {
		const { method, placeId, serverId, link, user, resolveCurrentGame } =
				store.launchInput,
			accountIds = [...store.selectedAccountIds],
			isChecked = workspace.launchOpen && store.vault.unlocked
		void [method, placeId, serverId, link, user, resolveCurrentGame]
		untrack(() => {
			if (isChecked)
				joinCheck.schedule(accountIds, $state.snapshot(store.launchInput))
			else joinCheck.cancel()
		})
	})

	$effect(() => () => joinCheck.cancel())

	let isCheckingJoin = $state(false),
		joinRestrictions = $state.raw<AccountJoinCheck[]>([])

	// Launch asks before it starts a join that Roblox would likely refuse. The
	// user can still launch.
	async function launchAfterJoinCheck(): Promise<void> {
		if (isCheckingJoin || store.launching) return
		isCheckingJoin = true
		try {
			joinRestrictions = await joinCheck.readRestrictions()
		} finally {
			isCheckingJoin = false
		}
		if (joinRestrictions.length === 0) void store.launchSelectedAccounts()
	}

	function launchAnyway(): void {
		joinRestrictions = []
		void store.launchSelectedAccounts()
	}
</script>

<section
	class={[
		"shrink-0 border-t border-line-subtle bg-surface",
		workspace.launchOpen ? "flex flex-col" : "h-9",
	]}
	aria-labelledby="launch-title">
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
			<ClientOptions {store} />
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
			class="flex flex-col"
			novalidate
			aria-busy={store.launching}
			onsubmit={(event) => {
				event.preventDefault()
				void launchAfterJoinCheck()
			}}>
			<div class="p-3">
				<div
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
									>Server ID <small class="text-xs text-fg-faint"
										>(optional)</small
									></span>
								<input
									class="field font-mono text-2xs"
									bind:value={store.launchInput.serverId}
									type="text"
									placeholder="Server ID"
									disabled={store.launching} />
							</label>
							{#if appSettings.roValraEnabled}
								<NearestServerButton {store} {notifications} />
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
								disabled={store.launching} />
						</label>
					{:else}
						<label class="min-w-0 flex-1"
							><span class="mb-1 block text-xs text-fg-muted"
								>Username / User ID</span>
							<input
								class="field font-mono text-2xs"
								data-launch-user
								bind:value={store.launchInput.user}
								type="text"
								placeholder="Username or id:123"
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
						data-tooltip="Clear join fields"
						data-tooltip-side="top"
						disabled={store.launching}
						onclick={() => store.clearJoinTarget()}>
						<Icon svg={eraserIcon} size={16} aria-hidden="true" />
					</button>
				</div>
				<LaunchArguments {store} />
				<LaunchButton {store} {workspace} {isCheckingJoin} />
			</div>
		</form>
	{/if}
	{#if joinRestrictions.length > 0}
		<LaunchJoinWarningDialog
			restrictions={joinRestrictions}
			{store}
			onLaunchAnyway={launchAnyway}
			onCancel={() => (joinRestrictions = [])} />
	{/if}
	{#if processesOpen}
		<RobloxProcessesDialog
			onClose={() => (processesOpen = false)}
			onClean={() => {
				processesOpen = false
				cleanOpen = true
			}} />
	{/if}
	{#if cleanOpen}
		<RobloxCleanDialog
			scope={RobloxCleanScope.ScopeData}
			{notifications}
			onClose={() => (cleanOpen = false)} />
	{/if}
</section>

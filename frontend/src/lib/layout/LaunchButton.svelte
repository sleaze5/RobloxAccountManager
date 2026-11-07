<script lang="ts">
	import checkIcon from "@phosphor-icons/core/regular/check.svg?raw"
	import circleNotchIcon from "@phosphor-icons/core/regular/circle-notch.svg?raw"
	import playFillIcon from "@phosphor-icons/core/fill/play-fill.svg?raw"
	import Icon from "../shared/Icon.svelte"
	import { onDestroy, tick } from "svelte"
	import type { AccountStore } from "../accounts/account-store.svelte"
	import { menuIn, menuOut, placeMenuAtPoint } from "../shared/presence"
	import type { WorkspaceState } from "./workspace-state.svelte"

	let { store, workspace }: { store: AccountStore; workspace: WorkspaceState } =
		$props()
	let root = $state<HTMLDivElement>(),
		trigger = $state<HTMLButtonElement>(),
		menu = $state<HTMLDivElement>(),
		point = $state<{ x: number; y: number } | null>(null),
		copied = $state(false)
	let copiedTimer = 0
	const disabled = $derived(
		store.selectedAccount.id === 0 || store.launching || !store.vault.unlocked,
	)

	$effect(() => {
		if (disabled) {
			point = null
			copied = false
		}
	})
	onDestroy(() => window.clearTimeout(copiedTimer))

	async function showMenu(x: number, y: number): Promise<void> {
		if (disabled) return
		workspace.closeAccountMenus()
		point = { x, y }
		await tick()
		menu?.querySelector<HTMLButtonElement>("button")?.focus()
	}

	function positionMenu(node: HTMLElement, position: { x: number; y: number }) {
		const place = (next: { x: number; y: number }) => {
			placeMenuAtPoint(node, next)
		}
		place(position)
		return { update: place }
	}

	function closeMenu(): void {
		point = null
		trigger?.focus()
	}

	async function copyOptions(): Promise<void> {
		point = null
		if (!(await store.copyLaunchOptions()) || !root?.isConnected) return
		copied = true
		trigger?.focus()
		window.clearTimeout(copiedTimer)
		copiedTimer = window.setTimeout(() => (copied = false), 2000)
	}
</script>

<svelte:window
	onresize={() => (point = null)}
	onclick={(event) => {
		if (event.target instanceof Node && !root?.contains(event.target)) point = null
	}}
	oncontextmenu={(event) => {
		if (event.target instanceof Node && !root?.contains(event.target)) point = null
	}} />

<div bind:this={root}>
	<!-- Copy launch options is a developer feature. It is intentionally reachable only from this context menu, with no visible control. -->
	<button
		class="button-primary h-8 min-w-24"
		type="submit"
		bind:this={trigger}
		{disabled}
		aria-keyshortcuts="Shift+F10"
		oncontextmenu={(event) => {
			event.preventDefault()
			event.stopPropagation()
			void showMenu(event.clientX, event.clientY)
		}}
		onkeydown={(event) => {
			if (
				event.key === "ContextMenu" ||
				(event.shiftKey && event.key === "F10")
			) {
				event.preventDefault()
				const bounds = event.currentTarget.getBoundingClientRect()
				void showMenu(bounds.left, bounds.bottom)
			}
		}}>
		{#if store.launching}<Icon
				svg={circleNotchIcon}
				class="spinner"
				size={16}
				aria-hidden="true" />
		{:else if copied}<Icon
				svg={checkIcon}
				class="animate-bloom"
				size={16}
				aria-hidden="true" />
		{:else}<Icon svg={playFillIcon} size={16} aria-hidden="true" />{/if}
		{store.copyingLaunchOptions
			? "Copying"
			: store.launching
				? "Launching"
				: copied
					? "Copied"
					: store.multipleSelected
						? `Launch (${store.selectedAccountIds.length})`
						: "Launch"}
	</button>
	{#if point}
		<div
			class="fixed menu"
			data-account-menu
			bind:this={menu}
			role="menu"
			tabindex="-1"
			aria-label="Launch actions"
			use:positionMenu={point}
			onkeydown={(event) => {
				if (event.key === "Escape" || event.key === "Tab") {
					if (event.key === "Escape") event.preventDefault()
					event.stopPropagation()
					closeMenu()
				} else if (
					["ArrowDown", "ArrowUp", "Home", "End"].includes(event.key)
				) {
					event.preventDefault()
					menu?.querySelector<HTMLButtonElement>("button")?.focus()
				}
			}}
			in:menuIn
			out:menuOut>
			<button
				class="menu-item"
				type="button"
				role="menuitem"
				tabindex="-1"
				{disabled}
				onclick={() => void copyOptions()}>
				<span>Copy launch options</span>
			</button>
		</div>
	{/if}
</div>

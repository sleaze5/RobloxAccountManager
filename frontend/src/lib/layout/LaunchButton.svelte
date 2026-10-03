<script lang="ts">
	import Check from "@lucide/svelte/icons/check"
	import LoaderCircle from "@lucide/svelte/icons/loader-circle"
	import Play from "@lucide/svelte/icons/play"
	import { onDestroy, tick } from "svelte"
	import type { AccountStore } from "../accounts/account-store.svelte"
	import { menuIn, menuOut } from "../shared/presence"
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
		),
		menuID = $props.id()

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
			const x = Math.max(
					8,
					Math.min(next.x, window.innerWidth - node.offsetWidth - 8),
				),
				y = Math.max(
					42,
					Math.min(next.y, window.innerHeight - node.offsetHeight - 8),
				)
			node.style.left = `${x}px`
			node.style.top = `${y}px`
			node.style.setProperty(
				"--menu-origin",
				`${y < next.y ? "bottom" : "top"} ${x < next.x ? "right" : "left"}`,
			)
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
	<button
		class="launch-control"
		type="submit"
		bind:this={trigger}
		{disabled}
		aria-haspopup="menu"
		aria-expanded={point !== null}
		aria-controls={point ? menuID : undefined}
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
		{#if store.launching}<LoaderCircle
				class="spinner"
				size={14}
				aria-hidden="true" />
		{:else if copied}<Check size={14} aria-hidden="true" />
		{:else}<Play size={14} fill="currentColor" aria-hidden="true" />{/if}
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
			class="account-context-menu account-menu"
			data-account-menu
			bind:this={menu}
			id={menuID}
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

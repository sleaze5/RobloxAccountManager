<script lang="ts">
	import type {
		AccountContextMenu,
		WorkspaceState,
	} from "../layout/workspace-state.svelte"
	import { menuIn, menuOut } from "../shared/presence"
	import AccountActionsMenu from "./AccountActionsMenu.svelte"
	import MultiAccountActionsMenu from "./MultiAccountActionsMenu.svelte"
	import type { AccountStore } from "./account-store.svelte"

	let {
		store,
		workspace,
		menu,
		onManageTags,
		onRemove,
		onRenewCookie,
		onBrowser,
	}: {
		store: AccountStore
		workspace: WorkspaceState
		menu: AccountContextMenu
		onManageTags: (accountId: number | null) => void
		onRemove: (accountId: number) => void
		onRenewCookie: (accountId: number) => void
		onBrowser: (accountId: number | null) => void
	} = $props()

	function positionMenu(node: HTMLElement, position: AccountContextMenu) {
		const previousFocus = document.activeElement
		let current = position
		const place = () => {
			// Keep the menu 8 px inside the window and at least 42 px from the top, below the title bar.
			const width = node.offsetWidth,
				height = node.offsetHeight,
				x = Math.max(8, Math.min(current.x, window.innerWidth - width - 8)),
				y = Math.max(42, Math.min(current.y, window.innerHeight - height - 8))
			node.style.left = `${x}px`
			node.style.top = `${y}px`
			node.style.setProperty(
				"--menu-origin",
				`${y < current.y ? "bottom" : "top"} ${x < current.x ? "right" : "left"}`,
			)
			current.submenuLeft = x + width * 2 + 12 > window.innerWidth
		}
		place()
		node.focus({ preventScroll: true })
		window.addEventListener("resize", place)
		return {
			update(next: AccountContextMenu) {
				current = next
				place()
			},
			destroy() {
				window.removeEventListener("resize", place)
				if (
					node.contains(document.activeElement) &&
					previousFocus instanceof HTMLElement &&
					previousFocus.isConnected
				) {
					previousFocus.focus({ preventScroll: true })
				}
			},
		}
	}
</script>

<div
	class="account-context-menu account-menu"
	role="group"
	aria-label={store.multipleSelected ? "Selected account actions" : "Account actions"}
	tabindex="-1"
	data-account-menu
	use:positionMenu={menu}
	in:menuIn
	out:menuOut>
	{#if store.multipleSelected}
		<MultiAccountActionsMenu
			{store}
			{workspace}
			submenuLeft={menu.submenuLeft}
			onManageTags={() => onManageTags(null)} />
	{:else}
		<AccountActionsMenu
			{store}
			{workspace}
			account={store.selectedAccount}
			submenuLeft={menu.submenuLeft}
			{onManageTags}
			{onRemove}
			{onRenewCookie}
			{onBrowser} />
	{/if}
</div>

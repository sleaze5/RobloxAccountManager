<script lang="ts">
	import type {
		AccountContextMenu,
		WorkspaceState,
	} from "../layout/workspace-state.svelte"
	import { menuIn, menuKeyboard, menuOut, placeMenuAtPoint } from "../shared/presence"
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
		let current = position
		const place = () => {
			const { x } = placeMenuAtPoint(node, current)
			current.submenuLeft = x + node.offsetWidth * 2 + 12 > window.innerWidth
		}
		place()
		window.addEventListener("resize", place)
		return {
			update(next: AccountContextMenu) {
				current = next
				place()
			},
			destroy() {
				window.removeEventListener("resize", place)
			},
		}
	}
</script>

<div
	class="fixed menu"
	role="group"
	aria-label={store.multipleSelected ? "Selected account actions" : "Account actions"}
	tabindex="-1"
	data-account-menu
	use:positionMenu={menu}
	use:menuKeyboard
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

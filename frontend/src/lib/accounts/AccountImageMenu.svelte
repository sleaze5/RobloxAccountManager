<script lang="ts">
	import type {
		ImageContextMenu,
		WorkspaceState,
	} from "../layout/workspace-state.svelte"
	import { floatMenuAtPoint, menuIn, menuKeyboard, menuOut } from "../shared/presence"
	import type { AccountStore } from "./account-store.svelte"

	let {
		store,
		workspace,
		menu,
	}: {
		store: AccountStore
		workspace: WorkspaceState
		menu: ImageContextMenu
	} = $props()
	let copyingImage = $state(false)

	async function copy(urlOnly: boolean): Promise<void> {
		copyingImage = !urlOnly
		if (await store.copyAccountImage(menu.imageUrl, urlOnly)) {
			workspace.closeAccountMenus()
		}
		copyingImage = false
	}
</script>

<div
	class="fixed menu"
	data-account-menu
	role="menu"
	aria-label="Image actions"
	tabindex="-1"
	use:floatMenuAtPoint={menu}
	use:menuKeyboard
	in:menuIn
	out:menuOut>
	<button
		class="menu-item"
		type="button"
		role="menuitem"
		disabled={store.copying}
		onclick={() => void copy(false)}>
		<span>{copyingImage ? "Copying image…" : "Copy image"}</span>
	</button>
	<button
		class="menu-item"
		type="button"
		role="menuitem"
		disabled={store.copying}
		onclick={() => void copy(true)}>
		<span>Copy image URL</span>
	</button>
</div>

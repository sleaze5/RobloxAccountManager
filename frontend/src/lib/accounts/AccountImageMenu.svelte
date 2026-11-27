<script lang="ts">
	import type {
		ImageContextMenu,
		WorkspaceState,
	} from "../layout/workspace-state.svelte"
	import copyIcon from "@phosphor-icons/core/regular/copy.svg?raw"
	import imageIcon from "@phosphor-icons/core/regular/image.svg?raw"
	import linkIcon from "@phosphor-icons/core/regular/link.svg?raw"
	import MenuItem from "../shared/MenuItem.svelte"
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
		if (await store.copyAccountImage(menu.image.imageUrl, urlOnly)) {
			workspace.closeAccountMenus()
		}
		copyingImage = false
	}

	function expandImage(): void {
		const { image } = menu
		workspace.closeAccountMenus()
		workspace.viewedImage = image
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
	<MenuItem label="Expand image" icon={imageIcon} onSelect={expandImage} />
	<MenuItem
		label={copyingImage ? "Copying image…" : "Copy image"}
		icon={copyIcon}
		disabled={store.copying}
		onSelect={() => void copy(false)} />
	<MenuItem
		label="Copy image URL"
		icon={linkIcon}
		disabled={store.copying}
		onSelect={() => void copy(true)} />
</div>

<script lang="ts">
	import type {
		ImageContextMenu,
		WorkspaceState,
	} from "../layout/workspace-state.svelte"
	import { menuIn, menuOut, placeMenuAtPoint } from "../shared/presence"
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

	function placeMenu(node: HTMLElement, position: ImageContextMenu) {
		const previousFocus = document.activeElement
		let current = position
		const place = () => {
			placeMenuAtPoint(node, current)
		}
		place()
		node.querySelector<HTMLButtonElement>("button")?.focus({ preventScroll: true })
		window.addEventListener("resize", place)
		return {
			update(next: ImageContextMenu) {
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

	function handleKeydown(event: KeyboardEvent): void {
		if (!["ArrowDown", "ArrowUp", "Home", "End"].includes(event.key)) return
		event.preventDefault()
		const node = event.currentTarget as HTMLElement,
			buttons = Array.from(
				node.querySelectorAll<HTMLButtonElement>("button:not(:disabled)"),
			),
			current = buttons.findIndex((button) => button === document.activeElement),
			direction = event.key === "ArrowUp" ? -1 : 1,
			next =
				event.key === "Home"
					? 0
					: event.key === "End"
						? buttons.length - 1
						: (current + direction + buttons.length) % buttons.length
		buttons[next]?.focus()
	}

	async function copy(urlOnly: boolean): Promise<void> {
		copyingImage = !urlOnly
		if (await store.copyAccountImage(menu.imageUrl, urlOnly)) {
			workspace.closeAccountMenus()
		}
		copyingImage = false
	}
</script>

<div
	class="account-context-menu account-menu"
	data-account-menu
	role="menu"
	aria-label="Image actions"
	tabindex="-1"
	onkeydown={handleKeydown}
	use:placeMenu={menu}
	in:menuIn
	out:menuOut>
	<button
		type="button"
		role="menuitem"
		disabled={store.copying}
		onclick={() => void copy(false)}>
		<span>{copyingImage ? "Copying image…" : "Copy image"}</span>
	</button>
	<button
		type="button"
		role="menuitem"
		disabled={store.copying}
		onclick={() => void copy(true)}>
		<span>Copy image URL</span>
	</button>
</div>

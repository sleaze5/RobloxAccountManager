<script lang="ts">
	import ListPlusIcon from "phosphor-svelte/lib/ListPlusIcon"
	import { menuIn, menuOut, placeMenuAtAnchor } from "../shared/presence"

	let {
		id,
		anchor,
		onClose,
		onFillLaunch,
	}: {
		id: string
		anchor: HTMLElement
		onClose: () => void
		onFillLaunch: () => void
	} = $props()
	let menu = $state<HTMLElement>()
	let restoreFocus = false

	function positionMenu(node: HTMLElement, trigger: HTMLElement) {
		const place = (current: HTMLElement) => {
			restoreFocus = false
			placeMenuAtAnchor(node, current)
			node.querySelector<HTMLButtonElement>("button")?.focus({
				preventScroll: true,
			})
		}
		place(trigger)
		return {
			update: place,
			destroy() {
				if (
					anchor.isConnected &&
					(node.contains(document.activeElement) ||
						(restoreFocus && document.activeElement === document.body))
				) {
					anchor.focus({ preventScroll: true })
				}
			},
		}
	}

	function closeOutside(event: Event): void {
		if (
			event.target instanceof Node &&
			!menu?.contains(event.target) &&
			!anchor.contains(event.target)
		) {
			onClose()
		}
	}

	function handleKeydown(event: KeyboardEvent): void {
		if (event.key === "Escape") {
			event.preventDefault()
			event.stopPropagation()
			restoreFocus = true
			onClose()
		} else if (event.key === "Tab") {
			onClose()
		} else if (["ArrowDown", "ArrowUp", "Home", "End"].includes(event.key)) {
			event.preventDefault()
			menu?.querySelector<HTMLButtonElement>("button")?.focus()
		}
	}
</script>

<svelte:window
	onpointerdown={closeOutside}
	onscrollcapture={closeOutside}
	onresize={onClose} />

<div
	class="account-context-menu account-menu"
	bind:this={menu}
	{id}
	role="menu"
	aria-label="Event actions"
	tabindex="-1"
	onkeydown={handleKeydown}
	use:positionMenu={anchor}
	in:menuIn|global
	out:menuOut|global>
	<button type="button" role="menuitem" onclick={onFillLaunch}>
		<ListPlusIcon size={16} aria-hidden="true" />
		<span>Fill launch options</span>
	</button>
</div>

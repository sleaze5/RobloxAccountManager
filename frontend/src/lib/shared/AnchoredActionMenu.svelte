<script lang="ts" module>
	export interface MenuAction {
		label: string
		icon: string
		danger?: boolean
		onSelect: () => void
	}
</script>

<script lang="ts">
	import Icon from "./Icon.svelte"
	import { focusMenuItemForKey, menuIn, menuOut, placeMenuAtAnchor } from "./presence"

	let {
		id,
		label,
		anchor,
		actions,
		onClose,
	}: {
		id: string
		label: string
		anchor: HTMLElement
		actions: MenuAction[]
		onClose: () => void
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
		} else {
			focusMenuItemForKey(
				event,
				Array.from(menu?.querySelectorAll<HTMLButtonElement>("button") ?? []),
			)
		}
	}
</script>

<svelte:window
	onpointerdown={closeOutside}
	onscrollcapture={closeOutside}
	onresize={onClose} />

<div
	class="fixed menu"
	bind:this={menu}
	{id}
	role="menu"
	aria-label={label}
	tabindex="-1"
	onkeydown={handleKeydown}
	use:positionMenu={anchor}
	in:menuIn|global
	out:menuOut|global>
	{#each actions as action (action.label)}
		{#if action.danger}<div class="menu-separator"></div>{/if}
		<button
			class={action.danger ? "menu-item-danger" : "menu-item"}
			type="button"
			role="menuitem"
			onclick={action.onSelect}>
			<Icon svg={action.icon} size={16} aria-hidden="true" />
			<span>{action.label}</span>
		</button>
	{/each}
</div>

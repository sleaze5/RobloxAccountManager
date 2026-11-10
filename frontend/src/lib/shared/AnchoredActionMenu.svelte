<script lang="ts" module>
	export interface ServerMenuAction {
		label: string
		icon: string
		danger?: boolean
		onSelect: () => void
	}
</script>

<script lang="ts">
	import Icon from "../shared/Icon.svelte"
	import { menuIn, menuOut, placeMenuAtAnchor } from "../shared/presence"

	let {
		id,
		anchor,
		actions,
		onClose,
	}: {
		id: string
		anchor: HTMLElement
		actions: ServerMenuAction[]
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
		const buttons = Array.from(
				menu?.querySelectorAll<HTMLButtonElement>("button") ?? [],
			),
			current = buttons.indexOf(document.activeElement as HTMLButtonElement)
		let next: number
		switch (event.key) {
			case "Escape":
				event.preventDefault()
				event.stopPropagation()
				restoreFocus = true
				onClose()
				return
			case "Tab":
				onClose()
				return
			case "ArrowDown":
				next = (current + 1) % buttons.length
				break
			case "ArrowUp":
				next = (current - 1 + buttons.length) % buttons.length
				break
			case "Home":
				next = 0
				break
			case "End":
				next = buttons.length - 1
				break
			default:
				return
		}
		event.preventDefault()
		buttons[next]?.focus()
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
	aria-label="Server actions"
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

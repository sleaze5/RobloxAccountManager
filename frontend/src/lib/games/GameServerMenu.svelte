<script lang="ts">
	import HashIcon from "phosphor-svelte/lib/HashIcon"
	import ListPlusIcon from "phosphor-svelte/lib/ListPlusIcon"
	import { menuIn, menuOut } from "../shared/presence"

	let {
		id,
		anchor,
		onClose,
		onFillLaunch,
		onCopyJobId,
	}: {
		id: string
		anchor: HTMLElement
		onClose: () => void
		onFillLaunch: () => void
		onCopyJobId: () => void
	} = $props()
	let menu = $state<HTMLElement>()
	let restoreFocus = false

	function positionMenu(node: HTMLElement, trigger: HTMLElement) {
		const place = (current: HTMLElement) => {
			restoreFocus = false
			// Keep the menu 8 px inside the window and at least 42 px from the top, below the title bar.
			const bounds = current.getBoundingClientRect(),
				opensUp =
					bounds.bottom + 4 + node.offsetHeight > window.innerHeight - 8,
				x = Math.max(
					8,
					Math.min(
						bounds.right - node.offsetWidth,
						window.innerWidth - node.offsetWidth - 8,
					),
				),
				y = Math.max(
					42,
					opensUp ? bounds.top - node.offsetHeight - 4 : bounds.bottom + 4,
				)
			node.style.left = `${x}px`
			node.style.top = `${y}px`
			node.style.setProperty(
				"--menu-origin",
				`${opensUp ? "bottom" : "top"} right`,
			)
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
	class="account-context-menu account-menu game-server-menu"
	bind:this={menu}
	{id}
	role="menu"
	aria-label="Server actions"
	tabindex="-1"
	onkeydown={handleKeydown}
	use:positionMenu={anchor}
	in:menuIn|global
	out:menuOut|global>
	<button type="button" role="menuitem" onclick={onFillLaunch}>
		<ListPlusIcon size={16} aria-hidden="true" />
		<span>Fill launch options</span>
	</button>
	<button type="button" role="menuitem" onclick={onCopyJobId}>
		<HashIcon size={16} aria-hidden="true" />
		<span>Copy job ID</span>
	</button>
</div>

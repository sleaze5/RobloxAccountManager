<script lang="ts">
	import FloppyDiskIcon from "phosphor-svelte/lib/FloppyDiskIcon"
	import KeyIcon from "phosphor-svelte/lib/KeyIcon"
	import PencilSimpleIcon from "phosphor-svelte/lib/PencilSimpleIcon"
	import { menuIn, menuOut } from "../shared/presence"

	let {
		hasSavedPassword,
		onEditSaved,
		onChange,
	}: {
		hasSavedPassword: boolean
		onEditSaved: () => void
		onChange: () => void
	} = $props()
	let open = $state(false),
		trigger = $state<HTMLButtonElement>(),
		menu = $state<HTMLDivElement>()
	const menuId = $props.id()

	function placeMenu(node: HTMLElement): void {
		const bounds = trigger?.getBoundingClientRect()
		if (!bounds) return
		const opensUp = bounds.bottom + 4 + node.offsetHeight > window.innerHeight - 8,
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
		node.style.setProperty("--menu-origin", `${opensUp ? "bottom" : "top"} right`)
		node.querySelector<HTMLButtonElement>("button")?.focus({ preventScroll: true })
	}

	function close(restoreFocus = false): void {
		open = false
		if (restoreFocus) trigger?.focus({ preventScroll: true })
	}

	function choose(action: () => void): void {
		close(true)
		action()
	}

	function closeOutside(event: Event): void {
		if (
			open &&
			event.target instanceof Node &&
			!menu?.contains(event.target) &&
			!trigger?.contains(event.target)
		) {
			close()
		}
	}

	function handleKeydown(event: KeyboardEvent): void {
		if (event.key === "Escape") {
			event.preventDefault()
			event.stopPropagation()
			close(true)
			return
		}
		if (event.key === "Tab") {
			close()
			return
		}
		if (!["ArrowDown", "ArrowUp", "Home", "End"].includes(event.key)) return
		event.preventDefault()
		const buttons = Array.from(
				menu?.querySelectorAll<HTMLButtonElement>("button") ?? [],
			),
			current = buttons.indexOf(document.activeElement as HTMLButtonElement),
			direction = event.key === "ArrowUp" ? -1 : 1,
			next =
				event.key === "Home"
					? 0
					: event.key === "End"
						? buttons.length - 1
						: (current + direction + buttons.length) % buttons.length
		buttons[next]?.focus()
	}
</script>

<svelte:window
	onpointerdown={closeOutside}
	onscrollcapture={closeOutside}
	onresize={() => close()} />

<button
	class="icon-action profile-fact-action"
	type="button"
	bind:this={trigger}
	aria-label="Password options"
	data-tooltip="Password options"
	aria-haspopup="menu"
	aria-expanded={open}
	aria-controls={open ? menuId : undefined}
	onclick={() => (open = !open)}
	onkeydown={(event) => {
		if (event.key === "ArrowDown" || event.key === "ArrowUp") {
			event.preventDefault()
			open = true
		}
	}}>
	<PencilSimpleIcon size={14} aria-hidden="true" />
</button>

{#if open}
	<div
		class="account-context-menu account-menu"
		bind:this={menu}
		id={menuId}
		role="menu"
		aria-label="Password options"
		tabindex="-1"
		onkeydown={handleKeydown}
		use:placeMenu
		in:menuIn
		out:menuOut>
		<button
			type="button"
			role="menuitem"
			tabindex="-1"
			onclick={() => choose(onEditSaved)}>
			<FloppyDiskIcon size={16} aria-hidden="true" />
			<span>{hasSavedPassword ? "Update saved password" : "Save password"}</span>
		</button>
		<button
			type="button"
			role="menuitem"
			tabindex="-1"
			onclick={() => choose(onChange)}>
			<KeyIcon size={16} aria-hidden="true" />
			<span>Change Roblox password</span>
		</button>
	</div>
{/if}

<script lang="ts" module>
	import type { MenuItemTone } from "./menu-tone"

	// MenuAction is one item of a menu. Consecutive actions with the same
	// section show under its heading. A danger action comes last, after a
	// separator.
	export interface MenuAction {
		label: string
		icon: string
		tone?: MenuItemTone
		section?: string
		detail?: string
		accessibleLabel?: string
		tooltip?: string
		disabled?: boolean
		onSelect: () => void
	}

	interface MenuActionGroup {
		section: string
		isDanger: boolean
		actions: MenuAction[]
	}

	function groupMenuActions(actions: MenuAction[]): MenuActionGroup[] {
		const groups: MenuActionGroup[] = []
		for (const action of actions) {
			const section = action.section ?? "",
				isDanger = action.tone === "danger",
				previous = groups.at(-1)
			if (previous?.section === section && previous.isDanger === isDanger) {
				previous.actions.push(action)
			} else groups.push({ section, isDanger, actions: [action] })
		}
		return groups
	}
</script>

<script lang="ts">
	import {
		focusMenuItemForKey,
		menuIn,
		menuOut,
		placeMenuAtAnchor,
	} from "./transitions"
	import MenuItem from "./MenuItem.svelte"
	import MenuSection from "./MenuSection.svelte"
	import MenuTitle from "./MenuTitle.svelte"

	// selection heads the menu with the count of the items that the actions
	// change, such as the selected rows of a list.
	let {
		id,
		label,
		anchor,
		actions,
		selection,
		onClose,
	}: {
		id: string
		label: string
		anchor: HTMLElement
		actions: MenuAction[]
		selection?: { count: number; noun: string }
		onClose: () => void
	} = $props()
	let menu = $state<HTMLElement>()
	let restoreFocus = false
	const actionGroups = $derived(groupMenuActions(actions))

	function enabledItems(node: HTMLElement | undefined): HTMLButtonElement[] {
		return Array.from(
			node?.querySelectorAll<HTMLButtonElement>("button:not(:disabled)") ?? [],
		)
	}

	function positionMenu(node: HTMLElement, trigger: HTMLElement) {
		const place = (current: HTMLElement) => {
			restoreFocus = false
			placeMenuAtAnchor(node, current)
			;(enabledItems(node)[0] ?? node).focus({ preventScroll: true })
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
			focusMenuItemForKey(event, enabledItems(menu))
		}
	}
</script>

<svelte:window
	onpointerdown={closeOutside}
	onscrollcapture={closeOutside}
	onresize={onClose} />

{#snippet item(action: MenuAction)}
	<MenuItem
		label={action.label}
		icon={action.icon}
		tone={action.tone}
		detail={action.detail}
		accessibleLabel={action.accessibleLabel}
		tooltip={action.tooltip}
		disabled={action.disabled}
		onSelect={action.onSelect} />
{/snippet}

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
	{#if selection}<MenuTitle count={selection.count} noun={selection.noun} />{/if}
	{#each actionGroups as group, index (index)}
		{#if group.isDanger && index > 0}<div class="menu-separator"></div>{/if}
		{#if group.section}
			<MenuSection heading={group.section}>
				{#each group.actions as action (action.label)}{@render item(
						action,
					)}{/each}
			</MenuSection>
		{:else}
			{#each group.actions as action (action.label)}{@render item(action)}{/each}
		{/if}
	{/each}
</div>

<script lang="ts" generics="T extends string | number">
	import CheckIcon from "phosphor-svelte/lib/CheckIcon"
	import CaretDownIcon from "phosphor-svelte/lib/CaretDownIcon"
	import { tick } from "svelte"
	import { menuIn, menuOut } from "./presence"

	let {
		label,
		value,
		options,
		disabled = false,
		onChange,
	}: {
		label: string
		value: T
		options: { value: T; label: string; description?: string; group?: string }[]
		disabled?: boolean
		onChange: (value: T) => void
	} = $props()
	let open = $state(false),
		root = $state<HTMLDivElement>(),
		menu = $state<HTMLDivElement>(),
		trigger = $state<HTMLButtonElement>()
	const maxMenuHeight = 280
	let menuPosition = $state("")
	const menuID = $props.id()
	const selectedLabel = $derived(
		options.find((option) => option.value === value)?.label ?? "",
	)

	$effect(() => {
		if (disabled) open = false
	})

	async function show(): Promise<void> {
		const bounds = trigger?.getBoundingClientRect()
		if (!bounds) return
		const width = Math.min(
			Math.max(
				bounds.width,
				options.some((option) => option.description) ? 320 : 0,
			),
			window.innerWidth - 24,
		)
		menuPosition = `position: fixed; visibility: hidden; width: ${width}px;`
		open = true
		await tick()
		if (!open || !menu) return
		const below = window.innerHeight - bounds.bottom - 12,
			above = bounds.top - 12,
			height = Math.min(menu.getBoundingClientRect().height, maxMenuHeight),
			opensUp = below < Math.min(height, above),
			left = Math.max(12, Math.min(bounds.left, window.innerWidth - width - 12)),
			edge = opensUp
				? `bottom: ${window.innerHeight - bounds.top + 4}px; top: auto;`
				: `top: ${bounds.bottom + 4}px;`
		menuPosition = `position: fixed; ${edge} left: ${left}px; right: auto; width: ${width}px; --select-menu-height: ${Math.max(28, Math.min(maxMenuHeight, opensUp ? above : below))}px; --menu-origin: ${opensUp ? "bottom" : "top"} left;`
		await tick()
		root?.querySelector<HTMLButtonElement>("[aria-selected='true']")?.focus()
	}

	function close(restoreFocus = false): void {
		open = false
		if (restoreFocus) trigger?.focus()
	}

	function handleKeydown(event: KeyboardEvent): void {
		if (event.key === "Escape") {
			event.preventDefault()
			event.stopPropagation()
			close(true)
			return
		}
		if (event.key === "Tab") {
			close(true)
			return
		}
		const buttons = Array.from(
			root?.querySelectorAll<HTMLButtonElement>("[role='option']") ?? [],
		)
		const current = buttons.indexOf(document.activeElement as HTMLButtonElement)
		let next: number
		switch (event.key) {
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
	onresize={() => close()}
	onscrollcapture={(event) => {
		if (open && event.target instanceof Node && !root?.contains(event.target))
			close()
	}}
	onclick={(event) => {
		if (open && event.target instanceof Node && !root?.contains(event.target))
			close()
	}} />

<div class="settings-select" bind:this={root}>
	<button
		class="settings-select-trigger"
		type="button"
		bind:this={trigger}
		aria-label={label}
		aria-haspopup="listbox"
		aria-expanded={open}
		aria-controls={open ? menuID : undefined}
		{disabled}
		onclick={() => {
			if (open) close()
			else void show()
		}}
		onkeydown={(event) => {
			if (event.key === "ArrowDown" || event.key === "ArrowUp") {
				event.preventDefault()
				void show()
			}
		}}>
		<span>{selectedLabel}</span><CaretDownIcon
			class={open ? "open" : undefined}
			size={16}
			aria-hidden="true" />
	</button>
	{#if open}
		<div
			class="settings-select-menu"
			bind:this={menu}
			style={menuPosition}
			id={menuID}
			role="listbox"
			tabindex="-1"
			aria-label={label}
			onkeydown={handleKeydown}
			in:menuIn
			out:menuOut>
			<div class="settings-select-options">
				{#each options as option, index (option.value)}
					{#if option.group && option.group !== options[index - 1]?.group}
						<div class="settings-select-group" aria-hidden="true">
							{option.group}
						</div>
					{/if}
					<button
						class:has-description={!!option.description}
						type="button"
						role="option"
						aria-label={option.group
							? `${option.label}, ${option.group}`
							: option.label}
						aria-describedby={option.description
							? `${menuID}-${index}-description`
							: undefined}
						aria-selected={option.value === value}
						tabindex={option.value === value ? 0 : -1}
						onclick={() => {
							onChange(option.value)
							close(true)
						}}>
						<span class="settings-select-option-copy">
							<span>{option.label}</span>
							{#if option.description}<small
									id={`${menuID}-${index}-description`}
									>{option.description}</small
								>{/if}
						</span>{#if option.value === value}<CheckIcon
								size={15}
								aria-hidden="true" />{/if}
					</button>
				{/each}
			</div>
		</div>
	{/if}
</div>

<script lang="ts">
	import minusIcon from "@phosphor-icons/core/regular/minus.svg?raw"
	import plusIcon from "@phosphor-icons/core/regular/plus.svg?raw"
	import Icon from "./Icon.svelte"
	import { tick } from "svelte"

	let {
		label,
		value,
		min,
		max,
		step,
		defaultValue,
		disabled = false,
		onChange,
	}: {
		label: string
		value: number
		min: number
		max: number
		step: number
		defaultValue: number
		disabled?: boolean
		onChange: (value: number) => Promise<void>
	} = $props()
	let input = $state<HTMLInputElement>(),
		draft = $state<number | undefined>(),
		saving = $state(false)

	$effect(() => {
		draft = value
	})

	function normalized(inputValue: number): number {
		if (!Number.isFinite(inputValue)) return defaultValue
		return Math.min(
			max,
			Math.max(min, min + Math.round((inputValue - min) / step) * step),
		)
	}

	async function save(): Promise<void> {
		if (disabled || saving || !input) return
		const next = normalized(input.valueAsNumber)
		input.value = String(next)
		draft = next
		if (next === value) return
		saving = true
		try {
			await onChange(next)
		} finally {
			await tick()
			draft = value
			saving = false
		}
	}

	function adjust(direction: number): void {
		if (disabled || saving || !input) return
		draft = Math.min(
			max,
			Math.max(min, normalized(input.valueAsNumber) + direction * step),
		)
		input.value = String(draft)
		void save()
	}
</script>

<div
	class="flex items-center gap-1"
	role="group"
	aria-label={label}
	aria-busy={saving}
	onfocusout={(event) => {
		if (
			!(event.relatedTarget instanceof Node) ||
			!event.currentTarget.contains(event.relatedTarget)
		)
			void save()
	}}>
	<input
		bind:this={input}
		bind:value={draft}
		class="field w-16 text-right font-mono text-xs"
		type="number"
		inputmode="numeric"
		aria-label={label}
		{min}
		{max}
		{step}
		disabled={disabled || saving}
		onkeydown={(event) => {
			if (event.key === "Enter") {
				event.preventDefault()
				void save()
			}
		}} />
	<button
		class="button-icon size-8"
		type="button"
		aria-label={`Decrease ${label.toLowerCase()}`}
		disabled={disabled || saving || normalized(draft ?? value) <= min}
		onclick={() => adjust(-1)}>
		<Icon svg={minusIcon} size={16} aria-hidden="true" />
	</button>
	<button
		class="button-icon size-8"
		type="button"
		aria-label={`Increase ${label.toLowerCase()}`}
		disabled={disabled || saving || normalized(draft ?? value) >= max}
		onclick={() => adjust(1)}>
		<Icon svg={plusIcon} size={16} aria-hidden="true" />
	</button>
</div>

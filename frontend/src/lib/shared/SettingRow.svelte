<script lang="ts" module>
	export type SettingRadioOption = {
		label: string
		value: string
		disabled?: boolean
		reason?: string
	}
</script>

<script lang="ts">
	import circleNotchIcon from "@phosphor-icons/core/regular/circle-notch.svg?raw"
	import Icon from "./Icon.svelte"

	// SettingRow shows one setting as a switch or as a list of options, with
	// its label, description, and notes. The controls show the saved value
	// until the page saves a new one, so a canceled change does not stay. A
	// page of settings marks each row with data-setting, so that it can keep
	// the scroll position of a row.
	let {
		id,
		label,
		description,
		type,
		isChecked = false,
		radioValue = "",
		radioOptions = [],
		notes = [],
		currentValueLabel = "",
		disabled = false,
		isLoading = false,
		saving = false,
		onToggle,
		onRadioSelect,
	}: {
		id: string
		label: string
		description: string
		type: "toggle" | "radio"
		isChecked?: boolean
		radioValue?: string
		radioOptions?: SettingRadioOption[]
		// notes explain why the setting or one of its options is unavailable.
		notes?: string[]
		// currentValueLabel names a value that the switch cannot show.
		currentValueLabel?: string
		disabled?: boolean
		isLoading?: boolean
		saving?: boolean
		onToggle?: (checked: boolean) => void
		onRadioSelect?: (value: string) => void
	} = $props()

	const shownNotes = $derived(isLoading ? [] : notes),
		describedBy = $derived(
			shownNotes.length > 0
				? `setting-description-${id} setting-notes-${id}`
				: `setting-description-${id}`,
		),
		controlDisabled = $derived(disabled || saving)
</script>

{#snippet details()}
	<strong id={`setting-label-${id}`}>{label}</strong>
	<span id={`setting-description-${id}`}>{description}</span>
	{#if shownNotes.length > 0}
		<span id={`setting-notes-${id}`} class="flex flex-col gap-0.5">
			{#each shownNotes as note (note)}<span>{note}</span>{/each}
		</span>
	{/if}
	{#if type !== "radio" && currentValueLabel}
		<span>Current value: {currentValueLabel}</span>
	{/if}

	{#if saving}
		<span class="flex items-center gap-1" role="status"
			><Icon
				svg={circleNotchIcon}
				class="spinner"
				size={14}
				aria-hidden="true" />Saving…</span>
	{/if}
{/snippet}

{#if type === "toggle"}
	<div class="settings-row" data-setting={id}>
		<div class="settings-row-copy">
			{@render details()}
		</div>
		{#if isLoading}
			<span class="inline-flex h-7 w-9 shrink-0 items-center" aria-hidden="true"
				><span class="h-5 w-full skeleton rounded-full"></span></span>
		{:else}
			<label class="switch">
				<input
					type="checkbox"
					role="switch"
					aria-labelledby={`setting-label-${id}`}
					aria-describedby={describedBy}
					checked={isChecked}
					disabled={controlDisabled}
					onchange={(event) => {
						const requestedChecked = event.currentTarget.checked
						event.currentTarget.checked = isChecked
						onToggle?.(requestedChecked)
					}} />
				<span aria-hidden="true"></span>
			</label>
		{/if}
	</div>
{:else}
	<div
		class="flex flex-col gap-2 border-b border-line-subtle px-3 py-2.5 last:border-b-0"
		data-setting={id}>
		<div class="settings-row-copy">
			{@render details()}
		</div>
		<div
			class="-mx-2 flex flex-col gap-0.5"
			role="radiogroup"
			aria-labelledby={`setting-label-${id}`}>
			{#each radioOptions as option (option.value)}
				{#if isLoading}
					<span
						class="flex min-h-7 items-center gap-2 px-2"
						aria-hidden="true">
						<span class="size-4 shrink-0 skeleton rounded-full"></span>
						<span class="text-xs text-fg-muted">{option.label}</span>
					</span>
				{:else}
					<label
						class="radio-option"
						data-disabled={controlDisabled || option.disabled || undefined}>
						<span class="radio">
							<input
								type="radio"
								name={id}
								value={option.value}
								checked={radioValue === option.value}
								aria-describedby={describedBy}
								disabled={controlDisabled || option.disabled}
								onchange={(event) => {
									const radioGroup =
										event.currentTarget.closest(
											'[role="radiogroup"]',
										)
									for (const input of radioGroup?.querySelectorAll(
										"input",
									) ?? [])
										input.checked = input.value === radioValue
									onRadioSelect?.(option.value)
								}} />
							<span aria-hidden="true"></span>
						</span>
						<span class="text-xs text-fg">{option.label}</span>
					</label>
				{/if}
			{/each}
		</div>
	</div>
{/if}

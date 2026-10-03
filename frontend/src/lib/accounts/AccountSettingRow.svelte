<script lang="ts">
	import LoaderCircle from "@lucide/svelte/icons/loader-circle"
	import type { AccountSettingView } from "../backend/bridge"

	export type SettingRadioOption = {
		label: string
		value: string
		disabled?: boolean
		reason?: string
	}

	let {
		id,
		label,
		description,
		type,
		view,
		disabled = false,
		saving = false,
		radioOptions = [],
		error = "",
		onToggle,
		onRadioSelect,
	}: {
		id: string
		label: string
		description: string
		type: "toggle" | "radio"
		view?: AccountSettingView
		disabled?: boolean
		saving?: boolean
		radioOptions?: SettingRadioOption[]
		error?: string
		onToggle?: (checked: boolean) => void
		onRadioSelect?: (value: string) => void
	} = $props()

	const isChecked = $derived.by(() => {
		if (!view) return false
		if (view.boolValue !== null && view.boolValue !== undefined) {
			return view.boolValue === true
		}
		if (view.stringValue) {
			return [
				"Enabled",
				"Yes",
				"AllUsers",
				"AllConnections",
				"Friends",
				"Discoverable",
			].includes(view.stringValue)
		}
		return false
	})

	const currentRadioValue = $derived(
			view?.valueState === "known" &&
				view.options?.some((option) => option.stringValue === view.stringValue)
				? (view.stringValue ?? "")
				: "",
		),
		unavailable = $derived(
			view?.unavailableReason ||
				(!view ? "This setting is unavailable for this account." : ""),
		),
		controlDisabled = $derived(disabled || saving || !view?.editable)
</script>

{#snippet details()}
	<strong id={`setting-label-${id}`}>{label}</strong>
	<span id={`setting-description-${id}`}>{description}</span>
	{#if unavailable && (type !== "radio" || currentRadioValue)}<span
			>{unavailable}</span
		>{/if}
	{#if type !== "radio" && view && view.valueState !== "known"}
		<span>Current value: {view.valueLabel}</span>
	{/if}

	{#if saving}
		<span class="settings-row-detail" role="status"
			><LoaderCircle class="spinner" size={12} aria-hidden="true" />Saving</span>
	{/if}
	{#if error}<div class="settings-inline-error" role="alert">{error}</div>{/if}
{/snippet}

{#if type === "toggle"}
	<div
		class="settings-row"
		data-account-setting={id}
		class:settings-row-disabled={controlDisabled}>
		<div class="settings-row-copy">
			{@render details()}
		</div>
		<label class="settings-toggle">
			<input
				type="checkbox"
				aria-labelledby={`setting-label-${id}`}
				aria-describedby={`setting-description-${id}`}
				checked={isChecked}
				disabled={controlDisabled}
				onchange={(e) => onToggle?.(e.currentTarget.checked)} />
			<span aria-hidden="true"></span>
		</label>
	</div>
{:else}
	<div
		class="settings-radio-group"
		data-account-setting={id}
		class:settings-radio-group-disabled={controlDisabled}>
		<div class="settings-radio-group-header">
			{@render details()}
		</div>
		<div
			class="settings-radio-options"
			role="radiogroup"
			aria-labelledby={`setting-label-${id}`}>
			{#each radioOptions as option (option.value)}
				<label
					class="settings-radio-option"
					class:disabled={controlDisabled || option.disabled}
					data-tooltip={option.disabled && option.reason
						? option.reason
						: undefined}>
					<span class="settings-radio">
						<input
							type="radio"
							name={id}
							value={option.value}
							checked={currentRadioValue === option.value}
							aria-describedby={`setting-description-${id}`}
							disabled={controlDisabled || option.disabled}
							onchange={() => onRadioSelect?.(option.value)} />
						<span aria-hidden="true"></span>
					</span>
					<span class="settings-radio-label">{option.label}</span>
				</label>
			{/each}
		</div>
	</div>
{/if}

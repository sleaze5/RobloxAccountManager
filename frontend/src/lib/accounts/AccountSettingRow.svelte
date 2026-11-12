<script lang="ts">
	import circleNotchIcon from "@phosphor-icons/core/regular/circle-notch.svg?raw"
	import Icon from "../shared/Icon.svelte"
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
		isLoading = false,
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
		isLoading?: boolean
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
		notes = $derived(
			isLoading
				? []
				: [
						...new Set([
							unavailable,
							...radioOptions.map((option) =>
								option.disabled ? (option.reason ?? "") : "",
							),
						]),
					].filter(Boolean),
		),
		controlDisabled = $derived(disabled || saving || !view?.editable)
</script>

{#snippet details()}
	<strong id={`setting-label-${id}`}>{label}</strong>
	<span id={`setting-description-${id}`}>{description}</span>
	{#if notes.length > 0}
		<span id={`setting-notes-${id}`} class="flex flex-col gap-0.5">
			{#each notes as note (note)}<span>{note}</span>{/each}
		</span>
	{/if}
	{#if type !== "radio" && view && view.valueState !== "known"}
		<span>Current value: {view.valueLabel}</span>
	{/if}

	{#if saving}
		<span class="flex items-center gap-1" role="status"
			><Icon
				svg={circleNotchIcon}
				class="spinner"
				size={14}
				aria-hidden="true" />Saving…</span>
	{/if}
	{#if error}<div class="notice-error" role="alert">{error}</div>{/if}
{/snippet}

{#if type === "toggle"}
	<div class="settings-row" data-account-setting={id}>
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
					aria-describedby={notes.length > 0
						? `setting-description-${id} setting-notes-${id}`
						: `setting-description-${id}`}
					checked={isChecked}
					disabled={controlDisabled}
					onchange={(event) => onToggle?.(event.currentTarget.checked)} />
				<span aria-hidden="true"></span>
			</label>
		{/if}
	</div>
{:else}
	<div
		class="flex flex-col gap-2 border-b border-line-subtle px-3 py-2.5 last:border-b-0"
		data-account-setting={id}>
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
								checked={currentRadioValue === option.value}
								aria-describedby={notes.length > 0
									? `setting-description-${id} setting-notes-${id}`
									: `setting-description-${id}`}
								disabled={controlDisabled || option.disabled}
								onchange={() => onRadioSelect?.(option.value)} />
							<span aria-hidden="true"></span>
						</span>
						<span class="text-xs text-fg">{option.label}</span>
					</label>
				{/if}
			{/each}
		</div>
	</div>
{/if}

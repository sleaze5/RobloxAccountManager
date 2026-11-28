<script lang="ts">
	import { notifyErrors } from "../notifications/error-notifications.svelte"
	import type { AccountSettingView } from "../backend/bridge"
	import SettingRow from "../shared/SettingRow.svelte"
	import type { SettingRadioOption } from "../shared/SettingRow.svelte"

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

	notifyErrors("Setting not saved", () => error)
</script>

<SettingRow
	{id}
	{label}
	{description}
	{type}
	{isChecked}
	radioValue={currentRadioValue}
	{radioOptions}
	{notes}
	currentValueLabel={view && view.valueState !== "known" ? view.valueLabel : ""}
	disabled={controlDisabled}
	{isLoading}
	{saving}
	{onToggle}
	{onRadioSelect} />

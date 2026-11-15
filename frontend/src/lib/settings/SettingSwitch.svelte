<script lang="ts">
	let {
		labelledBy,
		describedBy,
		isChecked,
		isDisabled,
		onToggle,
	}: {
		labelledBy: string
		describedBy: string
		isChecked: boolean
		isDisabled: boolean
		onToggle: (enabled: boolean) => void
	} = $props()
</script>

<label class="switch">
	<input
		type="checkbox"
		role="switch"
		aria-labelledby={labelledBy}
		aria-describedby={describedBy}
		checked={isChecked}
		disabled={isDisabled}
		onchange={(event) => {
			const enabled = event.currentTarget.checked
			// The switch shows the saved state until the backend confirms the change.
			event.currentTarget.checked = isChecked
			onToggle(enabled)
		}} />
	<span aria-hidden="true"></span>
</label>

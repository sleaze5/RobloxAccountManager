<script lang="ts">
	import { automaticUnlockSummary } from "../shared/automatic-unlock"
	import CheckboxMark from "../shared/CheckboxMark.svelte"

	let { checked = $bindable() }: { checked: boolean } = $props()
</script>

<label class="check-option mt-3 text-left">
	<span class="checkbox">
		<input bind:checked type="checkbox" />
		<CheckboxMark />
	</span>
	<span
		><strong>Automatic unlock</strong><small>{automaticUnlockSummary}</small></span>
</label>

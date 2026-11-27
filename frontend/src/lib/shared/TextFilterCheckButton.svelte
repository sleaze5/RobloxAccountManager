<script lang="ts">
	import checkCircleIcon from "@phosphor-icons/core/regular/check-circle.svg?raw"
	import Icon from "./Icon.svelte"
	import type { TextFilterCheck } from "./text-filter-check.svelte"

	let {
		check,
		text,
		isChecking,
		disabled,
		onCheck,
		onEdit,
	}: {
		check: TextFilterCheck
		text: string
		isChecking: boolean
		disabled: boolean
		onCheck: () => void
		onEdit: () => void
	} = $props()
</script>

<button
	class="button"
	type="button"
	aria-busy={isChecking}
	disabled={disabled ||
		(check.filteredPreview === undefined &&
			(text === "" || check.current !== null))}
	onclick={() => (check.filteredPreview === undefined ? onCheck() : onEdit())}>
	{#if isChecking}
		Checking…
	{:else if check.filteredPreview !== undefined}
		Edit
	{:else}
		{#if check.isClean}<Icon
				svg={checkCircleIcon}
				class="text-success"
				size={14}
				aria-hidden="true" />{/if}
		Check
	{/if}
</button>

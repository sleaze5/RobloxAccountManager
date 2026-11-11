<script lang="ts">
	import checkCircleIcon from "@phosphor-icons/core/regular/check-circle.svg?raw"
	import Icon from "./Icon.svelte"
	import type { TextFilterCheck } from "./text-filter-check.svelte"

	let {
		check,
		text,
		isValidating,
		disabled,
		onValidate,
		onEdit,
	}: {
		check: TextFilterCheck
		text: string
		isValidating: boolean
		disabled: boolean
		onValidate: () => void
		onEdit: () => void
	} = $props()
</script>

<button
	class="button"
	type="button"
	aria-busy={isValidating}
	disabled={disabled ||
		(check.filteredPreview === undefined &&
			(text === "" || check.current !== null))}
	onclick={() => (check.filteredPreview === undefined ? onValidate() : onEdit())}>
	{#if isValidating}
		Validating…
	{:else if check.filteredPreview !== undefined}
		Edit
	{:else}
		{#if check.isClean}<Icon
				svg={checkCircleIcon}
				class="text-success"
				size={14}
				aria-hidden="true" />{/if}
		Validate
	{/if}
</button>

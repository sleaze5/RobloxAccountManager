<script lang="ts">
	import checkBoldIcon from "@phosphor-icons/core/bold/check-bold.svg?raw"
	import minusBoldIcon from "@phosphor-icons/core/bold/minus-bold.svg?raw"
	import Icon from "./Icon.svelte"

	let { isIndeterminate = false }: { isIndeterminate?: boolean } = $props()
</script>

<span aria-hidden="true"
	><Icon
		svg={isIndeterminate ? minusBoldIcon : checkBoldIcon}
		size={11}
		aria-hidden="true" /></span>

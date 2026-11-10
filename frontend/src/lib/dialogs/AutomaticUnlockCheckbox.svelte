<script lang="ts">
	import infoIcon from "@phosphor-icons/core/regular/info.svg?raw"
	import { automaticUnlockDialogDescription } from "../shared/automatic-unlock"
	import CheckboxMark from "../shared/CheckboxMark.svelte"
	import Icon from "../shared/Icon.svelte"

	let { checked = $bindable() }: { checked: boolean } = $props()
</script>

<div class="mt-2 flex items-center justify-start gap-1 text-xs text-fg-muted">
	<label class="flex items-center gap-1">
		<span class="checkbox">
			<input bind:checked type="checkbox" />
			<CheckboxMark />
		</span>
		Enable automatic unlock
	</label>
	<button
		class="flex size-5 cursor-help items-center justify-center text-fg-faint transition-colors duration-100 hover:text-fg focus-visible:text-fg"
		type="button"
		aria-label="About automatic unlock"
		data-tooltip={automaticUnlockDialogDescription}>
		<Icon svg={infoIcon} size={14} aria-hidden="true" />
	</button>
</div>

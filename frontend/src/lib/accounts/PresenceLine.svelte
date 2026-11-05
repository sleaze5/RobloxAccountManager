<script lang="ts">
	import Icon from "../shared/Icon.svelte"
	import type { UserPresence } from "../backend/bridge"
	import {
		presenceDetail,
		presenceIcon,
		presenceLabel,
		presenceTextClass,
	} from "./account-model"

	let { presence }: { presence: UserPresence } = $props()
	const detail = $derived(presenceDetail(presence))
</script>

<small class="flex w-full min-w-0 items-center gap-1 text-2xs text-fg-muted">
	<span
		class={[
			"inline-flex shrink-0 items-center gap-1 font-medium",
			presenceTextClass(presence),
		]}
		><Icon
			svg={presenceIcon(presence)}
			size={12}
			aria-hidden="true" />{presenceLabel(presence)}{detail ? ":" : ""}</span>
	{#if detail}<span class="min-w-0 truncate">{detail}</span>{/if}
</small>

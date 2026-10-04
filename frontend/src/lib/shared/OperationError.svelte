<script lang="ts">
	import XIcon from "phosphor-svelte/lib/XIcon"
	import { errorIn, errorOut } from "./presence"

	let { store }: { store: { error: string; clearError: () => void } } = $props()
</script>

{#if store.error}
	<div
		class="operation-error"
		role="alert"
		aria-live="assertive"
		in:errorIn
		out:errorOut>
		<span>{store.error}</span>
		<button
			type="button"
			aria-label="Dismiss error"
			onclick={() => store.clearError()}>
			<XIcon size={15} aria-hidden="true" />
		</button>
	</div>
{/if}

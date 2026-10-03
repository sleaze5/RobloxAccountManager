<script lang="ts">
	import X from "@lucide/svelte/icons/x"
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
			<X size={13} />
		</button>
	</div>
{/if}

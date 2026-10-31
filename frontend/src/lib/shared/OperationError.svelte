<script lang="ts">
	import XIcon from "phosphor-svelte/lib/XIcon"
	import { errorIn, errorOut } from "./presence"

	let { store }: { store: { error: string; clearError: () => void } } = $props()
</script>

{#if store.error}
	<div
		class="fixed top-20 right-2 z-20 flex max-w-md items-center gap-2 rounded-lg border border-danger/40 bg-toast-danger py-2 pr-2 pl-3 text-left text-xs text-danger-muted shadow-float"
		role="alert"
		aria-live="assertive"
		in:errorIn
		out:errorOut>
		<span class="max-h-48 overflow-y-auto whitespace-pre-line">{store.error}</span>
		<button
			class="btn-close"
			type="button"
			aria-label="Dismiss error"
			onclick={() => store.clearError()}>
			<XIcon size={14} aria-hidden="true" />
		</button>
	</div>
{/if}

<script module lang="ts">
	export type ImageView = "2D" | "3D"
</script>

<script lang="ts">
	let { view = $bindable() }: { view: ImageView } = $props()
</script>

<div class="segmented w-24 shrink-0" role="group" aria-label="Image view">
	{#each ["2D", "3D"] as const as option (option)}
		<button
			type="button"
			aria-pressed={view === option}
			onclick={() => (view = option)}>{option}</button>
	{/each}
</div>

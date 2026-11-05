<script lang="ts">
	let {
		value,
		tone = "neutral",
		id,
	}: {
		value: string
		tone?: "neutral" | "success" | "warning" | "danger"
		id?: string
	} = $props()

	const tones = {
		neutral: "text-fg-muted",
		success: "text-success",
		warning: "text-warning",
		danger: "text-danger",
	}
</script>

<span class="mt-0.5 text-2xs text-fg-faint" {id} role="status"
	>Status: <span class={tones[tone]}>{value}</span></span>

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
</script>

<span class="settings-status" {id} role="status"
	>status: <span class="settings-status-value" data-tone={tone}>{value}</span></span>

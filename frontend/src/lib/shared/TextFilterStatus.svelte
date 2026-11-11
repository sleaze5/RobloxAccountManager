<script lang="ts">
	import { TextModeration } from "../backend/bridge"
	import type { TextFilterCheck } from "./text-filter-check.svelte"

	let { check, noun }: { check: TextFilterCheck; noun: string } = $props()

	const moderation = $derived(check.current?.filter?.moderation)
</script>

{#if check.isClean}
	<span class="sr-only">Roblox's filter won't tag this {noun}.</span>
{:else if moderation === TextModeration.TextModerationPartial}
	<span class="sr-only">Roblox will tag part of this {noun}.</span>
{:else if moderation === TextModeration.TextModerationFull}
	<span class="sr-only">Roblox will tag all of this {noun}.</span>
{/if}

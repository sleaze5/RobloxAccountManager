<script lang="ts">
	// MenuTitle heads a menu that acts on several items with their count,
	// such as "3 accounts selected".
	let { count, noun }: { count: number; noun: string } = $props()
</script>

<div class="flex h-8 items-center gap-1 px-2 text-xs" aria-hidden="true">
	<strong class="font-mono font-semibold text-fg">{count}</strong>
	<span class="min-w-0 truncate text-fg-muted">{noun} selected</span>
</div>
<div class="menu-separator"></div>

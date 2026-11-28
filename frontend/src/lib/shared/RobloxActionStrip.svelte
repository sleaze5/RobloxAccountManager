<script lang="ts">
	import type { Snippet } from "svelte"

	// RobloxActionStrip groups the actions that change Roblox for the selected
	// account, which the status strip names. Each action asks first, and the
	// confirmation names the account again. Actions that change only the app
	// stay outside the strip.
	let { username, children }: { username: string; children: Snippet } = $props()
</script>

<div class="action-strip" role="group" aria-label={`Roblox actions as @${username}`}>
	{@render children()}
</div>

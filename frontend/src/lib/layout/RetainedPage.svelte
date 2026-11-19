<script lang="ts">
	import type { Snippet } from "svelte"
	import { setPageVisibility } from "./page-visibility"

	// RetainedPage holds a page that stays mounted while another page shows, so
	// that switching back keeps its scroll positions, sub-pages, and loaded
	// lists.
	let { isShown, children }: { isShown: boolean; children: Snippet } = $props()

	setPageVisibility({
		get isShown() {
			return isShown
		},
	})
</script>

<div class="flex min-h-0 flex-1 flex-col" hidden={!isShown} data-retained-page>
	{@render children()}
</div>

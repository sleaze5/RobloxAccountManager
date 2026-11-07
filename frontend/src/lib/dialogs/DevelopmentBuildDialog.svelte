<script lang="ts">
	import Dialog from "../shared/Dialog.svelte"
	import { Application } from "@wailsio/runtime"

	let { version, onContinue }: { version: string; onContinue: () => void } = $props()
</script>

<Dialog
	class="max-w-sm text-left"
	role="alertdialog"
	aria-labelledby="development-build-title"
	aria-describedby="development-build-summary"
	strong>
	<div class="dialog-heading">
		<div>
			<h2 id="development-build-title">Development build</h2>
			<p id="development-build-summary">
				Version <span class="font-mono">{version}</span> isn't a release. It may have
				bugs that affect your vault, and it can't update itself.
			</p>
		</div>
	</div>
	<div class="mt-3 notice-warning">Keep your accounts in a release build.</div>
	<div class="dialog-actions">
		<button class="button" type="button" onclick={() => void Application.Quit()}
			>Quit</button>
		<button
			class="button-primary"
			type="button"
			{@attach (button) => button.focus()}
			onclick={onContinue}>Continue</button>
	</div>
</Dialog>

<script lang="ts">
	import { Application } from "@wailsio/runtime"
	import {
		modalBackdropIn,
		modalBackdropOut,
		modalCardIn,
		modalCardOut,
		modalFocus,
	} from "../shared/presence"

	let { version, onContinue }: { version: string; onContinue: () => void } = $props()
</script>

<div
	class="dialog-backdrop z-50 bg-scrim-strong"
	in:modalBackdropIn
	out:modalBackdropOut>
	<div
		class="dialog-card max-w-sm text-left"
		role="alertdialog"
		aria-modal="true"
		aria-labelledby="development-build-title"
		aria-describedby="development-build-summary"
		use:modalFocus
		in:modalCardIn
		out:modalCardOut>
		<div class="dialog-heading">
			<div>
				<h2 id="development-build-title">Development build</h2>
				<p id="development-build-summary">
					Version <span class="font-mono">{version}</span> isn't a release. It may
					have bugs that affect your vault, and it can't update itself.
				</p>
			</div>
		</div>
		<div class="mt-3 notice-warning">Keep your accounts in a release build.</div>
		<div class="dialog-actions">
			<button class="btn" type="button" onclick={() => void Application.Quit()}
				>Quit</button>
			<button
				class="btn-primary"
				type="button"
				{@attach (button) => button.focus()}
				onclick={onContinue}>Continue</button>
		</div>
	</div>
</div>

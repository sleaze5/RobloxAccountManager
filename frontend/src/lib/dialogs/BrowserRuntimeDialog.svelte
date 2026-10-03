<script lang="ts">
	import Download from "@lucide/svelte/icons/download"
	import X from "@lucide/svelte/icons/x"
	import BrowserRuntimeActivity from "../browser/BrowserRuntimeActivity.svelte"
	import type { BrowserStore } from "../browser/browser-store.svelte"
	import { RuntimeStatus } from "../backend/bridge"
	import { formatCompactBytes } from "../shared/bytes"
	import {
		modalBackdropIn,
		modalBackdropOut,
		modalCardIn,
		modalCardOut,
		modalFocus,
	} from "../shared/presence"

	let { browser, onDismiss }: { browser: BrowserStore; onDismiss: () => void } =
		$props()
	const runtime = $derived(browser.snapshot.runtime),
		active = $derived(
			runtime.status === RuntimeStatus.RuntimeDownloading ||
				runtime.status === RuntimeStatus.RuntimeInstalling,
		),
		damaged = $derived(runtime.status === RuntimeStatus.RuntimeDamaged)

	$effect(() => browser.trackRuntimeView())
</script>

<div class="modal-backdrop" in:modalBackdropIn out:modalBackdropOut>
	<section
		class="modal-card runtime-card"
		use:modalFocus
		in:modalCardIn
		out:modalCardOut
		aria-labelledby="runtime-title">
		<div class="modal-heading">
			<div>
				<h2 id="runtime-title">
					{damaged ? "Redownload browser?" : "Download browser?"}
				</h2>
				<p>
					{damaged
						? "The installed browser is invalid and must be replaced."
						: "Chrome for Testing is required for isolated browser sessions."}
				</p>
			</div>
			<button
				type="button"
				aria-label="Dismiss browser download"
				onclick={onDismiss}><X size={15} /></button>
		</div>
		<div class="runtime-details">
			<span>Version</span><code>{runtime.requiredVersion}</code><span
				>Download size</span
			><code
				>{runtime.approximateBytes > 0
					? formatCompactBytes(runtime.approximateBytes)
					: "Unknown"}</code>
		</div>
		{#if browser.error || runtime.error}<div class="modal-error" role="alert">
				{browser.error || runtime.error}
			</div>{/if}
		{#if active}<BrowserRuntimeActivity {runtime} />{/if}
		<div class="modal-actions">
			<button type="button" onclick={onDismiss}
				>{active ? "Continue in background" : "Not now"}</button
			>{#if runtime.canCancel}<button
					type="button"
					onclick={() => void browser.cancelDownload()}
					>Cancel download</button
				>{:else if !active}<button
					class="primary-action"
					type="button"
					onclick={() =>
						void browser.download(
							runtime.status === RuntimeStatus.RuntimeDamaged,
						)}
					><Download size={14} />{damaged
						? "Redownload browser"
						: "Download browser"}</button
				>{/if}
		</div>
	</section>
</div>

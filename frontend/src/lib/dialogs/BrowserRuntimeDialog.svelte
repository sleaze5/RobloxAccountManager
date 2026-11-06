<script lang="ts">
	import downloadSimpleIcon from "@phosphor-icons/core/regular/download-simple.svg?raw"
	import xIcon from "@phosphor-icons/core/regular/x.svg?raw"
	import Icon from "../shared/Icon.svelte"
	import BrowserRuntimeActivity from "../browser/BrowserRuntimeActivity.svelte"
	import type { BrowserStore } from "../browser/browser-store.svelte"
	import { RuntimeStatus } from "../backend/bridge"
	import { formatCompactBytes } from "../shared/bytes"
	import Dialog from "../shared/Dialog.svelte"

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

<Dialog class="max-w-md" aria-labelledby="runtime-title">
	<div class="dialog-heading">
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
		<button type="button" aria-label="Dismiss browser download" onclick={onDismiss}
			><Icon svg={xIcon} size={14} aria-hidden="true" /></button>
	</div>
	<div
		class="mt-3 grid grid-cols-2 gap-2 text-xs text-fg-muted *:even:justify-self-end">
		<span>Version</span><code>{runtime.requiredVersion}</code><span
			>Download size</span
		><code
			>{runtime.approximateBytes > 0
				? formatCompactBytes(runtime.approximateBytes)
				: "Unknown"}</code>
	</div>
	{#if browser.error || runtime.error}<div class="mt-3 notice-error" role="alert">
			{browser.error || runtime.error}
		</div>{/if}
	{#if active}<div class="mt-3"><BrowserRuntimeActivity {runtime} /></div>{/if}
	<div class="dialog-actions">
		<button class="btn" type="button" onclick={onDismiss}
			>{active ? "Continue in background" : "Not now"}</button
		>{#if runtime.canCancel}<button
				class="btn"
				type="button"
				onclick={() => void browser.cancelDownload()}>Cancel download</button
			>{:else if !active}<button
				class="btn-primary"
				type="button"
				onclick={() =>
					void browser.download(
						runtime.status === RuntimeStatus.RuntimeDamaged,
					)}
				><Icon svg={downloadSimpleIcon} size={16} aria-hidden="true" />{damaged
					? "Redownload browser"
					: "Download browser"}</button
			>{/if}
	</div>
</Dialog>

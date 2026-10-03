<script lang="ts">
	import { RuntimeStatus } from "../backend/bridge"
	import type { RuntimeState } from "../backend/bridge"
	import { formatCompactBytes } from "../shared/bytes"

	let { runtime }: { runtime: RuntimeState } = $props()
	const installing = $derived(runtime.status === RuntimeStatus.RuntimeInstalling),
		determinate = $derived(runtime.downloadTotalBytes > 0),
		progress = $derived(
			determinate
				? Math.min(
						100,
						(runtime.downloadedBytes / runtime.downloadTotalBytes) * 100,
					)
				: 0,
		)

	function formatETA(seconds: number | null): string {
		if (seconds === null) return "--:--"
		const hours = Math.floor(seconds / 3600),
			minutes = String(Math.floor((seconds % 3600) / 60)).padStart(2, "0"),
			remainder = String(seconds % 60).padStart(2, "0")
		return `${hours > 0 ? `${hours}:` : ""}${minutes}:${remainder}`
	}
</script>

<div class="runtime-activity">
	<strong>{installing ? "Installing browser" : "Downloading browser"}</strong>
	{#if installing}
		<span>Preparing Chrome for Testing for use.</span>
	{:else}
		<div class="runtime-transfer">
			<span
				>{formatCompactBytes(runtime.downloadedBytes)}/{determinate
					? formatCompactBytes(runtime.downloadTotalBytes)
					: "--"}</span>
			<span>@ {formatCompactBytes(runtime.downloadBytesPerSecond)}/s</span>
			<span>ETA: {formatETA(runtime.downloadEtaSeconds)}</span>
		</div>
	{/if}
	{#if determinate && !installing}
		<div
			class="runtime-progress"
			role="progressbar"
			aria-label="Browser download progress"
			aria-valuemin={0}
			aria-valuemax={100}
			aria-valuenow={Math.round(progress)}>
			<div style:transform="scaleX({progress / 100})"></div>
		</div>
	{:else}
		<div
			class="runtime-progress indeterminate"
			role="progressbar"
			aria-label={installing
				? "Browser installation progress"
				: "Browser download progress"}>
			<div></div>
		</div>
	{/if}
</div>

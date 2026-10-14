<script lang="ts">
	import CircleNotchIcon from "phosphor-svelte/lib/CircleNotchIcon"
	import { RuntimeStatus } from "../backend/bridge"
	import type { RuntimeState } from "../backend/bridge"
	import { formatCompactBytes } from "../shared/bytes"
	import { formatClock } from "../shared/duration"

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
</script>

<div class="runtime-activity">
	<strong>
		{#if installing || !determinate}
			<CircleNotchIcon
				class="spinner"
				size={14}
				role="progressbar"
				aria-label={installing
					? "Browser installation progress"
					: "Browser download progress"} />
		{/if}
		{installing ? "Installing browser" : "Downloading browser"}
	</strong>
	{#if installing}
		<span>Preparing Chrome for Testing for use.</span>
	{:else}
		<div class="runtime-transfer">
			<span
				>{formatCompactBytes(runtime.downloadedBytes)}/{determinate
					? formatCompactBytes(runtime.downloadTotalBytes)
					: "--"}</span>
			<span>@ {formatCompactBytes(runtime.downloadBytesPerSecond)}/s</span>
			<span
				>ETA: {runtime.downloadEtaSeconds === null
					? "--:--"
					: formatClock(runtime.downloadEtaSeconds, "days")}</span>
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
	{/if}
</div>

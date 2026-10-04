<script lang="ts">
	import DownloadSimpleIcon from "phosphor-svelte/lib/DownloadSimpleIcon"
	import CircleNotchIcon from "phosphor-svelte/lib/CircleNotchIcon"
	import ArrowsClockwiseIcon from "phosphor-svelte/lib/ArrowsClockwiseIcon"
	import ArrowCounterClockwiseIcon from "phosphor-svelte/lib/ArrowCounterClockwiseIcon"
	import { UpdateStatus } from "../backend/bridge"
	import { formatCompactBytes } from "../shared/bytes"
	import { updateStore } from "../updates/update-store.svelte"
	import SettingsStatus from "./SettingsStatus.svelte"

	const update = $derived(updateStore.state),
		error = $derived(updateStore.error || update.error || ""),
		version = $derived(update.availableVersion ?? ""),
		determinate = $derived(update.totalBytes > 0),
		progress = $derived(
			determinate
				? Math.min(100, (update.downloadedBytes / update.totalBytes) * 100)
				: 0,
		)

	const summary = $derived.by((): string => {
		switch (update.status) {
			case UpdateStatus.StatusChecking:
				return "Checking for a newer version."
			case UpdateStatus.StatusUpToDate:
				return "You have the latest version."
			case UpdateStatus.StatusAvailable:
				return `Version ${version} is available.`
			case UpdateStatus.StatusDownloading:
				return `Downloading version ${version}.`
			case UpdateStatus.StatusReady:
				return `Version ${version} is downloaded and verified. Restart to install it.`
			case UpdateStatus.StatusRestarting:
				return `Restarting to install version ${version}.`
			default:
				return "Updates have not been checked yet."
		}
	})

	const status = $derived.by(
		(): { value: string; tone: "neutral" | "success" | "warning" } => {
			switch (update.status) {
				case UpdateStatus.StatusChecking:
					return { value: "Checking", tone: "neutral" }
				case UpdateStatus.StatusUpToDate:
					return { value: "Up to date", tone: "success" }
				case UpdateStatus.StatusAvailable:
					return { value: "Update available", tone: "warning" }
				case UpdateStatus.StatusDownloading:
					return { value: "Downloading", tone: "neutral" }
				case UpdateStatus.StatusReady:
					return { value: "Ready to install", tone: "success" }
				case UpdateStatus.StatusRestarting:
					return { value: "Restarting", tone: "neutral" }
				default:
					return { value: "Not checked", tone: "neutral" }
			}
		},
	)
</script>

<section class="settings-section" aria-labelledby="about-updates-title">
	<h3 id="about-updates-title">Updates</h3>
	{#if error}<div class="settings-inline-error" role="alert">{error}</div>{/if}
	<div class="settings-rows">
		<div class="settings-row">
			<div class="settings-row-copy">
				<strong>Version {update.currentVersion}</strong><span>{summary}</span>
				<SettingsStatus value={status.value} tone={status.tone} />
			</div>
			<div class="settings-row-actions">
				{#if update.status === UpdateStatus.StatusAvailable}<button
						class="settings-row-action"
						type="button"
						onclick={() => void updateStore.install()}
						><DownloadSimpleIcon size={16} aria-hidden="true" />Download
						update</button
					>{:else if update.status === UpdateStatus.StatusDownloading}<button
						class="settings-row-action"
						type="button"
						onclick={() => void updateStore.cancelDownload()}
						>Cancel download</button
					>{:else if update.status === UpdateStatus.StatusReady}<button
						class="settings-row-action"
						type="button"
						onclick={() => void updateStore.restart()}
						><ArrowCounterClockwiseIcon
							size={16}
							aria-hidden="true" />Restart to update</button
					>{:else if update.status === UpdateStatus.StatusRestarting}<button
						class="settings-row-action"
						type="button"
						disabled
						><CircleNotchIcon
							class="spinner"
							size={16}
							aria-hidden="true" />Restarting</button
					>{:else if update.status === UpdateStatus.StatusChecking}<button
						class="settings-row-action"
						type="button"
						disabled
						><CircleNotchIcon
							class="spinner"
							size={16}
							aria-hidden="true" />Checking</button
					>{:else}<button
						class="settings-row-action"
						type="button"
						onclick={() => void updateStore.check()}
						><ArrowsClockwiseIcon size={16} aria-hidden="true" />Check for
						updates</button
					>{/if}
			</div>
		</div>
		{#if update.status === UpdateStatus.StatusDownloading}
			<div class="settings-row">
				<div class="runtime-activity">
					<strong>Downloading update</strong>
					<div class="runtime-transfer">
						<span
							>{formatCompactBytes(update.downloadedBytes)}/{determinate
								? formatCompactBytes(update.totalBytes)
								: "--"}</span>
					</div>
					{#if determinate}
						<div
							class="runtime-progress"
							role="progressbar"
							aria-label="Update download progress"
							aria-valuemin={0}
							aria-valuemax={100}
							aria-valuenow={Math.round(progress)}>
							<div style:transform="scaleX({progress / 100})"></div>
						</div>
					{:else}
						<div
							class="runtime-progress indeterminate"
							role="progressbar"
							aria-label="Update download progress">
							<div></div>
						</div>
					{/if}
				</div>
			</div>
		{/if}
	</div>
</section>

<script lang="ts">
	import arrowCounterClockwiseIcon from "@phosphor-icons/core/regular/arrow-counter-clockwise.svg?raw"
	import arrowSquareOutIcon from "@phosphor-icons/core/regular/arrow-square-out.svg?raw"
	import arrowsClockwiseIcon from "@phosphor-icons/core/regular/arrows-clockwise.svg?raw"
	import circleNotchIcon from "@phosphor-icons/core/regular/circle-notch.svg?raw"
	import downloadSimpleIcon from "@phosphor-icons/core/regular/download-simple.svg?raw"
	import { Browser } from "@wailsio/runtime"
	import Icon from "../shared/Icon.svelte"
	import { repository } from "../../../package.json"
	import { UpdateStatus } from "../backend/bridge"
	import { formatCompactBytes } from "../shared/bytes"
	import { updateStore } from "../updates/update-store.svelte"

	const update = $derived(updateStore.state),
		error = $derived(updateStore.error || update.error || ""),
		version = $derived(update.availableVersion ?? ""),
		releaseURL = $derived(
			`${repository.url.replace(/^git\+/, "").replace(/\.git$/, "")}/releases/tag/v${version}`,
		),
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
			case UpdateStatus.StatusDisabled:
				return "Updates are off in development builds."
			default:
				return "Updates have not been checked yet."
		}
	})
</script>

<section class="settings-section" aria-labelledby="about-updates-title">
	<h3 id="about-updates-title">Updates</h3>
	{#if error}<div class="notice-error" role="alert">{error}</div>{/if}
	<div class="settings-rows">
		<div class="settings-row">
			<div class="settings-row-copy">
				<strong>Version {update.currentVersion}</strong><span role="status"
					>{summary}</span>
			</div>
			<div class="flex items-center gap-2">
				{#if version && update.status !== UpdateStatus.StatusUpToDate}<button
						class="btn"
						type="button"
						onclick={() => void Browser.OpenURL(releaseURL)}
						><Icon
							svg={arrowSquareOutIcon}
							size={16}
							aria-hidden="true" />What's new</button
					>{/if}
				{#if update.status === UpdateStatus.StatusAvailable}<button
						class="btn"
						type="button"
						onclick={() => void updateStore.install()}
						><Icon
							svg={downloadSimpleIcon}
							size={16}
							aria-hidden="true" />Download update</button
					>{:else if update.status === UpdateStatus.StatusDownloading}<button
						class="btn"
						type="button"
						onclick={() => void updateStore.cancelDownload()}
						>Cancel download</button
					>{:else if update.status === UpdateStatus.StatusReady}<button
						class="btn"
						type="button"
						onclick={() => void updateStore.restart()}
						><Icon
							svg={arrowCounterClockwiseIcon}
							size={16}
							aria-hidden="true" />Restart to update</button
					>{:else if update.status === UpdateStatus.StatusRestarting}<button
						class="btn"
						type="button"
						disabled
						><Icon
							svg={circleNotchIcon}
							class="spinner"
							size={16}
							aria-hidden="true" />Restarting…</button
					>{:else if update.status === UpdateStatus.StatusChecking}<button
						class="btn"
						type="button"
						disabled
						><Icon
							svg={circleNotchIcon}
							class="spinner"
							size={16}
							aria-hidden="true" />Checking…</button
					>{:else if update.status !== UpdateStatus.StatusDisabled}<button
						class="btn"
						type="button"
						onclick={() => void updateStore.check()}
						><Icon
							svg={arrowsClockwiseIcon}
							data-turn
							size={16}
							aria-hidden="true" />Check for updates</button
					>{/if}
			</div>
		</div>
		{#if update.status === UpdateStatus.StatusDownloading}
			<div class="settings-row">
				<div class="transfer">
					<strong>
						{#if !determinate}
							<Icon
								svg={circleNotchIcon}
								class="spinner"
								size={14}
								role="progressbar"
								aria-label="Update download progress" />
						{/if}
						Downloading update
					</strong>
					<div
						class="flex flex-wrap gap-x-2 gap-y-1 font-mono text-2xs *:whitespace-nowrap">
						<span
							>{formatCompactBytes(update.downloadedBytes)}/{determinate
								? formatCompactBytes(update.totalBytes)
								: "--"}</span>
					</div>
					{#if determinate}
						<div
							class="progress-bar"
							role="progressbar"
							aria-label="Update download progress"
							aria-valuemin={0}
							aria-valuemax={100}
							aria-valuenow={Math.round(progress)}>
							<div style:transform="scaleX({progress / 100})"></div>
						</div>
					{/if}
				</div>
			</div>
		{/if}
	</div>
</section>

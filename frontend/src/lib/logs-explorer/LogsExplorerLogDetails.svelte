<script lang="ts">
	import CircleNotchIcon from "phosphor-svelte/lib/CircleNotchIcon"
	import ArrowsClockwiseIcon from "phosphor-svelte/lib/ArrowsClockwiseIcon"
	import { ClientLaunchMode, ClientLaunchSource } from "../backend/bridge"
	import type { LogsExplorerSession } from "../backend/bridge"
	import Timestamp from "../shared/Timestamp.svelte"
	import { formatCompactBytes } from "../shared/bytes"
	import LogsExplorerAccount from "./LogsExplorerAccount.svelte"
	import { formatElapsed } from "./logs-explorer-time"

	let {
		session,
		refreshing,
		onRefresh,
	}: {
		session: LogsExplorerSession
		refreshing: boolean
		onRefresh: () => void
	} = $props()
	const modes: Record<ClientLaunchMode, string> = {
		[ClientLaunchMode.$zero]: "Not recorded",
		[ClientLaunchMode.LaunchCold]: "Cold start",
		[ClientLaunchMode.LaunchWarm]: "Warm start",
		[ClientLaunchMode.LaunchTray]: "Tray startup",
		[ClientLaunchMode.LaunchTrayResume]: "Resumed from tray",
	}
	const sources: Record<ClientLaunchSource, string> = {
		[ClientLaunchSource.$zero]: "Not recorded",
		[ClientLaunchSource.SourceWebsite]: "Website",
		[ClientLaunchSource.SourceExistingClient]: "Existing client",
		[ClientLaunchSource.SourceTray]: "Tray",
	}
</script>

<div class="logs-explorer-details">
	<div class="logs-explorer-file-heading">
		<dl
			class="logs-explorer-detail-grid logs-explorer-file-metadata"
			aria-label="Log file">
			<div>
				<dt>File</dt>
				<dd class="logs-explorer-detail-mono">{session.fileName}</dd>
			</div>
			<div>
				<dt>Log size</dt>
				<dd class="logs-explorer-detail-mono">
					{formatCompactBytes(session.sizeBytes)}
				</dd>
			</div>
		</dl>
		<button
			class="control-button"
			type="button"
			disabled={refreshing}
			aria-busy={refreshing}
			onclick={onRefresh}
			aria-label={refreshing ? "Refreshing log" : "Refresh log"}
			data-tooltip="Refresh log">
			{#if refreshing}<CircleNotchIcon
					class="spinner"
					size={16}
					aria-hidden="true" />{:else}<ArrowsClockwiseIcon
					size={16}
					aria-hidden="true" />{/if}
		</button>
	</div>
	{#if session.userId}<LogsExplorerAccount userId={session.userId} />{/if}
	<dl
		class="logs-explorer-detail-grid logs-explorer-client-metadata"
		aria-label="Client details">
		<div>
			<dt>User ID</dt>
			<dd class="logs-explorer-detail-mono">
				{session.userId || "Not recorded"}
			</dd>
		</div>
		<div>
			<dt>Channel</dt>
			<dd class="logs-explorer-detail-mono">
				{session.channel || "Not recorded"}
			</dd>
		</div>
		<div class="logs-explorer-detail-pair">
			<dt>Client version</dt>
			<dd class="logs-explorer-detail-mono">
				{session.version || "Not recorded"}
			</dd>
		</div>
	</dl>
	<dl
		class="logs-explorer-detail-grid logs-explorer-client-metadata"
		aria-label="Client lifecycle">
		<div>
			<dt>Launch mode</dt>
			<dd>{modes[session.launchMode]}</dd>
		</div>
		<div>
			<dt>Launch source</dt>
			<dd>{sources[session.launchSource]}</dd>
		</div>
		<div>
			<dt>
				{session.endEstimated ? "Client end (last recorded)" : "Client end"}
			</dt>
			<dd>
				{#if session.endedAtMs}<Timestamp value={session.endedAtMs} />{:else}Not
					recorded{/if}
			</dd>
		</div>
		<div>
			<dt>Client lifetime</dt>
			<dd class="logs-explorer-detail-mono">
				{#if session.startedAtMs && session.endedAtMs}{session.endEstimated
						? "≈"
						: ""}{formatElapsed(session.lifetimeMs)}{:else}Not recorded{/if}
			</dd>
		</div>
	</dl>
</div>

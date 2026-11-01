<script lang="ts">
	import CircleNotchIcon from "phosphor-svelte/lib/CircleNotchIcon"
	import ArrowsClockwiseIcon from "phosphor-svelte/lib/ArrowsClockwiseIcon"
	import { ClientLaunchMode, ClientLaunchSource } from "../backend/bridge"
	import type { LogsExplorerSession } from "../backend/bridge"
	import Timestamp from "../shared/Timestamp.svelte"
	import { formatCompactBytes } from "../shared/bytes"
	import LogsExplorerAccount from "./LogsExplorerAccount.svelte"
	import { formatClock } from "../shared/duration"

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
		[ClientLaunchMode.$zero]: "",
		[ClientLaunchMode.LaunchCold]: "Cold start",
		[ClientLaunchMode.LaunchWarm]: "Warm start",
		[ClientLaunchMode.LaunchTray]: "Tray startup",
		[ClientLaunchMode.LaunchTrayResume]: "Resumed from tray",
	}
	const sources: Record<ClientLaunchSource, string> = {
		[ClientLaunchSource.$zero]: "",
		[ClientLaunchSource.SourceWebsite]: "Website",
		[ClientLaunchSource.SourceExistingClient]: "Existing client",
		[ClientLaunchSource.SourceTray]: "Tray",
	}
	const launch = $derived(
		[sources[session.launchSource], modes[session.launchMode]]
			.filter(Boolean)
			.join("; ") || "Not recorded",
	)
</script>

<div class="flex flex-col gap-2">
	<div class="flex items-center gap-3">
		<dl class="detail-grid min-w-0 flex-1" data-columns="fit" aria-label="Log file">
			<div>
				<dt>File</dt>
				<dd class="font-mono">{session.fileName}</dd>
			</div>
			<div>
				<dt>Log size</dt>
				<dd class="font-mono">
					{formatCompactBytes(session.sizeBytes)}
				</dd>
			</div>
		</dl>
		<button
			class="btn-icon"
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
					data-turn
					size={16}
					aria-hidden="true" />{/if}
		</button>
	</div>
	{#if session.userId}<LogsExplorerAccount userId={session.userId} />{/if}
	<dl class="detail-grid grid-cols-4" aria-label="Client details">
		<div>
			<dt>User ID</dt>
			<dd class="font-mono">
				{session.userId || "Not recorded"}
			</dd>
		</div>
		<div>
			<dt>Channel</dt>
			<dd class="font-mono">
				{session.channel || "Not recorded"}
			</dd>
		</div>
		<div>
			<dt>Client version</dt>
			<dd class="font-mono">
				{session.version || "Not recorded"}
			</dd>
		</div>
		<div>
			<dt>Version hash</dt>
			<dd class="font-mono">
				{session.versionHash || "Not recorded"}
			</dd>
		</div>
	</dl>
	<dl class="detail-grid grid-cols-4" aria-label="Client lifecycle">
		<div>
			<dt>Launch</dt>
			<dd>{launch}</dd>
		</div>
		<div>
			<dt>Client start</dt>
			<dd>
				{#if session.startedAtMs}<Timestamp
						value={session.startedAtMs} />{:else}Not recorded{/if}
			</dd>
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
			<dd class="font-mono">
				{#if session.startedAtMs && session.endedAtMs}{session.endEstimated
						? "≈"
						: ""}{formatClock(session.lifetimeMs / 1000, "days")}{:else}Not
					recorded{/if}
			</dd>
		</div>
	</dl>
</div>

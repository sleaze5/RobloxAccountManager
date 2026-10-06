<script lang="ts">
	import type { LogsExplorerVisit } from "../backend/bridge"
	import LogsExplorerGame from "./LogsExplorerGame.svelte"

	let { visit }: { visit: LogsExplorerVisit } = $props()

	const universeId = $derived(Number(visit.universeId)),
		placeId = $derived(Number(visit.placeId))
	const validId = (id: number) => Number.isSafeInteger(id) && id > 0
</script>

<div class="logs-explorer-details">
	{#if validId(universeId) && validId(placeId)}<LogsExplorerGame
			{universeId}
			{placeId} />{/if}
	<dl class="logs-explorer-detail-grid" aria-label="Game identifiers">
		<div>
			<dt>Place ID</dt>
			<dd class="logs-explorer-detail-mono">{visit.placeId}</dd>
		</div>
		<div>
			<dt>Job ID</dt>
			<dd class="logs-explorer-detail-mono">{visit.jobId}</dd>
		</div>
		{#if visit.universeId}<div class="logs-explorer-detail-wide">
				<dt>Universe ID</dt>
				<dd class="logs-explorer-detail-mono">{visit.universeId}</dd>
			</div>{/if}
	</dl>
	{#if visit.serverAddress || visit.datacenterId}
		<dl class="logs-explorer-detail-grid" aria-label="Server details">
			{#if visit.datacenterId}<div>
					<dt>Datacenter ID</dt>
					<dd class="logs-explorer-detail-mono">{visit.datacenterId}</dd>
				</div>{/if}
			{#if visit.serverAddress}<div>
					<dt>Server address</dt>
					<dd class="logs-explorer-detail-mono">{visit.serverAddress}</dd>
				</div>{/if}
		</dl>
	{/if}
	<dl class="logs-explorer-detail-grid" aria-label="Session details">
		<div class="logs-explorer-detail-wide">
			<dt>End reason</dt>
			<dd class="logs-explorer-detail-mono">
				{#if visit.disconnectCode}
					{visit.disconnectCode}{visit.disconnectReason
						? ` (${visit.disconnectReason})`
						: ""}
				{:else}
					{visit.disconnectReason || "Not recorded"}
				{/if}
			</dd>
		</div>
		{#if visit.referralPage}<div>
				<dt>Referral page</dt>
				<dd>{visit.referralPage}</dd>
			</div>{/if}
		{#if visit.joinSource}<div>
				<dt>Join source</dt>
				<dd>{visit.joinSource}</dd>
			</div>{/if}
		{#if visit.requestType}<div>
				<dt>Request type</dt>
				<dd>{visit.requestType}</dd>
			</div>{/if}
	</dl>
</div>

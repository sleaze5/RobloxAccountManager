<script lang="ts">
	import lockKeyIcon from "@phosphor-icons/core/regular/lock-key.svg?raw"
	import warningIcon from "@phosphor-icons/core/regular/warning.svg?raw"
	import Icon from "../shared/Icon.svelte"
	import ServerListSkeleton from "./ServerListSkeleton.svelte"
	import ServerActionsButton from "./ServerActionsButton.svelte"
	import { openServerMenuFromRow } from "./server-row-menu"
	import type { ServerMenuOpener } from "./server-row-menu"
	import type { ListedServers } from "./game-servers-state.svelte"

	let {
		servers,
		activeKey,
		menuID,
		onMenu,
	}: {
		servers: ListedServers
		activeKey: string | undefined
		menuID: string
		onMenu: ServerMenuOpener
	} = $props()

	const numbers = new Intl.NumberFormat(),
		shown = $derived(servers.page?.servers ?? [])
</script>

<div class="flex min-h-0 flex-1 flex-col">
	{#if servers.error}
		<div class="empty-state" role="alert">
			<Icon svg={warningIcon} size={28} aria-hidden="true" />
			<h2>RBXServers servers are unavailable</h2>
			<p>{servers.error}</p>
			<button class="button" type="button" onclick={() => void servers.load()}
				>Try again</button>
		</div>
	{:else if !servers.page}
		<ServerListSkeleton label="Loading RBXServers servers" />
	{:else if !shown.length}
		<div class="empty-state" role="status">
			<Icon svg={lockKeyIcon} size={28} aria-hidden="true" />
			<h2>No servers listed</h2>
			<p>Nobody has shared a private server of this place on RBXServers.</p>
		</div>
	{:else}
		<p
			class="flex flex-wrap gap-x-1 px-3 pt-2 font-mono text-2xs text-fg-faint"
			role="status">
			<span
				>{numbers.format(shown.length)}
				{shown.length === 1 ? "server" : "servers"}</span>
			<span aria-hidden="true">-</span>
			<span>Up to 30 shared on RBXServers</span>
		</p>
		<div
			class="server-table"
			data-variant="listed"
			data-loading={servers.loading || undefined}>
			<div class="server-columns" aria-hidden="true">
				<span>Name</span>
				<span>UUID</span>
			</div>
			<ol aria-label="RBXServers private servers" aria-busy={servers.loading}>
				{#each shown as server (server.id)}
					<li
						class="server-row"
						data-open={activeKey === server.id || undefined}
						oncontextmenu={(event) =>
							openServerMenuFromRow(event, server.id, onMenu)}>
						<span
							class="truncate text-xs font-medium text-fg"
							data-tooltip={server.name || undefined}
							>{server.name || "Unnamed server"}</span>
						<span
							class="truncate font-mono text-2xs text-fg-faint select-text"
							data-tooltip={server.id}>{server.id}</span>
						<ServerActionsButton
							open={activeKey === server.id}
							busy={servers.resolving === server.id}
							{menuID}
							onMenu={(trigger, toggle) =>
								onMenu(server.id, trigger, toggle)} />
					</li>
				{/each}
			</ol>
		</div>
	{/if}
</div>

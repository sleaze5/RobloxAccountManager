<script lang="ts">
	import circleNotchIcon from "@phosphor-icons/core/regular/circle-notch.svg?raw"
	import Icon from "../shared/Icon.svelte"
	import type { OwnedPrivateServer } from "../backend/bridge"
	import GameIcon from "../games/GameIcon.svelte"
	import { loadGamePlace } from "../games/game-place-cache"
	import PlaceIcon from "../games/PlaceIcon.svelte"
	import type { OwnedPrivateServersState } from "./owned-private-servers-state.svelte"

	let {
		owned,
		username,
		onManage,
	}: {
		owned: OwnedPrivateServersState
		username: string
		onManage: (server: OwnedPrivateServer) => void
	} = $props()

	const dates = new Intl.DateTimeFormat(undefined, { dateStyle: "medium" }),
		numbers = new Intl.NumberFormat()

	function renewal(server: OwnedPrivateServer): string {
		if (!server.expiresAtMs)
			return server.renews ? "Renews monthly" : "Does not renew"
		const date = dates.format(server.expiresAtMs)
		if (server.expiresAtMs < Date.now()) return `Ended ${date}`
		if (!server.renews) return `Ends ${date}`
		return server.price
			? `Renews ${date} for ${numbers.format(server.price)} Robux`
			: `Renews ${date}`
	}
</script>

{#if owned.error}
	<div class="notice-error" role="alert">{owned.error}</div>
{/if}

{#if !owned.servers}
	{#if owned.loading}
		<div class="flex items-center gap-2 text-xs text-fg-muted" role="status">
			<Icon svg={circleNotchIcon} class="spinner" size={16} aria-hidden="true" />
			Loading private servers
		</div>
	{/if}
{:else if owned.servers.length === 0}
	<p class="text-xs text-fg-faint">
		@{username} does not own any private servers. Buy one on a game's page on Roblox.
	</p>
{:else}
	<div class="settings-section">
		<ul
			class="settings-rows"
			aria-label="Private servers"
			aria-busy={owned.loading}>
			{#each owned.servers as server (server.serverId)}
				{@const name = server.name || "Unnamed server"}
				<li class="settings-row animate-enter justify-start gap-3">
					{#await loadGamePlace(server.universeId, server.placeId)}
						<GameIcon url="" />
					{:then summary}
						<PlaceIcon
							place={summary?.place ?? null}
							rootPlace={summary?.rootPlace ?? null} />
					{/await}
					<div class="settings-row-copy">
						<strong class="flex min-w-0 items-center gap-1.5">
							<span class="truncate">{name}</span>
							{#if !server.active}<span class="shrink-0 server-tag"
									>Inactive</span
								>{/if}
						</strong>
						<span class="truncate"
							>{server.gameName || `Place ${server.placeId}`} - {renewal(
								server,
							)}</span>
					</div>
					<button
						class="btn"
						type="button"
						aria-label={`Manage ${name}`}
						disabled={owned.loading}
						onclick={() => onManage(server)}>Manage</button>
				</li>
			{/each}
		</ul>
	</div>
	{#if owned.nextCursor}
		<button
			class="btn self-start"
			type="button"
			disabled={owned.loading}
			onclick={() => void owned.loadMore()}>
			{#if owned.loading}
				<Icon
					svg={circleNotchIcon}
					class="spinner"
					size={14}
					aria-hidden="true" />
				Loading…
			{:else}
				Load more
			{/if}
		</button>
	{/if}
{/if}

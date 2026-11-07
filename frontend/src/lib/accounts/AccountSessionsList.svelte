<script lang="ts">
	import circleNotchIcon from "@phosphor-icons/core/regular/circle-notch.svg?raw"
	import desktopIcon from "@phosphor-icons/core/regular/desktop.svg?raw"
	import deviceMobileIcon from "@phosphor-icons/core/regular/device-mobile.svg?raw"
	import devicesIcon from "@phosphor-icons/core/regular/devices.svg?raw"
	import globeIcon from "@phosphor-icons/core/regular/globe.svg?raw"
	import Icon from "../shared/Icon.svelte"
	import Timestamp from "../shared/Timestamp.svelte"
	import type { AccountSession } from "../backend/bridge"
	import type { AccountSessionsState } from "./account-sessions-state.svelte"

	let {
		sessions,
		username,
		onSignOut,
	}: {
		sessions: AccountSessionsState
		username: string
		onSignOut: () => void
	} = $props()

	const others = $derived(
		(sessions.sessions ?? []).filter((session) => !session.current).length,
	)

	function device(session: AccountSession): string {
		const client =
			session.agent === "Browser"
				? session.browser || "Browser"
				: session.agent === "App"
					? "Roblox app"
					: session.agent === "Studio"
						? "Roblox Studio"
						: session.agent || session.browser
		if (!client) return session.os || "Unknown device"
		return session.os ? `${client} on ${session.os}` : client
	}

	function icon(session: AccountSession): string {
		if (session.agent === "Browser") return globeIcon
		if (["iOS", "Android"].includes(session.os)) return deviceMobileIcon
		return session.agent || session.os ? desktopIcon : devicesIcon
	}
</script>

{#if sessions.error}
	<div class="notice-error" role="alert">{sessions.error}</div>
{/if}

{#if !sessions.sessions}
	{#if sessions.loading}
		<div class="flex items-center gap-2 text-xs text-fg-muted" role="status">
			<Icon svg={circleNotchIcon} class="spinner" size={16} aria-hidden="true" />
			Loading sessions…
		</div>
	{/if}
{:else}
	<section class="settings-section" aria-labelledby="account-sessions-title">
		<h3 class="flex items-center justify-between gap-2">
			<span id="account-sessions-title"
				>Active sessions <span class="font-normal text-fg-muted"
					>{sessions.sessions.length}</span
				></span>
			<button
				class="-my-1 btn"
				type="button"
				disabled={others === 0 || sessions.loading || sessions.signingOut}
				onclick={onSignOut}>Sign out other sessions</button>
		</h3>
		<p class="text-xs text-fg-muted">
			{others === 0
				? `No other browser, device, or app is signed in as @${username}.`
				: `Every browser, device, and app signed in as @${username}, most recent first.`}
		</p>
		<ul class="settings-rows" aria-label="Sessions" aria-busy={sessions.loading}>
			{#each sessions.sessions as session}
				<li class="settings-row justify-start gap-3">
					<span
						class="grid size-7 shrink-0 place-items-center rounded-md border border-line bg-raised text-fg-muted"
						aria-hidden="true">
						<Icon svg={icon(session)} size={16} />
					</span>
					<div class="settings-row-copy">
						<strong class="flex min-w-0 items-center gap-1.5">
							<span class="truncate">{device(session)}</span>
							{#if session.current}<span class="shrink-0 server-tag"
									>This app</span
								>{/if}
						</strong>
						<span class="flex min-w-0 flex-wrap gap-x-3">
							<span class="min-w-0 truncate"
								>{session.location || "Unknown location"}</span>
							{#if session.ip}<span
									class="font-mono text-2xs leading-4 text-fg-faint"
									>{session.ip}</span
								>{/if}
						</span>
					</div>
					<span class="shrink-0 text-xs text-fg-muted">
						{#if session.lastActiveMs}
							<Timestamp value={session.lastActiveMs} />
						{:else}
							Unknown
						{/if}
					</span>
				</li>
			{/each}
		</ul>
	</section>
{/if}

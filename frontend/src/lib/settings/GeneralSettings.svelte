<script lang="ts">
	import { PresenceScope } from "../backend/bridge"
	import NumberStepper from "../shared/NumberStepper.svelte"
	import type { LoggingLevel, SettingsStore } from "./settings-store.svelte"

	const presenceScopes = [
		{
			id: PresenceScope.PresenceAccounts,
			name: "Accounts bar",
			description: "Update presence for all stored accounts.",
		},
		{
			id: PresenceScope.PresenceProfile,
			name: "Active profile",
			description:
				"Update presence for the selected account while its profile is open.",
		},
	] as const
	const levels: { id: LoggingLevel; name: string; description: string }[] = [
		{
			description: "Detailed requests and internal operations.",
			id: "trace",
			name: "Trace",
		},
		{
			description: "Diagnostic information for development.",
			id: "debug",
			name: "Debug",
		},
		{
			description: "Normal application activity and completed operations.",
			id: "info",
			name: "Info",
		},
		{
			description: "Recoverable problems that need attention.",
			id: "warn",
			name: "Warning",
		},
		{
			description: "Failed operations and critical problems.",
			id: "error",
			name: "Error",
		},
	]

	let { store, query = "" }: { store: SettingsStore; query?: string } = $props(),
		normalizedQuery = $derived(query.trim().toLowerCase()),
		searchTerms = $derived(normalizedQuery.split(/\s+/).filter(Boolean)),
		visibleLevels = $derived(
			levels.filter((level) => {
				const text =
					`general logging ${level.name} ${level.description}`.toLowerCase()
				return searchTerms.every((term) => text.includes(term))
			}),
		),
		visiblePresenceScopes = $derived(
			presenceScopes.filter((scope) =>
				searchTerms.every((term) =>
					`general presence updates automatic enabled disabled interval seconds minutes online offline in game ${scope.name} ${scope.description}`
						.toLowerCase()
						.includes(term),
				),
			),
		),
		showAllLevels = $derived(
			searchTerms.every((term) =>
				"general logging all log levels enable disable every level at once".includes(
					term,
				),
			),
		)
</script>

{#if store.error}<div class="settings-inline-error" role="alert">
		{store.error}
	</div>{/if}

{#if visiblePresenceScopes.length > 0}
	<section class="settings-section" aria-labelledby="presence-updates-title">
		<h3 id="presence-updates-title">Presence updates</h3>
		{#each visiblePresenceScopes as scope (scope.id)}
			{@const settings = store.presence[scope.id]}
			<div class="settings-rows">
				<div class="settings-row">
					<div class="settings-row-copy">
						<strong id={`presence-${scope.id}-title`}>{scope.name}</strong>
						<span id={`presence-${scope.id}-description`}
							>{scope.description}</span>
					</div>
					<label class="settings-toggle">
						<input
							type="checkbox"
							role="switch"
							aria-labelledby={`presence-${scope.id}-title`}
							aria-describedby={`presence-${scope.id}-description`}
							checked={settings.enabled}
							disabled={!store.initialized || store.busyPresence}
							onchange={(event) => {
								const enabled = event.currentTarget.checked
								event.currentTarget.checked = settings.enabled
								void store.setPresenceUpdates(
									scope.id,
									enabled,
									settings.intervalSeconds,
								)
							}} />
						<span aria-hidden="true"></span>
					</label>
				</div>
				<div class="settings-row">
					<div class="settings-row-copy">
						<strong>Update interval</strong>
						<span>Seconds</span>
					</div>
					<NumberStepper
						label={`${scope.name} presence update interval in seconds`}
						value={settings.intervalSeconds}
						min={30}
						max={900}
						step={5}
						defaultValue={scope.id === PresenceScope.PresenceAccounts
							? 120
							: 60}
						disabled={!store.initialized ||
							store.busyPresence ||
							!settings.enabled}
						onChange={(seconds) =>
							store.setPresenceUpdates(
								scope.id,
								settings.enabled,
								seconds,
							)} />
				</div>
			</div>
		{/each}
	</section>
{/if}

{#if showAllLevels || visibleLevels.length > 0}
	<section class="settings-section" aria-labelledby="logging-title">
		<h3 id="logging-title">Logging</h3>
		<div class="settings-rows">
			{#if showAllLevels}
				<div class="settings-row settings-row-master">
					<div class="settings-row-copy">
						<strong>All log levels</strong>
						<span>Enable or disable every log level at once.</span>
					</div>
					<label class="settings-toggle">
						<input
							type="checkbox"
							aria-label="Enable all log levels"
							checked={store.areAllLoggingLevelsEnabled()}
							disabled={store.busyLevel !== null}
							onchange={(event) =>
								void store.setAllLoggingLevelsEnabled(
									event.currentTarget.checked,
								)} />
						<span aria-hidden="true"></span>
					</label>
				</div>
			{/if}
			{#each visibleLevels as level (level.id)}
				<div class="settings-row">
					<div class="settings-row-copy">
						<strong>{level.name}</strong>
						<span>{level.description}</span>
					</div>
					<label class="settings-toggle">
						<input
							type="checkbox"
							aria-label={`Enable ${level.name} logs`}
							checked={store.isLoggingLevelEnabled(level.id)}
							disabled={store.busyLevel !== null}
							onchange={(event) =>
								void store.setLoggingLevelEnabled(
									level.id,
									event.currentTarget.checked,
								)} />
						<span aria-hidden="true"></span>
					</label>
				</div>
			{/each}
		</div>
	</section>
{/if}

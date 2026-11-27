<script lang="ts">
	import { notifyErrors } from "../notifications/error-notifications.svelte"
	import { matchesSearch, searchTerms } from "../shared/search"
	import { matchesSetting, settingWords } from "./settings-search"
	import { OnlineStatusScope } from "../backend/bridge"
	import NumberStepper from "../shared/NumberStepper.svelte"
	import type { LoggingLevel, SettingsStore } from "./settings-store.svelte"

	const onlineStatusScopes = [
		{
			id: OnlineStatusScope.OnlineStatusAccounts,
			name: "Account list",
			description: "Update online status for all saved accounts.",
		},
		{
			id: OnlineStatusScope.OnlineStatusProfile,
			name: "Active profile",
			description:
				"Update online status for the selected account while its profile is open.",
		},
		{
			id: OnlineStatusScope.OnlineStatusSocial,
			name: "Social",
			description: "Update online status for listed users while Social is open.",
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
			description: "Normal app activity and completed operations.",
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
		terms = $derived(searchTerms(query)),
		visibleLevels = $derived(
			levels.filter((level) =>
				matchesSearch(
					terms,
					`${settingWords("logging")} ${level.name} ${level.description}`,
				),
			),
		),
		visibleOnlineStatusScopes = $derived(
			onlineStatusScopes.filter((scope) =>
				matchesSearch(
					terms,
					`${settingWords("onlineStatusUpdates")} ${scope.name} ${scope.description}`,
				),
			),
		),
		showAllLevels = $derived(matchesSetting(terms, "allLogLevels"))

	notifyErrors("Setting not saved", () => store.error)
</script>

{#if visibleOnlineStatusScopes.length > 0}
	<section class="settings-section" aria-labelledby="online-status-updates-title">
		<h3 id="online-status-updates-title">Online status updates</h3>
		{#each visibleOnlineStatusScopes as scope (scope.id)}
			{@const settings = store.onlineStatus[scope.id]}
			<div class="settings-rows">
				<div class="settings-row">
					<div class="settings-row-copy">
						<strong id={`onlineStatus-${scope.id}-title`}
							>{scope.name}</strong>
						<span id={`onlineStatus-${scope.id}-description`}
							>{scope.description}</span>
					</div>
					<label class="switch">
						<input
							type="checkbox"
							role="switch"
							aria-labelledby={`onlineStatus-${scope.id}-title`}
							aria-describedby={`onlineStatus-${scope.id}-description`}
							checked={settings.enabled}
							disabled={!store.initialized || store.busyOnlineStatus}
							onchange={(event) => {
								const enabled = event.currentTarget.checked
								event.currentTarget.checked = settings.enabled
								void store.setOnlineStatusUpdates(
									scope.id,
									enabled,
									settings.intervalSeconds,
								)
							}} />
						<span aria-hidden="true"></span>
					</label>
				</div>
				<div
					class="settings-suboptions"
					role="group"
					aria-label={`${scope.name} update options`}>
					<div class="settings-row">
						<div class="settings-row-copy">
							<strong>Update interval</strong>
							<span>Seconds between updates.</span>
						</div>
						<NumberStepper
							label={`${scope.name} update interval in seconds`}
							value={settings.intervalSeconds}
							min={30}
							max={900}
							step={5}
							defaultValue={scope.id ===
							OnlineStatusScope.OnlineStatusAccounts
								? 300
								: 60}
							disabled={!store.initialized ||
								store.busyOnlineStatus ||
								!settings.enabled}
							onChange={(seconds) =>
								store.setOnlineStatusUpdates(
									scope.id,
									settings.enabled,
									seconds,
								)} />
					</div>
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
				<div class="settings-row border-b-line-strong">
					<div class="settings-row-copy">
						<strong>All log levels</strong>
						<span>Turn every log level on or off.</span>
					</div>
					<label class="switch">
						<input
							type="checkbox"
							role="switch"
							aria-label="All log levels"
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
					<label class="switch">
						<input
							type="checkbox"
							role="switch"
							aria-label={`${level.name} logs`}
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

<script lang="ts">
	import { onMount } from "svelte"
	import { accountBackend, StorageMode } from "../backend/bridge"
	import type { AppLocation, LaunchReport } from "../backend/bridge"
	import { formatCompactDuration } from "../shared/duration"
	import Timestamp from "../shared/Timestamp.svelte"

	let report = $state<LaunchReport | null>(null),
		location = $state<AppLocation | null>(null),
		error = $state("")
	const build = $derived(report?.build),
		stages = $derived(report?.stages ?? []),
		components = $derived(report?.components ?? []),
		dependencyGroups = $derived([
			{
				label: "runtime",
				dependencies: [
					{ name: "go", version: build?.go_version ?? "" },
					...(build?.dependencies ?? []),
					...FRONTEND_DEPENDENCIES.runtime,
				],
			},
			{ label: "build", dependencies: FRONTEND_DEPENDENCIES.build },
		])

	onMount(() => {
		let disposed = false
		void (async () => {
			try {
				const [next, nextLocation] = await Promise.all([
					accountBackend.GetLaunchReport(),
					accountBackend.GetAppLocation(),
				])
				if (!disposed) {
					report = next
					location = nextLocation
				}
			} catch {
				if (!disposed) error = "Build and launch details could not be loaded."
			}
		})()
		return () => {
			disposed = true
		}
	})
</script>

{#if error}
	<div class="settings-inline-error" role="alert">{error}</div>
{:else if !report || !build}
	<p class="about-empty" role="status">Loading…</p>
{:else}
	<section class="settings-section" aria-labelledby="about-build-title">
		<h3 id="about-build-title">Build</h3>
		<dl class="about-grid">
			<dt>version</dt>
			<dd>{build.version}</dd>
			<dt>platform</dt>
			<dd>{build.os}/{build.architecture}</dd>
			<dt>built</dt>
			<dd>
				{#if build.build_time}
					<Timestamp value={build.build_time * 1000} />
				{:else}
					<span class="about-faint">not stamped</span>
				{/if}
			</dd>
		</dl>
	</section>

	<section class="settings-section" aria-labelledby="about-dependencies-title">
		<h3 id="about-dependencies-title">Dependencies</h3>
		<table class="about-table">
			{#each dependencyGroups as group (group.label)}
				<tbody>
					<tr>
						<th scope="colgroup" colspan="2">{group.label}</th>
					</tr>
					{#each group.dependencies as dependency (dependency.name)}
						<tr>
							<td>{dependency.name}</td>
							<td>{dependency.version}</td>
						</tr>
					{/each}
				</tbody>
			{/each}
		</table>
	</section>

	<section class="settings-section" aria-labelledby="about-launch-title">
		<h3 id="about-launch-title">Launch</h3>
		<dl class="about-grid">
			<dt>launch id</dt>
			<dd>{report.launchId}</dd>
			<dt>started</dt>
			<dd>
				<Timestamp value={report.startedAt} />
			</dd>
			<dt>ready</dt>
			<dd>{report.ready ? formatCompactDuration(report.readyMs) : "starting"}</dd>
			{#if location}
				<dt>storage</dt>
				<dd>
					{location.mode === StorageMode.ModePortable
						? "portable"
						: "standard"}
				</dd>
				<dt>data folder</dt>
				<dd>{location.directory}</dd>
			{/if}
		</dl>
		<table class="about-table">
			<thead>
				<tr>
					<th scope="col">stage</th>
					<th scope="col">duration</th>
				</tr>
			</thead>
			<tbody>
				{#each stages as stage, index (index)}
					<tr>
						<td>{stage.name}</td>
						<td>{formatCompactDuration(stage.duration_ms)}</td>
					</tr>
				{/each}
			</tbody>
		</table>
	</section>

	<section class="settings-section" aria-labelledby="about-components-title">
		<h3 id="about-components-title">Loaded components</h3>
		{#if components.length === 0}
			<p class="about-empty">No component records yet.</p>
		{:else}
			<ul class="about-components">
				{#each components as component (`${component.module}:${component.message}`)}
					<li>
						<div class="about-component-heading">
							<span class="about-module">{component.module}</span>
							<span>{component.message}</span>
							<Timestamp value={component.recordedAt} />
						</div>
						{#if component.attributes?.length}
							<dl class="about-attributes">
								{#each component.attributes as attribute (attribute.key)}
									<div>
										<dt>{attribute.key}</dt>
										<dd>{attribute.value}</dd>
									</div>
								{/each}
							</dl>
						{/if}
					</li>
				{/each}
			</ul>
		{/if}
	</section>
{/if}

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
	const gridClass =
			"grid grid-cols-4 border-t border-line-subtle font-mono text-2xs *:flex *:min-w-0 *:items-center *:gap-2 *:border-b *:border-line-subtle *:p-1 *:wrap-anywhere",
		tableClass =
			"w-full border-collapse border-t border-line-subtle font-mono text-2xs",
		cellClass = "border-b border-line-subtle p-1",
		numberClass = "w-24 text-right whitespace-nowrap"
</script>

{#if error}
	<div class="notice-error" role="alert">{error}</div>
{:else if !report || !build}
	<p class="px-1 text-xs text-fg-faint" role="status">Loading…</p>
{:else}
	<section class="settings-section" aria-labelledby="about-build-title">
		<h3 id="about-build-title">Build</h3>
		<dl class={gridClass}>
			<dt class="font-normal text-fg-faint">version</dt>
			<dd class="col-span-3 text-fg">{build.version}</dd>
			<dt class="font-normal text-fg-faint">platform</dt>
			<dd class="col-span-3 text-fg">{build.os}/{build.architecture}</dd>
			<dt class="font-normal text-fg-faint">built</dt>
			<dd class="col-span-3 text-fg">
				{#if build.build_time}
					<Timestamp value={build.build_time * 1000} />
				{:else}
					<span class="text-fg-faint">not stamped</span>
				{/if}
			</dd>
		</dl>
	</section>

	<section class="settings-section" aria-labelledby="about-dependencies-title">
		<h3 id="about-dependencies-title">Dependencies</h3>
		<table class={tableClass}>
			{#each dependencyGroups as group, groupIndex (group.label)}
				<tbody>
					<tr>
						<th
							class={[
								cellClass,
								"text-left font-normal text-fg-faint",
								groupIndex > 0 && "pt-3",
							]}
							scope="colgroup"
							colspan="2">
							{group.label}
						</th>
					</tr>
					{#each group.dependencies as dependency (dependency.name)}
						<tr>
							<td class={[cellClass, "text-fg"]}>{dependency.name}</td>
							<td class={[cellClass, numberClass, "text-fg"]}
								>{dependency.version}</td>
						</tr>
					{/each}
				</tbody>
			{/each}
		</table>
	</section>

	<section class="settings-section" aria-labelledby="about-launch-title">
		<h3 id="about-launch-title">Launch</h3>
		<dl class={gridClass}>
			<dt class="font-normal text-fg-faint">launch id</dt>
			<dd class="col-span-3 text-fg">{report.launchId}</dd>
			<dt class="font-normal text-fg-faint">started</dt>
			<dd class="col-span-3 text-fg">
				<Timestamp value={report.startedAt} />
			</dd>
			<dt class="font-normal text-fg-faint">ready</dt>
			<dd class="col-span-3 text-fg">
				{report.ready ? formatCompactDuration(report.readyMs) : "starting"}
			</dd>
			{#if location}
				<dt class="font-normal text-fg-faint">storage</dt>
				<dd class="col-span-3 text-fg">
					{location.mode === StorageMode.ModePortable
						? "portable"
						: "standard"}
				</dd>
				<dt class="font-normal text-fg-faint">data folder</dt>
				<dd class="col-span-3 text-fg">{location.directory}</dd>
			{/if}
		</dl>
		<table class={tableClass}>
			<thead>
				<tr>
					<th
						class={[cellClass, "text-left font-normal text-fg-faint"]}
						scope="col">stage</th>
					<th
						class={[cellClass, numberClass, "font-normal text-fg-faint"]}
						scope="col">
						duration
					</th>
				</tr>
			</thead>
			<tbody>
				{#each stages as stage, index (index)}
					<tr>
						<td class={[cellClass, "text-fg"]}>{stage.name}</td>
						<td class={[cellClass, numberClass, "text-fg"]}>
							{formatCompactDuration(stage.duration_ms)}
						</td>
					</tr>
				{/each}
			</tbody>
		</table>
	</section>

	<section class="settings-section" aria-labelledby="about-components-title">
		<h3 id="about-components-title">Loaded components</h3>
		{#if components.length === 0}
			<p class="px-1 text-xs text-fg-faint">No component records yet.</p>
		{:else}
			<ul class="flex flex-col border-t border-line-subtle font-mono text-2xs">
				{#each components as component (`${component.module}:${component.message}`)}
					<li class="flex flex-col gap-1 border-b border-line-subtle p-1">
						<div class="flex items-baseline gap-2 text-fg">
							<span class="font-semibold text-fg-muted"
								>{component.module}</span>
							<span>{component.message}</span>
							<span class="ml-auto"
								><Timestamp value={component.recordedAt} /></span>
						</div>
						{#if component.attributes?.length}
							<dl class="flex flex-wrap gap-x-3 gap-y-0.5">
								{#each component.attributes as attribute (attribute.key)}
									<div class="flex gap-1">
										<dt class="text-fg-faint">{attribute.key}</dt>
										<dd class="text-fg">{attribute.value}</dd>
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

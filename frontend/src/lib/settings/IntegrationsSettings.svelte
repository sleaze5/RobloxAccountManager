<script lang="ts">
	import arrowsClockwiseIcon from "@phosphor-icons/core/regular/arrows-clockwise.svg?raw"
	import Icon from "../shared/Icon.svelte"
	import { untrack } from "svelte"
	import Select from "../shared/Select.svelte"
	import type { SettingsStore } from "./settings-store.svelte"

	let { store, query = "" }: { store: SettingsStore; query?: string } = $props()
	const terms = $derived(query.trim().toLowerCase().split(/\s+/).filter(Boolean)),
		visible = $derived(
			terms.every((term) =>
				"integrations rovalra rbxservers api private server browser servers information features enabled disabled on off preferred region country nearest launch join refresh regions domain domains data sent privacy apis.rovalra.com rbxservers.xyz".includes(
					term,
				),
			),
		),
		regions = $derived(store.serverRegions),
		code = $derived(store.roValraRegion),
		regionOptions = $derived([
			...(regions?.some((region) => region.code === code)
				? []
				: [
						{
							value: code,
							label: code
								? regions
									? `${code} (unavailable)`
									: code
								: store.serverRegionsFailed
									? "Regions unavailable"
									: regions
										? "Choose a region"
										: "Loading regions…",
						},
					]),
			...(regions ?? []).map((region) => ({
				value: region.code,
				label: region.name,
				group: region.country,
			})),
		])

	const codeClass = "rounded-sm bg-fill px-1 font-mono text-2xs text-fg-muted"

	$effect(() => {
		if (store.initialized && store.roValraEnabled && visible) {
			untrack(() => void store.loadServerRegions())
		}
	})
</script>

{#if visible}
	{#if store.error}<div class="notice-error" role="alert">
			{store.error}
		</div>{/if}

	<section class="settings-section" aria-labelledby="rovalra-title">
		<h3 id="rovalra-title">RoValra API</h3>
		<div class="settings-rows">
			<div class="settings-row">
				<div class="settings-row-copy">
					<strong id="rovalra-enabled">Enabled</strong>
					<span id="rovalra-description"
						>Enable this to show more server information in the server
						browser and add other features.</span>
				</div>
				<label class="switch">
					<input
						type="checkbox"
						role="switch"
						aria-labelledby="rovalra-title rovalra-enabled"
						aria-describedby="rovalra-description"
						checked={store.roValraEnabled}
						disabled={!store.initialized || store.busyRoValra}
						onchange={(event) => {
							const enabled = event.currentTarget.checked
							event.currentTarget.checked = store.roValraEnabled
							void store.setRoValraEnabled(enabled)
						}} />
					<span aria-hidden="true"></span>
				</label>
			</div>
			<div class="settings-row">
				<div class="settings-row-copy">
					<strong>Domain</strong>
					<span
						>While enabled, place IDs, server job IDs, and your region
						choice are sent to this domain.</span>
				</div>
				<code class={codeClass}>apis.rovalra.com</code>
			</div>
			{#if store.roValraEnabled}
				<div class="settings-row">
					<div class="settings-row-copy">
						<strong>Preferred region</strong>
						<span
							>The launch panel fills a server from this region, or from
							the closest region that has one.</span>
					</div>
					<div class="flex items-center gap-2">
						<Select
							className="w-56"
							label="Preferred region"
							value={store.roValraRegion}
							options={regionOptions}
							disabled={!store.initialized ||
								!regions?.length ||
								store.loadingServerRegions ||
								store.busyRoValraRegion}
							onChange={(region) => {
								if (region && region !== store.roValraRegion)
									void store.setRoValraRegion(region)
							}} />
						<button
							class="btn"
							type="button"
							aria-busy={store.loadingServerRegions}
							disabled={!store.initialized ||
								store.loadingServerRegions ||
								store.busyRoValraRegion ||
								store.busyRoValra}
							onclick={() => void store.loadServerRegions(true)}>
							<Icon
								svg={arrowsClockwiseIcon}
								data-turn
								class={store.loadingServerRegions
									? "spinner"
									: undefined}
								size={16}
								aria-hidden="true" />
							Refresh regions
						</button>
					</div>
				</div>
			{/if}
		</div>
	</section>

	<section class="settings-section" aria-labelledby="rbxservers-title">
		<h3 id="rbxservers-title">RBXServers</h3>
		<div class="settings-rows">
			<div class="settings-row">
				<div class="settings-row-copy">
					<strong id="rbxservers-enabled">Enabled</strong>
					<span id="rbxservers-description"
						>Enable this to list private servers that people share on
						RBXServers in the server browser.</span>
				</div>
				<label class="switch">
					<input
						type="checkbox"
						role="switch"
						aria-labelledby="rbxservers-title rbxservers-enabled"
						aria-describedby="rbxservers-description"
						checked={store.rbxServersEnabled}
						disabled={!store.initialized || store.busyRBXServers}
						onchange={(event) => {
							const enabled = event.currentTarget.checked
							event.currentTarget.checked = store.rbxServersEnabled
							void store.setRBXServersEnabled(enabled)
						}} />
					<span aria-hidden="true"></span>
				</label>
			</div>
			<div class="settings-row">
				<div class="settings-row-copy">
					<strong>Domain</strong>
					<span
						>While enabled, the place ID is sent to this domain when you
						open the RBXServers server list.</span>
				</div>
				<code class={codeClass}>rbxservers.xyz</code>
			</div>
		</div>
	</section>
{/if}

<script lang="ts">
	import arrowsClockwiseIcon from "@phosphor-icons/core/regular/arrows-clockwise.svg?raw"
	import Icon from "../shared/Icon.svelte"
	import ConnectionHosts from "./ConnectionHosts.svelte"
	import SectionLink from "./SectionLink.svelte"
	import { untrack } from "svelte"
	import Select from "../shared/Select.svelte"
	import type { SettingsStore } from "./settings-store.svelte"

	let { store, query = "" }: { store: SettingsStore; query?: string } = $props()
	const terms = $derived(query.trim().toLowerCase().split(/\s+/).filter(Boolean)),
		visible = $derived(
			terms.every((term) =>
				"integrations rovalra rbxservers api private server browser servers information features enabled disabled on off preferred region country nearest launch join refresh regions hosts connect domain data sent apis.rovalra.com rbxservers.xyz api.rbxservers.xyz".includes(
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
		<h3 class="flex items-center justify-between gap-2">
			<span id="rovalra-title">RoValra API</span>
			<SectionLink label="Open RoValra website" url="https://www.rovalra.com/" />
		</h3>
		<div class="settings-rows">
			<div class="settings-row">
				<div class="settings-row-copy">
					<strong id="rovalra-enabled">Server data</strong>
					<span id="rovalra-description"
						>Show server regions and ages, and fill the nearest server from
						the launch panel.</span>
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
							class="button"
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
			<ConnectionHosts hosts={["apis.rovalra.com"]} />
		</div>
	</section>

	<section class="settings-section" aria-labelledby="rbxservers-title">
		<h3 class="flex items-center justify-between gap-2">
			<span id="rbxservers-title">RBXServers</span>
			<SectionLink
				label="Open RBXServers website"
				url="https://rbxservers.xyz/" />
		</h3>
		<div class="settings-rows">
			<div class="settings-row">
				<div class="settings-row-copy">
					<strong id="rbxservers-enabled">Shared private servers</strong>
					<span id="rbxservers-description"
						>List private servers that people share on RBXServers. Opening
						the list sends the place ID. Getting a launch link sends the
						server ID.</span>
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
			<ConnectionHosts hosts={["rbxservers.xyz", "api.rbxservers.xyz"]} />
		</div>
	</section>
{/if}

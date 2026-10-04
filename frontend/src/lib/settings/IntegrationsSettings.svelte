<script lang="ts">
	import RefreshCw from "@lucide/svelte/icons/refresh-cw"
	import { untrack } from "svelte"
	import Select from "../shared/Select.svelte"
	import type { SettingsStore } from "./settings-store.svelte"

	let { store, query = "" }: { store: SettingsStore; query?: string } = $props()
	const terms = $derived(query.trim().toLowerCase().split(/\s+/).filter(Boolean)),
		visible = $derived(
			terms.every((term) =>
				"integrations rovalra api server browser servers information features enabled disabled on off preferred region country nearest launch join refresh regions".includes(
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
	{#if store.error}<div class="settings-inline-error" role="alert">
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
				<label class="settings-toggle">
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
					<div class="settings-region settings-row-actions">
						<Select
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
							class="settings-row-action"
							type="button"
							aria-busy={store.loadingServerRegions}
							disabled={!store.initialized ||
								store.loadingServerRegions ||
								store.busyRoValraRegion ||
								store.busyRoValra}
							onclick={() => void store.loadServerRegions(true)}>
							<RefreshCw
								class={store.loadingServerRegions
									? "spinner"
									: undefined}
								size={14}
								aria-hidden="true" />
							Refresh regions
						</button>
					</div>
				</div>
			{/if}
		</div>
	</section>
{/if}

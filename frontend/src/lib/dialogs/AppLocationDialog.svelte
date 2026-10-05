<script lang="ts">
	import { Application } from "@wailsio/runtime"
	import { accountBackend } from "../backend/bridge"
	import type { AppLocation } from "../backend/bridge"
	import {
		modalBackdropIn,
		modalBackdropOut,
		modalCardIn,
		modalCardOut,
		modalFocus,
	} from "../shared/presence"

	let { location, onConfirm }: { location: AppLocation; onConfirm: () => void } =
			$props(),
		busy = $state(false),
		error = $state("")

	const listedItems = $derived(location.otherItems ?? []),
		hiddenItems = $derived(location.otherItemCount - listedItems.length)

	async function confirm(): Promise<void> {
		busy = true
		error = ""
		try {
			await accountBackend.ConfirmAppLocation()
			onConfirm()
		} catch (cause) {
			const reason = cause instanceof Error ? `: ${cause.message}` : ""
			error = `Setup failed${reason}. Check the folder permissions.`
		} finally {
			busy = false
		}
	}
</script>

<div class="modal-backdrop vault-gate" in:modalBackdropIn out:modalBackdropOut>
	<div
		class="modal-card location-card"
		role="alertdialog"
		aria-modal="true"
		aria-labelledby="app-location-title"
		use:modalFocus
		in:modalCardIn
		out:modalCardOut>
		<div class="modal-heading compact-heading">
			<div>
				<h2 id="app-location-title">Set up in this folder?</h2>
				<p>
					The vault, settings, and logs are stored next to the app. You can
					move or rename the folder later.
				</p>
			</div>
		</div>
		<div class="location-path">{location.directory}</div>

		{#if location.otherItemCount > 0}
			<div class="modal-warning">
				This folder already contains other items. Use an empty folder that holds
				only the app.
				<ul class="location-items" aria-label="Items in this folder">
					{#each listedItems as item (item)}
						<li>{item}</li>
					{/each}
					{#if hiddenItems > 0}
						<li class="location-items-more">and {hiddenItems} more</li>
					{/if}
				</ul>
			</div>
		{/if}

		{#if error}
			<div class="modal-error" role="alert">{error}</div>
		{/if}

		<div class="modal-actions">
			<button
				type="button"
				disabled={busy}
				onclick={() => void Application.Quit()}>Quit</button>
			<button
				class="primary-action"
				type="button"
				disabled={busy}
				onclick={() => void confirm()}>
				{busy
					? "Setting up..."
					: location.otherItemCount > 0
						? "Use anyway"
						: "Use this folder"}
			</button>
		</div>
	</div>
</div>

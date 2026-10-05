<script lang="ts">
	import { Application } from "@wailsio/runtime"
	import {
		accountBackend,
		ChoiceReason,
		RootState,
		StorageMode,
	} from "../backend/bridge"
	import type { AppLocation, StorageCandidate } from "../backend/bridge"
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
		restarting = $state(false),
		error = $state(""),
		selected = $state<StorageMode | null>(null)

	const candidates = $derived(location.candidates ?? []),
		conflict = $derived(location.choice === ChoiceReason.ChoiceConflict),
		portable = $derived(
			candidates.find((candidate) => candidate.mode === StorageMode.ModePortable),
		),
		heading = $derived(
			conflict
				? "Choose which data to use"
				: location.choice === ChoiceReason.ChoicePortableWithoutVault
					? "Continue in this folder?"
					: "Choose where to store data",
		),
		summary = $derived(
			conflict
				? "App data exists next to the app and in your user folder. The other copy stays unchanged. Move or delete it to stop this question."
				: location.choice === ChoiceReason.ChoicePortableWithoutVault
					? portable?.state === RootState.RootIncomplete
						? "This folder has an incomplete vault. Keep using it to recover the vault, or start fresh in your user folder."
						: "This folder has app data but no vault. Keep using it, or start fresh in your user folder."
					: "The vault, settings, and logs are stored together in one folder.",
		),
		selectedCandidate = $derived(
			candidates.find((candidate) => candidate.mode === selected),
		),
		listedItems = $derived(selectedCandidate?.otherItems ?? []),
		otherItemCount = $derived(selectedCandidate?.otherItemCount ?? 0),
		hiddenItems = $derived(otherItemCount - listedItems.length)

	$effect.pre(() => {
		if (selected !== null || conflict) return
		// The first run prefers Portable only in a folder that holds just the app.
		selected =
			portable &&
			(location.choice === ChoiceReason.ChoicePortableWithoutVault ||
				portable.otherItemCount === 0)
				? StorageMode.ModePortable
				: (candidates[candidates.length - 1]?.mode ?? null)
	})

	function title(candidate: StorageCandidate): string {
		return candidate.mode === StorageMode.ModeStandard ? "Standard" : "Portable"
	}

	function description(candidate: StorageCandidate): string {
		if (conflict) {
			return candidate.state === RootState.RootVault
				? "Has a vault."
				: "Has an incomplete vault that can be recovered."
		}
		if (candidate.mode === StorageMode.ModeStandard) {
			return "In your user folder. Move or replace the app without moving data."
		}
		return location.choice === ChoiceReason.ChoicePortableWithoutVault
			? "Keep using this folder."
			: "Next to the app. Move the folder to take your data with you."
	}

	async function confirm(): Promise<void> {
		if (!selected) return
		busy = true
		error = ""
		try {
			restarting = await accountBackend.ConfirmAppLocation(selected)
			if (!restarting) onConfirm()
		} catch (cause) {
			const reason = cause instanceof Error ? `: ${cause.message}` : ""
			error = `Setup failed${reason}. Check the folder permissions.`
		} finally {
			busy = restarting
		}
	}
</script>

<div class="modal-backdrop vault-gate" in:modalBackdropIn out:modalBackdropOut>
	<div
		class="modal-card location-card"
		role="alertdialog"
		aria-modal="true"
		aria-labelledby="app-location-title"
		aria-describedby="app-location-summary"
		use:modalFocus
		in:modalCardIn
		out:modalCardOut>
		<div class="modal-heading compact-heading">
			<div>
				<h2 id="app-location-title">{heading}</h2>
				<p id="app-location-summary">{summary}</p>
			</div>
		</div>

		<div
			class="location-options"
			role="radiogroup"
			aria-labelledby="app-location-title">
			{#each candidates as candidate (candidate.mode)}
				<label
					class="settings-radio-option location-option"
					class:disabled={busy}>
					<span class="settings-radio">
						<input
							type="radio"
							name="app-location"
							value={candidate.mode}
							checked={selected === candidate.mode}
							disabled={busy}
							aria-describedby={`app-location-${candidate.mode}`}
							onchange={() => (selected = candidate.mode)} />
						<span aria-hidden="true"></span>
					</span>
					<span class="location-option-text">
						<span class="settings-radio-label">{title(candidate)}</span>
						<span
							class="location-option-description"
							id={`app-location-${candidate.mode}`}>
							{description(candidate)}
							<span class="location-option-path"
								>{candidate.directory}</span>
						</span>
					</span>
				</label>
			{/each}
		</div>

		{#if otherItemCount > 0}
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
				disabled={busy || !selected}
				onclick={() => void confirm()}>
				{restarting
					? "Restarting..."
					: busy
						? "Setting up..."
						: otherItemCount > 0
							? "Use anyway"
							: "Continue"}
			</button>
		</div>
	</div>
</div>

<script lang="ts">
	import checkBoldIcon from "@phosphor-icons/core/bold/check-bold.svg?raw"
	import Icon from "../shared/Icon.svelte"
	import Dialog from "../shared/Dialog.svelte"
	import { Application } from "@wailsio/runtime"
	import {
		accountBackend,
		ChoiceReason,
		RootState,
		StorageMode,
	} from "../backend/bridge"
	import type { AppLocation, StorageCandidate } from "../backend/bridge"

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

<Dialog
	class="max-w-md text-left"
	role="alertdialog"
	aria-labelledby="app-location-title"
	aria-describedby="app-location-summary"
	strong>
	<div class="dialog-heading mb-2">
		<div>
			<h2 id="app-location-title">{heading}</h2>
			<p id="app-location-summary">{summary}</p>
		</div>
	</div>

	<div
		class="flex flex-col gap-1"
		role="radiogroup"
		aria-labelledby="app-location-title">
		{#each candidates as candidate (candidate.mode)}
			<label
				class={[
					"relative flex cursor-pointer items-start gap-2 rounded-md border px-3 py-2 transition-colors duration-100 has-focus-visible:outline-1 has-focus-visible:outline-offset-1 has-focus-visible:outline-focus",
					selected === candidate.mode
						? "border-line-strong bg-active"
						: "border-line",
					busy ? "cursor-not-allowed opacity-45" : "hover:bg-hover",
				]}>
				<input
					class="pointer-events-none absolute opacity-0"
					type="radio"
					name="app-location"
					value={candidate.mode}
					checked={selected === candidate.mode}
					disabled={busy}
					aria-describedby={`app-location-${candidate.mode}`}
					onchange={() => (selected = candidate.mode)} />
				<span class="flex min-w-0 flex-1 flex-col gap-0.5">
					<span class="text-xs font-medium text-fg">{title(candidate)}</span>
					<span
						class="flex flex-col gap-0.5 text-xs leading-snug text-fg-muted"
						id={`app-location-${candidate.mode}`}>
						{description(candidate)}
						<span class="font-mono text-2xs wrap-anywhere text-fg-faint"
							>{candidate.directory}</span>
					</span>
				</span>
				<span
					class="inline-flex h-5 w-3.5 shrink-0 items-center text-fg"
					aria-hidden="true">
					{#if selected === candidate.mode}
						<Icon svg={checkBoldIcon} size={14} />
					{/if}
				</span>
			</label>
		{/each}
	</div>

	{#if otherItemCount > 0}
		<div class="mt-3 notice-warning">
			This folder already contains other items. Use an empty folder that holds
			only the app.
			<ul
				class="mt-2 grid gap-0.5 pl-4 font-mono text-2xs wrap-anywhere"
				aria-label="Items in this folder">
				{#each listedItems as entryName (entryName)}
					<li>{entryName}</li>
				{/each}
				{#if hiddenItems > 0}
					<li class="list-none font-sans">and {hiddenItems} more</li>
				{/if}
			</ul>
		</div>
	{/if}

	{#if error}
		<div class="mt-3 notice-error" role="alert">{error}</div>
	{/if}

	<div class="dialog-actions">
		<button
			class="button"
			type="button"
			disabled={busy}
			onclick={() => void Application.Quit()}>Quit</button>
		<button
			class="button-primary"
			type="button"
			disabled={busy || !selected}
			onclick={() => void confirm()}>
			{restarting
				? "Restarting…"
				: busy
					? "Setting up…"
					: otherItemCount > 0
						? "Use anyway"
						: "Continue"}
		</button>
	</div>
</Dialog>

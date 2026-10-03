<script lang="ts">
	import ExternalLink from "@lucide/svelte/icons/external-link"
	import Info from "@lucide/svelte/icons/info"
	import RotateCcw from "@lucide/svelte/icons/rotate-ccw"
	import Save from "@lucide/svelte/icons/save"
	import { Browser } from "@wailsio/runtime"
	import { MotionPreference } from "../backend/bridge"
	import Select from "../shared/Select.svelte"
	import { formatTimestamp, timestampFormats } from "../shared/timestamp"
	import type { SettingsStore } from "./settings-store.svelte"

	const maxFormatLength = 256,
		momentFormatDocsURL = "https://momentjs.com/docs/#/displaying/format/",
		motionOptions = [
			{
				value: MotionPreference.MotionSystem,
				label: "System",
				description: "Follow the operating system's animation setting.",
			},
			{
				value: MotionPreference.MotionReduced,
				label: "Reduced",
				description: "Turn off animations.",
			},
			{
				value: MotionPreference.MotionFull,
				label: "Full",
				description: "Always show animations.",
			},
		]

	let { store, query = "" }: { store: SettingsStore; query?: string } = $props(),
		timestampFormat = $state<string>(timestampFormats.shown),
		timestampHoverFormat = $state<string>(timestampFormats.tooltip),
		previewValue = $state(Date.now() - 16 * 60 * 60 * 1_000),
		previewNow = $state(Date.now()),
		normalizedQuery = $derived(query.trim().toLowerCase()),
		searchTerms = $derived(normalizedQuery.split(/\s+/).filter(Boolean)),
		previewDateTime = $derived(new Date(previewValue).toISOString()),
		preview = $derived(formatTimestamp(previewValue, timestampFormat, previewNow)),
		previewHover = $derived(
			formatTimestamp(previewValue, timestampHoverFormat, previewNow),
		),
		validationError = $derived(
			!timestampFormat.trim() || !timestampHoverFormat.trim()
				? "Enter both timestamp formats."
				: timestampFormat.length > maxFormatLength ||
					  timestampHoverFormat.length > maxFormatLength
					? "Timestamp formats cannot exceed 256 characters."
					: "",
		),
		changed = $derived(
			timestampFormat !== store.timestampFormat ||
				timestampHoverFormat !== store.timestampHoverFormat,
		),
		showMotion = $derived(
			matches(
				"Animations",
				"reduce reduced full system operating system animation effects",
				"motion",
			),
		),
		showTimestampFormat = $derived(
			matches("Timestamp format", "Moment.js tokens date time display format"),
		),
		showHoverFormat = $derived(
			matches("Hover format", "Moment.js tokens tooltip relative time"),
		),
		showPreview = $derived(
			matches("Preview", "display hover tooltip formatted timestamp"),
		),
		showTimestampActions = $derived(
			matches("Save restore defaults", "save reset timestamp formats"),
		)

	$effect(() => {
		if (!store.initialized) {
			return
		}
		timestampFormat = store.timestampFormat
		timestampHoverFormat = store.timestampHoverFormat
	})

	function matches(
		name: string,
		description: string,
		section = "timestamps",
	): boolean {
		const text = `user interface ${section} ${name} ${description}`.toLowerCase()
		return searchTerms.every((term) => text.includes(term))
	}

	function restoreDefaults(): void {
		timestampFormat = timestampFormats.shown
		timestampHoverFormat = timestampFormats.tooltip
	}

	function refreshPreview(): void {
		previewValue = Date.now()
		previewNow = previewValue
	}

	function openMomentFormatDocs(): void {
		void Browser.OpenURL(momentFormatDocsURL)
	}

	function saveFormats(): void {
		if (validationError || !changed) {
			return
		}
		void store.setTimestampFormats(timestampFormat, timestampHoverFormat)
	}
</script>

{#if showMotion}
	<section class="settings-section" aria-labelledby="motion-title">
		<h3 id="motion-title">Motion</h3>
		{#if store.error}
			<div class="settings-inline-error" role="alert">
				{store.error}
			</div>
		{/if}
		<div class="settings-rows">
			<div class="settings-row">
				<div class="settings-row-copy">
					<strong>Animations</strong>
					<span>Choose whether the interface animates.</span>
				</div>
				<Select
					label="Animations"
					value={store.motion}
					options={motionOptions}
					disabled={!store.initialized || store.busyMotion}
					onChange={(motion) => void store.setMotion(motion)} />
			</div>
		</div>
	</section>
{/if}

{#if showTimestampFormat || showHoverFormat || showPreview || showTimestampActions}
	<section class="settings-section" aria-labelledby="timestamps-title">
		<h3 id="timestamps-title">Timestamps</h3>
		<div class="settings-section-help">
			<Info size={13} aria-hidden="true" />
			<p>
				Format dates with
				<button type="button" onclick={openMomentFormatDocs}
					>Moment.js tokens<ExternalLink
						size={10}
						aria-hidden="true" /></button
				>. This app also supports <code>[relative]</code> for values such as “4 hours
				ago.”
			</p>
		</div>
		{#if store.error && !showMotion}
			<div class="settings-inline-error" role="alert">
				{store.error}
			</div>
		{/if}
		{#if validationError}
			<div class="settings-inline-error" role="alert">
				{validationError}
			</div>
		{/if}
		<form
			novalidate
			onsubmit={(event) => {
				event.preventDefault()
				saveFormats()
			}}>
			<div class="settings-rows">
				{#if showTimestampFormat}
					<div class="settings-row">
						<div class="settings-row-copy">
							<strong id="timestamp-format-label"
								>Timestamp format</strong>
							<span
								>Use Moment.js tokens such as YYYY, MMMM, and HH:mm.</span>
						</div>
						<input
							class="settings-format-input"
							type="text"
							aria-labelledby="timestamp-format-label"
							autocomplete="off"
							spellcheck="false"
							maxlength={maxFormatLength}
							disabled={!store.initialized || store.busyTimestampFormats}
							bind:value={timestampFormat} />
					</div>
				{/if}

				{#if showHoverFormat}
					<div class="settings-row">
						<div class="settings-row-copy">
							<strong id="timestamp-hover-format-label"
								>Hover format</strong>
							<span>Use [relative] to insert relative time.</span>
						</div>
						<input
							class="settings-format-input"
							type="text"
							aria-labelledby="timestamp-hover-format-label"
							autocomplete="off"
							spellcheck="false"
							maxlength={maxFormatLength}
							disabled={!store.initialized || store.busyTimestampFormats}
							bind:value={timestampHoverFormat} />
					</div>
				{/if}

				{#if showPreview}
					<div class="settings-row">
						<div class="settings-row-copy">
							<strong>Preview</strong>
							<span
								>Hover over the timestamp to preview the hover format.</span>
						</div>
						<button
							class="settings-timestamp-preview"
							type="button"
							aria-label={previewHover || "Timestamp hover preview"}
							data-tooltip={previewHover}
							data-tooltip-side="top"
							onclick={refreshPreview}>
							<time datetime={previewDateTime}
								>{preview || "No timestamp output"}</time>
						</button>
					</div>
				{/if}
			</div>

			{#if showTimestampActions}
				<div class="settings-form-actions">
					<button
						class="settings-row-action"
						type="button"
						disabled={store.busyTimestampFormats ||
							(timestampFormat === timestampFormats.shown &&
								timestampHoverFormat === timestampFormats.tooltip)}
						onclick={restoreDefaults}>
						<RotateCcw size={13} aria-hidden="true" />
						Restore defaults
					</button>
					<button
						class="settings-row-action settings-save-action"
						type="submit"
						disabled={!store.initialized ||
							store.busyTimestampFormats ||
							!changed ||
							Boolean(validationError)}>
						<Save size={13} aria-hidden="true" />
						{store.busyTimestampFormats ? "Saving..." : "Save changes"}
					</button>
				</div>
			{/if}
		</form>
	</section>
{/if}

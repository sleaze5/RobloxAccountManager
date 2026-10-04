<script lang="ts">
	import CaretRightIcon from "phosphor-svelte/lib/CaretRightIcon"
	import InfoIcon from "phosphor-svelte/lib/InfoIcon"
	import ArrowCounterClockwiseIcon from "phosphor-svelte/lib/ArrowCounterClockwiseIcon"
	import FloppyDiskIcon from "phosphor-svelte/lib/FloppyDiskIcon"
	import { accountBackend, MotionPreference } from "../backend/bridge"
	import type { TimestampFormat } from "../backend/bridge"
	import Select from "../shared/Select.svelte"
	import { defaultTimestampFormats, formatTimestamp } from "../shared/timestamp"
	import type { SettingsStore } from "./settings-store.svelte"

	type DraftFormat = { format: TimestampFormat | null; error: string }

	const maxFormatLength = 256,
		tokenReference = [
			{ token: "$YYYY", label: "Year", example: "2026" },
			{ token: "$YY", label: "Year, two digits", example: "26" },
			{ token: "$MMMM", label: "Month name", example: "March" },
			{ token: "$MMM", label: "Short month name", example: "Mar" },
			{ token: "$MM", label: "Month, two digits", example: "03" },
			{ token: "$M", label: "Month", example: "3" },
			{ token: "$DD", label: "Day, two digits", example: "04" },
			{ token: "$D", label: "Day", example: "4" },
			{ token: "$dddd", label: "Weekday", example: "Wednesday" },
			{ token: "$ddd", label: "Short weekday", example: "Wed" },
			{ token: "$HH", label: "24-hour hour, two digits", example: "21" },
			{ token: "$H", label: "24-hour hour", example: "21" },
			{ token: "$hh", label: "12-hour hour, two digits", example: "09" },
			{ token: "$h", label: "12-hour hour", example: "9" },
			{ token: "$mm", label: "Minute, two digits", example: "05" },
			{ token: "$m", label: "Minute", example: "5" },
			{ token: "$ss", label: "Second, two digits", example: "07" },
			{ token: "$s", label: "Second", example: "7" },
			{ token: "$SSS", label: "Millisecond, three digits", example: "042" },
			{ token: "$A", label: "AM or PM", example: "PM" },
			{ token: "$a", label: "am or pm", example: "pm" },
			{ token: "$relative", label: "Relative time", example: "4 hours ago" },
			{ token: "$$", label: "Dollar sign", example: "$" },
		],
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
		timestampFormat = $state<string>(defaultTimestampFormats.shown),
		timestampHoverFormat = $state<string>(defaultTimestampFormats.hover),
		shownDraft = $state<DraftFormat>({ format: null, error: "" }),
		hoverDraft = $state<DraftFormat>({ format: null, error: "" }),
		previewValue = $state(Date.now() - 16 * 60 * 60 * 1_000),
		previewNow = $state(Date.now()),
		normalizedQuery = $derived(query.trim().toLowerCase()),
		searchTerms = $derived(normalizedQuery.split(/\s+/).filter(Boolean)),
		previewDateTime = $derived(new Date(previewValue).toISOString()),
		preview = $derived(
			formatTimestamp(previewValue, shownDraft.format, previewNow),
		),
		previewHover = $derived(
			formatTimestamp(previewValue, hoverDraft.format, previewNow),
		),
		validationErrors = $derived(
			[
				shownDraft.error && `Timestamp format: ${shownDraft.error}`,
				hoverDraft.error && `Hover format: ${hoverDraft.error}`,
			].filter(Boolean),
		),
		changed = $derived(
			timestampFormat !== store.timestampFormat?.source ||
				timestampHoverFormat !== store.timestampHoverFormat?.source,
		),
		showMotion = $derived(
			matches(
				"Animations",
				"reduce reduced full system operating system animation effects",
				"motion",
			),
		),
		showTimestampFormat = $derived(
			matches(
				"Timestamp format",
				"tokens token reference dollar date time display format",
			),
		),
		showHoverFormat = $derived(
			matches(
				"Hover format",
				"tokens token reference dollar tooltip relative time",
			),
		),
		showPreview = $derived(
			matches("Preview", "display hover tooltip formatted timestamp"),
		),
		showTimestampActions = $derived(
			matches("Save restore defaults", "save reset timestamp formats"),
		)

	$effect(() => {
		if (!store.timestampFormat || !store.timestampHoverFormat) {
			return
		}
		timestampFormat = store.timestampFormat.source
		timestampHoverFormat = store.timestampHoverFormat.source
	})

	$effect(() => parseDraft(timestampFormat, (draft) => (shownDraft = draft)))
	$effect(() => parseDraft(timestampHoverFormat, (draft) => (hoverDraft = draft)))

	// The backend owns the format syntax. The returned cleanup ignores results for outdated input.
	function parseDraft(
		source: string,
		apply: (draft: DraftFormat) => void,
	): () => void {
		let current = true
		void (async () => {
			let draft: DraftFormat
			try {
				draft = {
					format: await accountBackend.ParseTimestampFormat(source),
					error: "",
				}
			} catch (error) {
				draft = {
					format: null,
					error:
						error instanceof Error
							? error.message
							: "The format could not be read.",
				}
			}
			if (current) apply(draft)
		})()
		return () => {
			current = false
		}
	}

	function matches(
		name: string,
		description: string,
		section = "timestamps",
	): boolean {
		const text = `user interface ${section} ${name} ${description}`.toLowerCase()
		return searchTerms.every((term) => text.includes(term))
	}

	function restoreDefaults(): void {
		timestampFormat = defaultTimestampFormats.shown
		timestampHoverFormat = defaultTimestampFormats.hover
	}

	function refreshPreview(): void {
		previewValue = Date.now()
		previewNow = previewValue
	}

	function saveFormats(): void {
		if (validationErrors.length > 0 || !changed) {
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
			<InfoIcon size={15} aria-hidden="true" />
			<p>
				Start each token with <code>$</code>, such as <code>$YYYY</code>. Other
				text appears as typed. Use <code>$$</code> to show a <code>$</code> sign.
			</p>
		</div>
		<details class="settings-token-reference">
			<summary
				><CaretRightIcon size={14} aria-hidden="true" />Token reference</summary>
			<p>Examples use Wednesday, March 4, 2026, at 9:05:07.042 PM.</p>
			<dl>
				{#each tokenReference as item (item.token)}
					<div>
						<dt><code>{item.token}</code></dt>
						<dd>{item.label}<span>{item.example}</span></dd>
					</div>
				{/each}
			</dl>
		</details>
		{#if store.error && !showMotion}
			<div class="settings-inline-error" role="alert">
				{store.error}
			</div>
		{/if}
		{#each validationErrors as error (error)}
			<div class="settings-inline-error" role="alert">
				{error}
			</div>
		{/each}
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
							<span>Use tokens such as $YYYY, $MMMM, and $HH:$mm.</span>
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
							<span
								>Use $relative to show relative time, such as “4 hours
								ago”.</span>
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
							(timestampFormat === defaultTimestampFormats.shown &&
								timestampHoverFormat === defaultTimestampFormats.hover)}
						onclick={restoreDefaults}>
						<ArrowCounterClockwiseIcon size={15} aria-hidden="true" />
						Restore defaults
					</button>
					<button
						class="settings-row-action settings-save-action"
						type="submit"
						disabled={!store.initialized ||
							store.busyTimestampFormats ||
							!changed ||
							validationErrors.length > 0}>
						<FloppyDiskIcon size={15} aria-hidden="true" />
						{store.busyTimestampFormats ? "Saving..." : "Save changes"}
					</button>
				</div>
			{/if}
		</form>
	</section>
{/if}

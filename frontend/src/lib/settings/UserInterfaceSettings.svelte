<script lang="ts">
	import { matchesSearch, searchTerms } from "../shared/search"
	import arrowCounterClockwiseIcon from "@phosphor-icons/core/regular/arrow-counter-clockwise.svg?raw"
	import caretRightIcon from "@phosphor-icons/core/regular/caret-right.svg?raw"
	import floppyDiskIcon from "@phosphor-icons/core/regular/floppy-disk.svg?raw"
	import infoIcon from "@phosphor-icons/core/regular/info.svg?raw"
	import Icon from "../shared/Icon.svelte"
	import { accountBackend, MotionPreference } from "../backend/bridge"
	import type { TimestampFormat } from "../backend/bridge"
	import Select from "../shared/Select.svelte"
	import { defaultTimestampFormats, formatTimestamp } from "../shared/timestamp"
	import type { SettingsStore } from "./settings-store.svelte"
	import { errorMessage } from "../shared/errors"

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
		terms = $derived(searchTerms(query)),
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
					error: errorMessage(error, "The format could not be read."),
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
		return matchesSearch(terms, `user interface ${section} ${name} ${description}`)
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
	const codeClass = "rounded-sm bg-fill px-0.5 font-mono text-2xs text-fg-muted"
</script>

{#if showMotion}
	<section class="settings-section" aria-labelledby="motion-title">
		<h3 id="motion-title">Motion</h3>
		{#if store.error}
			<div class="notice-error" role="alert">
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
		<div class="flex items-start gap-1 text-xs leading-normal text-fg-muted">
			<Icon svg={infoIcon} class="mt-px shrink-0" size={14} aria-hidden="true" />
			<p>
				Start each token with <code class={codeClass}>$</code>, such as
				<code class={codeClass}>$YYYY</code>. Other text appears as typed. Use
				<code class={codeClass}>$$</code>
				to show a <code class={codeClass}>$</code> sign.
			</p>
		</div>
		<details class="group text-xs text-fg-muted">
			<summary
				class="inline-flex items-center gap-1 rounded-sm transition-colors duration-100 hover:text-fg focus-visible:text-fg focus-visible:outline-offset-2"
				><Icon
					svg={caretRightIcon}
					class="transition-transform duration-100 group-open:rotate-90"
					size={14}
					aria-hidden="true" />Token reference</summary>
			<p class="mt-2 mb-1">
				Examples use Wednesday, March 4, 2026, at 9:05:07.042 PM.
			</p>
			<dl class="grid grid-cols-2 gap-x-4 gap-y-0.5">
				{#each tokenReference as reference (reference.token)}
					<div class="flex items-baseline gap-2">
						<dt class="w-16 shrink-0">
							<code class={codeClass}>{reference.token}</code>
						</dt>
						<dd class="flex min-w-0 flex-1 justify-between gap-2">
							{reference.label}<span
								class="font-mono text-2xs whitespace-nowrap text-fg-faint"
								>{reference.example}</span>
						</dd>
					</div>
				{/each}
			</dl>
		</details>
		{#if store.error && !showMotion}
			<div class="notice-error" role="alert">
				{store.error}
			</div>
		{/if}
		{#each validationErrors as error (error)}
			<div class="notice-error" role="alert">
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
							class="field w-72 font-mono text-2xs"
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
							class="field w-72 font-mono text-2xs"
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
							class="max-w-72 truncate rounded-md border border-line px-2 py-1 text-xs font-medium text-fg-muted transition-colors duration-100 hover:border-line-strong hover:bg-hover hover:text-fg focus-visible:border-line-strong focus-visible:bg-hover focus-visible:text-fg"
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
				<div class="flex justify-end gap-2 px-3 pt-1 pb-3">
					<button
						class="button"
						type="button"
						disabled={store.busyTimestampFormats ||
							(timestampFormat === defaultTimestampFormats.shown &&
								timestampHoverFormat === defaultTimestampFormats.hover)}
						onclick={restoreDefaults}>
						<Icon
							svg={arrowCounterClockwiseIcon}
							size={16}
							aria-hidden="true" />
						Restore defaults
					</button>
					<button
						class="button-primary"
						type="submit"
						disabled={!store.initialized ||
							store.busyTimestampFormats ||
							!changed ||
							validationErrors.length > 0}>
						<Icon svg={floppyDiskIcon} size={16} aria-hidden="true" />
						{store.busyTimestampFormats ? "Saving…" : "Save changes"}
					</button>
				</div>
			{/if}
		</form>
	</section>
{/if}

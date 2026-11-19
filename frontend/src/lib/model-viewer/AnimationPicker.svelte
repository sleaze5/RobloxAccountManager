<script lang="ts">
	import caretDownIcon from "@phosphor-icons/core/regular/caret-down.svg?raw"
	import checkIcon from "@phosphor-icons/core/regular/check.svg?raw"
	import circleNotchIcon from "@phosphor-icons/core/regular/circle-notch.svg?raw"
	import magnifyingGlassIcon from "@phosphor-icons/core/regular/magnifying-glass.svg?raw"
	import personSimpleRunIcon from "@phosphor-icons/core/regular/person-simple-run.svg?raw"
	import xIcon from "@phosphor-icons/core/regular/x.svg?raw"
	import { tick } from "svelte"
	import Icon from "../shared/Icon.svelte"
	import ProfileImage from "../shared/ProfileImage.svelte"
	import { floatMenu, focusMenuItemForKey, menuIn, menuOut } from "../shared/presence"
	import { matchesSearch, searchTerms } from "../shared/search"
	import type {
		AnimationChoice,
		AnimationChoiceGroup,
		AnimationLibrary,
		AnimationPlayback,
		OwnedAnimations,
	} from "./model-subject"

	// AnimationPicker selects what the avatar of a 3D viewer plays: its
	// animation set, its emotes, and, with a library, what the user owns and
	// any emote or animation that the user names by ID or link. The panel
	// keeps one height while its lists load, so that it stays at its trigger.
	let {
		playback,
		library,
		className,
	}: {
		playback: AnimationPlayback | undefined
		library?: AnimationLibrary
		className: string
	} = $props()

	interface PickerSection {
		heading: string
		choices: AnimationChoice[]
		owned?: OwnedAnimations
		noun?: string
	}

	const panelId = $props.id(),
		headingId = `${panelId}-heading`,
		referenceId = `${panelId}-reference`,
		referenceErrorId = `${panelId}-reference-error`,
		headingByGroup: Record<AnimationChoiceGroup, string> = {
			animationSet: "Animations",
			emote: "Emotes",
			added: "Added",
			ownedEmote: "Owned emotes",
			ownedAnimation: "Owned animations",
		}

	let isOpen = $state(false),
		query = $state(""),
		reference = $state(""),
		trigger = $state<HTMLButtonElement>(),
		panel = $state<HTMLDivElement>(),
		searchInput = $state<HTMLInputElement>(),
		list = $state<HTMLDivElement>()

	const choices = $derived(playback?.animationChoices ?? []),
		selectedChoice = $derived(
			choices.find((choice) => choice.id === playback?.animationId),
		),
		terms = $derived(searchTerms(query)),
		sections = $derived.by(() => {
			const avatarChoiceIds = new Set(
					choices
						.filter(
							(choice) =>
								choice.group !== "ownedEmote" &&
								choice.group !== "ownedAnimation",
						)
						.map((choice) => choice.id),
				),
				isShown = (choice: AnimationChoice): boolean =>
					matchesSearch(terms, `${choice.label} ${choice.detail ?? ""}`),
				sectionOf = (group: AnimationChoiceGroup): PickerSection => ({
					heading: headingByGroup[group],
					choices: choices.filter(
						(choice) => choice.group === group && isShown(choice),
					),
				}),
				ownedSectionOf = (
					group: AnimationChoiceGroup,
					owned: OwnedAnimations,
					noun: string,
				): PickerSection => ({
					heading: headingByGroup[group],
					// An owned emote on the emote wheel shows once, under Emotes.
					choices: (owned.choices ?? []).filter(
						(choice) => !avatarChoiceIds.has(choice.id) && isShown(choice),
					),
					owned,
					noun,
				})
			const listedSections = [
				sectionOf("animationSet"),
				sectionOf("emote"),
				sectionOf("added"),
			]
			if (library) {
				listedSections.push(
					ownedSectionOf("ownedEmote", library.ownedEmotes, "owned emotes"),
					ownedSectionOf(
						"ownedAnimation",
						library.ownedAnimations,
						"owned animations",
					),
				)
			}
			return listedSections.filter(
				(section) =>
					section.choices.length > 0 || (section.owned && terms.length === 0),
			)
		}),
		shownChoices = $derived(sections.flatMap((section) => section.choices)),
		hasMatches = $derived(shownChoices.length > 0),
		// The list is one tab stop: the selected choice, or the first one.
		tabStopChoiceId = $derived(
			shownChoices.find((choice) => choice.id === playback?.animationId)?.id ??
				shownChoices[0]?.id,
		),
		triggerLabel = $derived(
			selectedChoice
				? `Animation: ${selectedChoice.label}${selectedChoice.detail ? `, ${selectedChoice.detail}` : ""}`
				: "Animation",
		)

	$effect(() => {
		if (!playback) isOpen = false
	})

	async function open(): Promise<void> {
		isOpen = true
		library?.loadOwned()
		await tick()
		searchInput?.focus()
		// The panel lays out in the top layer on the next frame, before which
		// its list cannot scroll.
		requestAnimationFrame(() => {
			list?.querySelector(
				"[role='option'][aria-selected='true']",
			)?.scrollIntoView({ block: "nearest" })
		})
	}

	function close(shouldRestoreFocus: boolean): void {
		isOpen = false
		query = ""
		if (shouldRestoreFocus) trigger?.focus()
	}

	// choose plays the choice at once, because the user picked it to see it.
	function choose(choice: AnimationChoice): void {
		playback?.chooseAnimation(choice)
		playback?.play()
		close(true)
	}

	async function playReference(): Promise<void> {
		if (!library || !reference.trim()) return
		const choice = await library.findAnimation(reference)
		if (!choice) return
		reference = ""
		choose(choice)
	}

	function options(): HTMLButtonElement[] {
		return Array.from(
			list?.querySelectorAll<HTMLButtonElement>("[role='option']") ?? [],
		)
	}

	function handleListKeydown(event: KeyboardEvent): void {
		focusMenuItemForKey(event, options())
	}
</script>

<!-- Escape closes the panel before it reaches a dialog around the viewer, and
a press outside the panel and its trigger closes it. -->
<svelte:window
	onkeydowncapture={(event) => {
		if (
			!isOpen ||
			event.key !== "Escape" ||
			!(event.target instanceof Node) ||
			!panel?.contains(event.target)
		) {
			return
		}
		event.preventDefault()
		event.stopImmediatePropagation()
		close(true)
	}}
	onpointerdown={(event) => {
		if (
			isOpen &&
			event.target instanceof Node &&
			!panel?.contains(event.target) &&
			!trigger?.contains(event.target)
		) {
			close(false)
		}
	}} />

<button
	bind:this={trigger}
	class={[
		"flex h-7 items-center justify-between gap-2 rounded-md border border-line bg-transparent px-2 text-xs font-medium text-fg-muted transition-colors duration-100 hover:border-line-strong hover:bg-hover hover:text-fg focus-visible:border-line-focus aria-expanded:border-line-strong aria-expanded:bg-hover aria-expanded:text-fg",
		className,
	]}
	type="button"
	aria-label={triggerLabel}
	aria-haspopup="dialog"
	aria-expanded={isOpen}
	aria-controls={isOpen ? panelId : undefined}
	disabled={!playback}
	onclick={() => {
		if (isOpen) close(false)
		else void open()
	}}
	onkeydown={(event) => {
		if (event.key === "ArrowDown" || event.key === "ArrowUp") {
			event.preventDefault()
			void open()
		}
	}}>
	<span class="flex min-w-0 items-baseline gap-1.5">
		<span class="truncate">{selectedChoice?.label ?? "Animation"}</span>
		{#if selectedChoice?.detail}
			<span class="truncate text-2xs font-normal text-fg-faint"
				>{selectedChoice.detail}</span>
		{/if}
	</span>
	<Icon
		svg={caretDownIcon}
		class={["shrink-0 transition-transform duration-100", isOpen && "rotate-180"]}
		size={16}
		aria-hidden="true" />
</button>

{#if isOpen && trigger}
	<div
		bind:this={panel}
		id={panelId}
		class="fixed flex w-80 flex-col overflow-hidden rounded-xl border border-line-strong bg-overlay shadow-menu"
		role="dialog"
		aria-labelledby={headingId}
		data-closes-on-escape
		use:floatMenu={{ anchor: trigger, align: "start" }}
		in:menuIn
		out:menuOut>
		<h2 id={headingId} class="sr-only">Select an animation</h2>
		<div class="p-2 pb-1">
			<div class="search-field">
				<Icon svg={magnifyingGlassIcon} size={16} aria-hidden="true" />
				<input
					bind:this={searchInput}
					bind:value={query}
					type="search"
					aria-label="Search animations and emotes"
					aria-controls={`${panelId}-list`}
					placeholder="Search animations and emotes"
					onkeydown={(event) => {
						if (event.key === "ArrowDown") {
							event.preventDefault()
							options()[0]?.focus()
						}
					}} />
				{#if query}
					<button
						type="button"
						aria-label="Clear search"
						onclick={() => {
							query = ""
							searchInput?.focus()
						}}>
						<Icon svg={xIcon} size={14} aria-hidden="true" />
					</button>
				{/if}
			</div>
		</div>

		<div
			bind:this={list}
			id={`${panelId}-list`}
			class="h-64 scrollbar-stable overflow-y-auto overscroll-contain px-1 pb-1"
			role="listbox"
			tabindex="-1"
			aria-label="Animations and emotes"
			onkeydown={handleListKeydown}>
			{#each sections as section (section.heading)}
				<div role="group" aria-label={section.heading}>
					<div class="menu-heading" aria-hidden="true">{section.heading}</div>
					{#each section.choices as choice (choice.id)}
						{@const isSelected = choice.id === playback?.animationId}
						<button
							class={[
								"flex w-full items-center justify-between gap-2 rounded-md px-2 text-left text-xs font-medium text-fg-muted transition-colors duration-100 hover:bg-hover hover:text-fg focus-visible:bg-hover focus-visible:text-fg aria-selected:bg-hover aria-selected:text-fg",
								choice.group === "animationSet" ? "h-7" : "h-10 pl-1.5",
							]}
							type="button"
							role="option"
							aria-label={choice.detail
								? `${choice.label}, ${choice.detail}`
								: choice.label}
							aria-selected={isSelected}
							tabindex={choice.id === tabStopChoiceId ? 0 : -1}
							onclick={() => choose(choice)}>
							<span class="flex min-w-0 items-center gap-2">
								{#if choice.group === "animationSet"}
									<Icon
										svg={personSimpleRunIcon}
										class="shrink-0 text-fg-faint"
										size={16}
										aria-hidden="true" />
								{:else}
									<span
										class="relative grid size-7 shrink-0 place-items-center overflow-hidden rounded-md border border-line-strong bg-raised"
										aria-hidden="true"
										><ProfileImage
											url={choice.imageUrl ?? ""}
											fallbackIcon={personSimpleRunIcon} /></span>
								{/if}
								<span class="flex min-w-0 flex-col leading-snug">
									<span class="truncate">{choice.label}</span>
									{#if choice.detail}
										<small
											class="truncate text-2xs font-normal text-fg-muted"
											>{choice.detail}</small>
									{/if}
								</span>
							</span>
							{#if isSelected}
								<Icon
									svg={checkIcon}
									class="shrink-0"
									size={16}
									aria-hidden="true" />
							{/if}
						</button>
					{/each}
					{#if section.owned && section.choices.length === 0 && terms.length === 0}
						{@const owned = section.owned}
						{#if owned.isLoading || (!owned.choices && !owned.error)}
							<p
								class="flex h-7 items-center gap-2 px-2 text-xs text-fg-muted"
								role="status">
								<Icon
									svg={circleNotchIcon}
									class="spinner"
									size={14}
									aria-hidden="true" />
								Loading {section.noun}…
							</p>
						{:else if owned.error}
							<div
								class="flex flex-wrap items-center gap-x-2 px-2 py-1 text-xs text-fg-muted"
								role="alert">
								<span class="min-w-0">{owned.error}</span>
								<button
									class="button-text"
									type="button"
									onclick={() => library?.loadOwned()}
									>Try again</button>
							</div>
						{:else}
							<p class="flex h-7 items-center px-2 text-xs text-fg-muted">
								None
							</p>
						{/if}
					{/if}
				</div>
			{/each}
			{#if !hasMatches && terms.length > 0}
				<div
					class="flex flex-col items-start gap-1 px-2 py-3 text-xs text-fg-muted">
					<p>No matching animations or emotes</p>
					<button
						class="button-text"
						type="button"
						onclick={() => {
							query = ""
							searchInput?.focus()
						}}>Clear search</button>
				</div>
			{/if}
		</div>

		{#if library}
			<form
				class="flex flex-col gap-1.5 border-t border-line-subtle p-2"
				novalidate
				onsubmit={(event) => {
					event.preventDefault()
					void playReference()
				}}>
				<label class="text-xs text-fg-muted" for={referenceId}
					>Play by ID or link</label>
				<div class="flex items-center gap-2">
					<input
						id={referenceId}
						class="field font-mono text-xs"
						type="text"
						autocomplete="off"
						spellcheck="false"
						placeholder="Emote or animation ID"
						aria-invalid={!!library.findError || undefined}
						aria-describedby={library.findError
							? referenceErrorId
							: undefined}
						disabled={library.isFinding}
						bind:value={reference}
						oninput={() => library?.clearFindError()} />
					<button
						class="button shrink-0"
						type="submit"
						aria-busy={library.isFinding}
						disabled={!reference.trim() || library.isFinding}
						>{library.isFinding ? "Finding…" : "Play"}</button>
				</div>
				{#if library.findError}
					<p
						id={referenceErrorId}
						class="text-xs text-danger-muted"
						role="alert">
						{library.findError}
					</p>
				{/if}
			</form>
		{/if}
	</div>
{/if}

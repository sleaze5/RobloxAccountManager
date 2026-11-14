<script lang="ts">
	import type { AvatarBodyColors } from "../backend/bridge"
	import { bodyParts } from "./avatar-draft"
	import type { BodyPart } from "./avatar-draft"

	let {
		bodyColors,
		selectedParts,
		onSelectPart,
	}: {
		bodyColors: AvatarBodyColors
		// selectedParts and onSelectPart are undefined when the figure only
		// shows the colors.
		selectedParts?: Set<BodyPart>
		onSelectPart?: (part: BodyPart, isAdding: boolean) => void
	} = $props()

	// The figure faces the viewer, so the right arm and leg are on the left.
	const partClasses: Record<BodyPart, string> = {
			head: "col-span-4 mb-0.5 size-10 justify-self-center rounded-md",
			rightArm: "h-16 w-8 rounded-sm",
			torso: "col-span-2 h-16 w-full rounded-sm",
			leftArm: "h-16 w-8 rounded-sm",
			rightLeg: "col-start-2 h-16 w-8 rounded-sm",
			leftLeg: "h-16 w-8 rounded-sm",
		},
		figureOrder: BodyPart[] = [
			"head",
			"rightArm",
			"torso",
			"leftArm",
			"rightLeg",
			"leftLeg",
		],
		labelByPart = new Map(bodyParts.map(({ part, label }) => [part, label]))
</script>

<div
	class="grid w-fit shrink-0 grid-cols-4 gap-0.5"
	role={onSelectPart ? "group" : "img"}
	aria-label={onSelectPart ? "Body parts" : "Body colors"}>
	{#each figureOrder as part (part)}
		{@const hexColor = bodyColors[part]}
		{@const label = labelByPart.get(part) ?? part}
		{#if onSelectPart}
			<button
				class={[
					partClasses[part],
					"border border-line-strong outline-offset-2 transition-colors duration-100 hover:border-line-focus focus-visible:outline-2 focus-visible:outline-focus data-selected:ring-2 data-selected:ring-fg-bright data-selected:ring-offset-2 data-selected:ring-offset-canvas",
				]}
				style:background-color={hexColor ? `#${hexColor}` : undefined}
				type="button"
				aria-label={label}
				aria-pressed={selectedParts?.has(part) ?? false}
				data-selected={selectedParts?.has(part) || undefined}
				data-tooltip={hexColor ? `${label} #${hexColor}` : label}
				onclick={(event) =>
					onSelectPart(
						part,
						event.ctrlKey || event.shiftKey || event.metaKey,
					)}></button>
		{:else}
			<span
				class={[
					partClasses[part],
					"border border-line-strong transition-colors duration-100",
				]}
				style:background-color={hexColor ? `#${hexColor}` : undefined}></span>
		{/if}
	{/each}
</div>

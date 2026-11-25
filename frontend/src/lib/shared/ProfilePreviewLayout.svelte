<script lang="ts" module>
	import type { Snippet } from "svelte"

	// ProfilePreviewConfirmation asks before a preview runs an action that
	// ends something, such as leaving a group. run returns whether it
	// succeeded, which closes the question.
	export interface ProfilePreviewConfirmation {
		title: string
		description: string
		confirmLabel: string
		progressLabel: string
		tone: "danger" | "warning"
		run: () => Promise<boolean>
	}

	// ProfilePreviewFrame is what the preview dialog gives the preview of each
	// kind of profile: the ID of its heading, the actions at its top right,
	// the way back to the preview that opened it, and the way to confirm an
	// action.
	export interface ProfilePreviewFrame {
		titleId: string
		actions: Snippet
		onBack: (() => void) | null
		confirm: (confirmation: ProfilePreviewConfirmation) => void
	}
</script>

<script lang="ts">
	import caretLeftIcon from "@phosphor-icons/core/regular/caret-left.svg?raw"
	import Icon from "./Icon.svelte"

	// ProfilePreviewLayout lays out the preview of a profile as the profile
	// lays out its header: the image, then the name with the actions at the
	// top right, and the summary below the name. The information follows.
	// quickActions are the profile's own actions, such as joining a group,
	// which come before the actions of the dialog.
	let {
		frame,
		image,
		title,
		summary,
		quickActions,
		children,
	}: {
		frame: ProfilePreviewFrame
		image: Snippet
		title: Snippet
		summary: Snippet
		quickActions?: Snippet
		children?: Snippet
	} = $props()
</script>

<div class="flex flex-col gap-5">
	<div class="flex items-start gap-4">
		{@render image()}
		<div class="flex min-w-0 flex-1 flex-col gap-1">
			<div class="flex items-start gap-3">
				{#if frame.onBack}
					<button
						class="-ml-1 button-ghost"
						type="button"
						aria-label="Back"
						data-tooltip="Back"
						onclick={frame.onBack}>
						<Icon
							svg={caretLeftIcon}
							data-nudge="left"
							size={16}
							aria-hidden="true" />
					</button>
				{/if}
				<h2
					id={frame.titleId}
					class="min-w-0 flex-1 leading-7 tracking-tight wrap-anywhere">
					{@render title()}
				</h2>
				<div class="flex shrink-0 items-center gap-1">
					{#if quickActions}
						{@render quickActions()}
						<span class="mx-1 h-4 w-px bg-line-strong" aria-hidden="true"
						></span>
					{/if}
					{@render frame.actions()}
				</div>
			</div>
			{@render summary()}
		</div>
	</div>
	{@render children?.()}
</div>

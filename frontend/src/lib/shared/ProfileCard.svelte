<script lang="ts">
	import starFillIcon from "@phosphor-icons/core/fill/star-fill.svg?raw"
	import caretRightIcon from "@phosphor-icons/core/regular/caret-right.svg?raw"
	import sealCheckIcon from "@phosphor-icons/core/regular/seal-check.svg?raw"
	import usersThreeIcon from "@phosphor-icons/core/regular/users-three.svg?raw"
	import type { CreatorKind } from "./creator-images"
	import Icon from "./Icon.svelte"
	import type { LabeledBadge } from "./labeled-badge"
	import LabeledBadges from "./LabeledBadges.svelte"
	import ProfileImage from "./ProfileImage.svelte"

	// ProfileCard shows a Roblox user or group: its image, its name, a line
	// below the name, and badges. With an action, the whole card is a button.
	let {
		kind,
		imageUrl,
		title,
		subtitle,
		hasVerifiedBadge = false,
		isFavorite = false,
		badges = [],
		action,
	}: {
		kind: CreatorKind
		imageUrl: string
		title: string
		// subtitle is the username of a user, or a short fact about a group.
		subtitle: string
		hasVerifiedBadge?: boolean
		// isFavorite marks a saved account with the Favorite tag.
		isFavorite?: boolean
		badges?: LabeledBadge[]
		action?: { label: string; open: () => void }
	} = $props()

	const cardClass =
		"flex h-full w-full min-w-0 flex-col gap-2 rounded-lg border border-line p-2 text-left"
</script>

{#snippet content()}
	<span class="flex min-w-0 items-center gap-2">
		<span
			class="grid size-8 shrink-0 place-items-center overflow-hidden rounded-md border border-line-strong bg-raised"
			aria-hidden="true">
			<ProfileImage
				url={imageUrl}
				fallbackIcon={kind === "Group" ? usersThreeIcon : undefined} />
		</span>
		<span class="flex min-w-0 flex-1 flex-col leading-snug">
			<span class="flex min-w-0 items-center gap-1">
				<strong class="truncate text-xs font-semibold text-fg">{title}</strong>
				{#if hasVerifiedBadge}<Icon
						svg={sealCheckIcon}
						class="shrink-0 text-fg-muted"
						size={12}
						aria-label={kind === "Group"
							? "Verified group"
							: "Verified user"} />{/if}
				{#if isFavorite}<Icon
						svg={starFillIcon}
						class="shrink-0 animate-pop text-accent-strong"
						size={12}
						aria-label="Favorite" />{/if}
			</span>
			<small class="truncate text-2xs text-fg-muted">{subtitle}</small>
		</span>
		{#if action}<Icon
				svg={caretRightIcon}
				class="shrink-0 text-fg-faint"
				data-nudge="right"
				size={14}
				aria-hidden="true" />{/if}
	</span>
	{#if badges.length > 0}
		<span class="flex min-w-0 flex-wrap items-center gap-1">
			<LabeledBadges {badges} />
		</span>
	{/if}
{/snippet}

{#if action}
	<button
		class={[
			cardClass,
			"transition-colors duration-100 hover:border-line-strong hover:bg-hover",
		]}
		type="button"
		data-tooltip={action.label}
		data-tooltip-strong={title}
		onclick={action.open}>
		{@render content()}
		<span class="sr-only">{action.label}</span>
	</button>
{:else}
	<div class={cardClass}>
		{@render content()}
	</div>
{/if}

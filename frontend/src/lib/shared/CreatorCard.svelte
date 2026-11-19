<script lang="ts">
	import caretRightIcon from "@phosphor-icons/core/regular/caret-right.svg?raw"
	import sealCheckIcon from "@phosphor-icons/core/regular/seal-check.svg?raw"
	import usersThreeIcon from "@phosphor-icons/core/regular/users-three.svg?raw"
	import { getCreatorNavigation } from "./creator-navigation"
	import { loadCreatorImage } from "./creator-images"
	import type { CreatorKind } from "./creator-images"
	import Icon from "./Icon.svelte"
	import ProfileImage from "./ProfileImage.svelte"

	let {
		kind,
		creatorId,
		name,
		displayName = "",
		hasVerifiedBadge,
		role,
	}: {
		kind: CreatorKind
		creatorId: number
		// name is the username of a user or the name of a group.
		name: string
		// displayName is the display name of a user, when the page knows it.
		displayName?: string
		hasVerifiedBadge: boolean
		// role names what the creator is to the item, such as Creator or Owner.
		role: string
	} = $props()

	const navigation = getCreatorNavigation()
	let imageUrl = $state("")

	const isGroup = $derived(kind === "Group"),
		title = $derived(isGroup ? name : displayName || name),
		actionLabel = $derived(isGroup ? "Show in Groups" : "Show in Social")

	$effect(() => {
		const requestedKind = kind,
			requestedCreatorId = creatorId
		imageUrl = ""
		let isCurrent = true
		void loadCreatorImage(requestedKind, requestedCreatorId).then((loadedUrl) => {
			if (isCurrent) imageUrl = loadedUrl
			return loadedUrl
		})
		return () => {
			isCurrent = false
		}
	})

	function open(): void {
		if (isGroup) navigation.showGroup(creatorId)
		else
			navigation.showUser({
				userId: creatorId,
				username: name,
				displayName: displayName || name,
			})
	}
</script>

<button
	class="group/creator flex w-full max-w-sm min-w-0 items-center gap-2 rounded-lg border border-line bg-surface p-2 text-left transition-colors duration-100 hover:border-line-strong hover:bg-hover"
	type="button"
	data-tooltip={actionLabel}
	data-tooltip-strong={title}
	onclick={open}>
	<span
		class="grid size-8 shrink-0 place-items-center overflow-hidden rounded-md border border-line-strong bg-raised"
		aria-hidden="true">
		{#if isGroup}
			<ProfileImage url={imageUrl} fallbackIcon={usersThreeIcon} />
		{:else}
			<ProfileImage url={imageUrl} />
		{/if}
	</span>
	<span class="flex min-w-0 flex-1 flex-col leading-snug">
		<span class="flex min-w-0 items-center gap-1">
			<strong class="truncate text-xs font-semibold text-fg">{title}</strong>
			{#if hasVerifiedBadge}<Icon
					svg={sealCheckIcon}
					class="shrink-0 text-fg-muted"
					size={12}
					aria-label={isGroup ? "Verified group" : "Verified user"} />{/if}
		</span>
		<small class="truncate text-2xs text-fg-muted"
			>{role} · {isGroup ? "Group" : `@${name}`} ·
			<span class="font-mono">ID {creatorId}</span></small>
	</span>
	<Icon
		svg={caretRightIcon}
		class="shrink-0 text-fg-faint"
		data-nudge="right"
		size={14}
		aria-hidden="true" />
	<span class="sr-only">{actionLabel}</span>
</button>

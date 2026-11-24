<script lang="ts">
	import sealCheckIcon from "@phosphor-icons/core/regular/seal-check.svg?raw"
	import { getCreatorNavigation } from "./creator-navigation"
	import type { CreatorKind } from "./creator-images"
	import Icon from "./Icon.svelte"

	// CreatorByline names the creator or the owner of a game, an item, or a
	// group in one line, with a link that opens them: a group in Groups, and a
	// user in Social. A creator can be a user or a group, so its line names
	// the kind. An owner is always a user.
	let {
		kind,
		creatorId,
		name,
		hasVerifiedBadge,
		role = "Creator",
	}: {
		kind: CreatorKind
		creatorId: number
		// name is the username of a user or the name of a group.
		name: string
		hasVerifiedBadge: boolean
		role?: "Creator" | "Owner"
	} = $props()

	const navigation = getCreatorNavigation(),
		openLabel = $derived(kind === "Group" ? "Show in Groups" : "Show in Social")

	function openCreator(): void {
		if (kind === "Group") navigation.showGroup(creatorId)
		else
			navigation.showUser({
				userId: creatorId,
				username: name,
				displayName: name,
			})
	}
</script>

<p class="flex min-w-0 flex-wrap items-center gap-1 text-xs text-fg-muted">
	{role === "Owner" ? "Owned by" : "By"}
	<button
		class="rounded-sm font-medium text-fg hover:underline hover:underline-offset-2 focus-visible:outline-offset-2"
		type="button"
		aria-label={`${name}. ${openLabel}`}
		data-tooltip={openLabel}
		onclick={openCreator}>{name}</button>
	{#if hasVerifiedBadge}<Icon
			svg={sealCheckIcon}
			class="text-accent-strong"
			size={14}
			aria-label="Verified creator" />{/if}
	{#if role === "Creator"}
		<span class="text-fg-faint" aria-hidden="true">·</span>
		<span>{kind}</span>
	{/if}
</p>

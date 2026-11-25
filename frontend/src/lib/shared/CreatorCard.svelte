<script lang="ts" module>
	// CreatorRole names what the creator is to the item, the game, or the
	// group that the page shows.
	export type CreatorRole = "Creator" | "Owner" | "Ally" | "Enemy"
</script>

<script lang="ts">
	import crownIcon from "@phosphor-icons/core/regular/crown.svg?raw"
	import handshakeIcon from "@phosphor-icons/core/regular/handshake.svg?raw"
	import paintBrushIcon from "@phosphor-icons/core/regular/paint-brush.svg?raw"
	import swordIcon from "@phosphor-icons/core/regular/sword.svg?raw"
	import { getProfilePreviews } from "./profile-preview"
	import { loadCreatorImage } from "./creator-images"
	import type { CreatorKind } from "./creator-images"
	import type { LabeledBadge } from "./labeled-badge"
	import ProfileCard from "./ProfileCard.svelte"

	let {
		kind,
		creatorId,
		name,
		displayName = "",
		hasVerifiedBadge,
		role,
		memberCount = null,
	}: {
		kind: CreatorKind
		creatorId: number
		// name is the username of a user or the name of a group.
		name: string
		displayName?: string
		hasVerifiedBadge: boolean
		role: CreatorRole
		memberCount?: number | null
	} = $props()

	const previews = getProfilePreviews(),
		compactNumbers = new Intl.NumberFormat("en-US", {
			notation: "compact",
			maximumFractionDigits: 1,
		}),
		roleBadges: Record<CreatorRole, LabeledBadge> = {
			Creator: { label: "Creator", icon: paintBrushIcon, isStrong: false },
			Owner: { label: "Owner", icon: crownIcon, isStrong: true },
			Ally: { label: "Ally", icon: handshakeIcon, isStrong: false },
			Enemy: { label: "Enemy", icon: swordIcon, isStrong: false },
		}
	let imageUrl = $state("")

	const isGroup = $derived(kind === "Group"),
		title = $derived(isGroup ? name : displayName || name),
		subtitle = $derived(
			!isGroup
				? `@${name}`
				: memberCount === null
					? "Group"
					: `${compactNumbers.format(memberCount)} ${memberCount === 1 ? "member" : "members"}`,
		)

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
		if (isGroup) previews.show({ kind: "Group", groupId: creatorId })
		else
			previews.show({
				kind: "User",
				userId: creatorId,
				username: name,
				displayName: displayName || name,
			})
	}
</script>

<ProfileCard
	{kind}
	{imageUrl}
	{title}
	{subtitle}
	{hasVerifiedBadge}
	badges={[roleBadges[role]]}
	action={{ label: isGroup ? "Show group" : "Show profile", open }} />

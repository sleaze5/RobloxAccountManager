<script lang="ts">
	import prohibitIcon from "@phosphor-icons/core/regular/prohibit.svg?raw"
	import vaultIcon from "@phosphor-icons/core/regular/vault.svg?raw"
	import Icon from "../shared/Icon.svelte"
	import type { SocialProfileSnapshot, SocialRelationship } from "../backend/bridge"
	import AccountMembershipBadges from "../accounts/AccountMembershipBadges.svelte"
	import RelationshipBadges from "./RelationshipBadges.svelte"
	import { relationshipBadges } from "./relationship"
	import type { RelationshipBadge } from "./relationship"

	let {
		profile,
		relationship,
	}: { profile: SocialProfileSnapshot; relationship: SocialRelationship | null } =
		$props()

	const badges = $derived.by((): RelationshipBadge[] => {
		if (!relationship) return []
		if (relationship.self) {
			return [
				{
					icon: vaultIcon,
					label: "This account",
					tooltip: "The account you are browsing with",
					isStrong: false,
				},
			]
		}
		const saved: RelationshipBadge[] =
			profile.savedAccountId > 0
				? [
						{
							icon: vaultIcon,
							label: "Saved account",
							tooltip: "Saved in this vault",
							isStrong: false,
						},
					]
				: []
		return [...saved, ...relationshipBadges(relationship)]
	})
</script>

{#if badges.length || profile.premium || profile.banned}
	<div class="flex flex-wrap items-center gap-1" aria-label="Relationship">
		<RelationshipBadges {badges} />
		<AccountMembershipBadges snapshot={{ premium: profile.premium, plus: null }} />
		{#if profile.banned}
			<span
				class="inline-flex h-5 cursor-help items-center gap-1 rounded-sm border border-danger/40 bg-danger/10 px-1 text-2xs font-medium text-danger-muted"
				data-tooltip="Roblox permanently banned this account">
				<Icon svg={prohibitIcon} size={12} aria-hidden="true" />Terminated
			</span>
		{/if}
	</div>
{/if}

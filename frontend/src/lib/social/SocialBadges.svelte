<script lang="ts">
	import handshakeIcon from "@phosphor-icons/core/regular/handshake.svg?raw"
	import paperPlaneTiltIcon from "@phosphor-icons/core/regular/paper-plane-tilt.svg?raw"
	import prohibitIcon from "@phosphor-icons/core/regular/prohibit.svg?raw"
	import shieldCheckIcon from "@phosphor-icons/core/regular/shield-check.svg?raw"
	import userCheckIcon from "@phosphor-icons/core/regular/user-check.svg?raw"
	import userFocusIcon from "@phosphor-icons/core/regular/user-focus.svg?raw"
	import userPlusIcon from "@phosphor-icons/core/regular/user-plus.svg?raw"
	import vaultIcon from "@phosphor-icons/core/regular/vault.svg?raw"
	import Icon from "../shared/Icon.svelte"
	import { FriendStatus, TrustedStatus } from "../backend/bridge"
	import type { SocialProfileSnapshot } from "../backend/bridge"
	import AccountMembershipBadges from "../accounts/AccountMembershipBadges.svelte"

	let { profile }: { profile: SocialProfileSnapshot } = $props()

	const relationship = $derived(profile.relationship),
		badges = $derived(
			[
				relationship.self && {
					icon: vaultIcon,
					label: "This account",
					tooltip: "The account you are browsing with",
				},
				!relationship.self &&
					profile.savedAccountId > 0 && {
						icon: vaultIcon,
						label: "Saved account",
						tooltip: "Saved in this vault",
					},
				relationship.friend === FriendStatus.FriendStatusFriends &&
					(relationship.trusted === TrustedStatus.TrustedStatusTrusted
						? {
								icon: shieldCheckIcon,
								label: "Trusted friend",
								tooltip: "A trusted connection of this account",
								strong: true,
							}
						: {
								icon: handshakeIcon,
								label: "Friend",
								tooltip: "Friends with this account",
								strong: true,
							}),
				relationship.friend === FriendStatus.FriendStatusRequestSent && {
					icon: paperPlaneTiltIcon,
					label: "Request sent",
					tooltip: "This account sent them a friend request",
				},
				relationship.friend === FriendStatus.FriendStatusRequestReceived && {
					icon: userPlusIcon,
					label: "Request received",
					tooltip: "They sent this account a friend request",
				},
				relationship.followedBy && {
					icon: userCheckIcon,
					label: "Follows you",
					tooltip: "They follow this account",
				},
				relationship.following && {
					icon: userFocusIcon,
					label: "Following",
					tooltip: "This account follows them",
				},
			].filter((badge) => !!badge),
		),
		pillClass =
			"inline-flex h-5 cursor-help items-center gap-1 rounded-sm border px-1 text-2xs font-medium"
</script>

{#if badges.length || profile.premium || profile.banned}
	<div class="flex flex-wrap items-center gap-1" aria-label="Relationship">
		{#each badges as badge (badge.label)}
			<span
				class={[
					pillClass,
					"animate-pop border-line-strong bg-raised",
					"strong" in badge ? "text-accent-strong" : "text-fg-muted",
				]}
				data-tooltip={badge.tooltip}>
				<Icon svg={badge.icon} size={12} aria-hidden="true" />{badge.label}
			</span>
		{/each}
		<AccountMembershipBadges snapshot={{ premium: profile.premium, plus: null }} />
		{#if profile.banned}
			<span
				class={[pillClass, "border-warning/40 bg-warning/10 text-warning"]}
				data-tooltip="Roblox banned this account">
				<Icon svg={prohibitIcon} size={12} aria-hidden="true" />Banned
			</span>
		{/if}
	</div>
{/if}

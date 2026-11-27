<script lang="ts">
	import crownIcon from "@phosphor-icons/core/regular/crown.svg?raw"
	import doorOpenIcon from "@phosphor-icons/core/regular/door-open.svg?raw"
	import identificationBadgeIcon from "@phosphor-icons/core/regular/identification-badge.svg?raw"
	import type { GroupMembership } from "../backend/bridge"
	import Icon from "../shared/Icon.svelte"

	// GroupMembershipFacts are the facts rows of a group that depend on the
	// account: its role, and whether joining needs a join request.
	let {
		membership,
		isPublicEntryAllowed,
	}: { membership: GroupMembership; isPublicEntryAllowed: boolean } = $props()

	const isMember = $derived(membership.role.rank > 0)
</script>

<div data-tone={isMember ? undefined : "muted"}>
	<dt>
		<Icon
			svg={membership.isOwner ? crownIcon : identificationBadgeIcon}
			size={16}
			aria-hidden="true" />Your role
	</dt>
	<dd>
		{isMember ? membership.role.name : "Not a member"}
		{#if isMember}<small
				>Rank {membership.role.rank}{membership.isOwner
					? " · Owner"
					: ""}</small
			>{/if}
	</dd>
</div>
<div>
	<dt><Icon svg={doorOpenIcon} size={16} aria-hidden="true" />Join requests</dt>
	<dd>{isPublicEntryAllowed ? "Not required" : "Required"}</dd>
</div>

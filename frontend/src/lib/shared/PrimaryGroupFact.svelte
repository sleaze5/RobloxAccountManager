<script lang="ts">
	import sealCheckIcon from "@phosphor-icons/core/regular/seal-check.svg?raw"
	import usersThreeIcon from "@phosphor-icons/core/regular/users-three.svg?raw"
	import type { UserPrimaryGroup } from "../backend/bridge"
	import { getProfilePreviews } from "./profile-preview"
	import Icon from "./Icon.svelte"

	let { group }: { group: UserPrimaryGroup } = $props()

	const previews = getProfilePreviews()
	const memberCounts = new Intl.NumberFormat("en-US", {
		notation: "compact",
		maximumFractionDigits: 1,
	})
</script>

<div>
	<dt>
		<Icon svg={usersThreeIcon} size={16} aria-hidden="true" />Primary group
	</dt>
	<dd>
		<button
			class="inline-flex min-w-0 items-center gap-1 rounded-sm text-left underline-offset-2 hover:underline"
			type="button"
			data-tooltip="Show group"
			onclick={() => previews.show({ kind: "Group", groupId: group.id })}
			>{group.name}{#if group.verified}<Icon
					svg={sealCheckIcon}
					class="shrink-0 text-fg-muted"
					size={14}
					aria-label="Verified group" />{/if}</button>
		<small
			>{group.role ? `${group.role} · ` : ""}{memberCounts.format(group.members)}
			{group.members === 1 ? "member" : "members"}</small>
	</dd>
</div>

<script lang="ts">
	import signInIcon from "@phosphor-icons/core/regular/sign-in.svg?raw"
	import { UserJoinability } from "../backend/bridge"
	import type { UserPresence } from "../backend/bridge"
	import Icon from "../shared/Icon.svelte"
	import { presenceJoinFor } from "./presence-join"
	import { notJoinableMessage } from "./user-joinability"

	// PresenceJoinButton is the Join action of a profile. It fills the launch
	// options with the user's place and server, or to follow a user whose
	// server is hidden, and shows nothing when the user cannot be followed.
	let {
		presence,
		joinability = UserJoinability.UserJoinabilityUnknown,
		onJoin,
		onJoinUser,
	}: {
		presence: UserPresence
		joinability?: UserJoinability
		onJoin: (placeId: number, serverId: string) => void
		onJoinUser?: () => void
	} = $props()

	const join = $derived(presenceJoinFor(presence, joinability, !!onJoinUser)),
		tooltip = $derived(
			join.hasVisibleServer
				? join.isNotJoinable
					? notJoinableMessage
					: "Fill launch options with this place and server"
				: "Fill launch options to follow this user. Their server is hidden.",
		)
</script>

{#if join.hasVisibleServer || join.canFollowUser}
	<button
		class="button-primary animate-enter"
		type="button"
		aria-label={`Join. ${tooltip}`}
		data-tooltip={tooltip}
		onclick={() => {
			if (join.hasVisibleServer) onJoin(join.placeId, join.serverId)
			else onJoinUser?.()
		}}>
		<Icon svg={signInIcon} size={14} aria-hidden="true" />
		Join
	</button>
{/if}

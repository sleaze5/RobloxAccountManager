<script lang="ts">
	import circleNotchIcon from "@phosphor-icons/core/regular/circle-notch.svg?raw"
	import Icon from "../shared/Icon.svelte"
	import RobloxActionStrip from "../shared/RobloxActionStrip.svelte"
	import type { RelationshipAction, RelationshipActionId } from "./relationship"

	// RelationshipActionStrip shows what the account can change about its
	// relationship with a user on Roblox, as icons that name the action in
	// their tooltip.
	let {
		username,
		actions,
		pendingActionId,
		isDisabled,
		onAction,
	}: {
		username: string
		actions: RelationshipAction[]
		pendingActionId: RelationshipActionId | null
		isDisabled: boolean
		onAction: (actionId: RelationshipActionId) => void
	} = $props()
</script>

<RobloxActionStrip {username}>
	{#each actions as action (action.id)}
		{@const isPending = pendingActionId === action.id}
		<button
			class="animate-enter"
			type="button"
			aria-label={isPending ? action.progress : action.label}
			aria-busy={isPending}
			data-tooltip={action.label}
			data-tone={action.tone === "neutral" ? undefined : action.tone}
			disabled={isDisabled}
			onclick={() => onAction(action.id)}>
			<Icon
				svg={isPending ? circleNotchIcon : action.icon}
				class={isPending ? "spinner" : undefined}
				size={14}
				aria-hidden="true" />
		</button>
	{/each}
</RobloxActionStrip>

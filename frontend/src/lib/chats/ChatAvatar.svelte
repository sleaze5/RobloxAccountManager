<script lang="ts">
	import usersIcon from "@phosphor-icons/core/regular/users.svg?raw"
	import Icon from "../shared/Icon.svelte"
	import type { ChatConversationView } from "../backend/bridge"
	import ProfileImage from "../shared/ProfileImage.svelte"

	let {
		conversation,
		selfUserId,
		className = "size-8",
	}: {
		conversation: ChatConversationView
		selfUserId: number
		className?: string
	} = $props()

	const faces = $derived(
		(conversation.participants ?? [])
			.filter(
				(participant) =>
					participant.userId !== selfUserId && participant.imageUrl,
			)
			.slice(0, 4),
	)
</script>

<span
	class={[
		"relative grid shrink-0 place-items-center overflow-hidden rounded-md border border-line-strong bg-raised",
		className,
	]}
	aria-hidden="true">
	{#if !conversation.group}
		<ProfileImage url={conversation.imageUrl} />
	{:else if faces.length > 1}
		<span
			class={[
				"grid size-full gap-px bg-line-strong",
				faces.length === 2 ? "grid-cols-2" : "grid-cols-2 grid-rows-2",
			]}>
			{#each faces as face, index (face.userId)}
				<span
					class={[
						"relative overflow-hidden bg-raised",
						faces.length === 3 && index === 0 && "row-span-2",
					]}><ProfileImage url={face.imageUrl} /></span>
			{/each}
		</span>
	{:else if faces.length === 1}
		<ProfileImage url={faces[0].imageUrl} />
	{:else}
		<Icon svg={usersIcon} class="size-1/2 text-fg-faint" aria-hidden="true" />
	{/if}
</span>

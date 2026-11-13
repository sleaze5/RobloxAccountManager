<script lang="ts">
	import medalIcon from "@phosphor-icons/core/regular/medal.svg?raw"
	import type { RobloxBadge } from "../backend/bridge"
	import ProfileImage from "../shared/ProfileImage.svelte"

	let { badges }: { badges: RobloxBadge[] } = $props()
</script>

<ul class="flex min-w-0 flex-wrap gap-1" aria-label="Roblox badges">
	{#each badges as badge (badge.name)}
		<li>
			<button
				class="inline-flex h-6 animate-pop cursor-help items-center gap-1 rounded-sm border border-line-strong bg-raised pr-1.5 pl-0.5 text-2xs font-medium text-fg-muted focus-visible:outline-offset-2"
				type="button"
				aria-label={badge.description
					? `${badge.name}. ${badge.description}`
					: badge.name}
				data-tooltip={badge.description || undefined}
				data-tooltip-side="top">
				<span
					class="grid size-5 shrink-0 place-items-center overflow-hidden rounded-sm"
					aria-hidden="true">
					<ProfileImage url={badge.imageUrl} fallbackIcon={medalIcon} />
				</span>
				{badge.name}
			</button>
		</li>
	{/each}
</ul>

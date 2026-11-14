<script lang="ts">
	import type { AvatarBodyColors, AvatarPaletteColor } from "../backend/bridge"
	import { bodyParts } from "./avatar-draft"

	let {
		bodyColors,
		paletteColorByHex,
		isPaletteLoaded,
	}: {
		bodyColors: AvatarBodyColors
		paletteColorByHex: Map<string, AvatarPaletteColor>
		// isPaletteLoaded tells whether a color without a palette entry is a
		// custom color, or only lacks its name while the palette loads.
		isPaletteLoaded: boolean
	} = $props()
</script>

<dl class="facts min-w-0 flex-1">
	{#each bodyParts as { part, label } (part)}
		{@const hexColor = bodyColors[part]}
		<div>
			<dt>
				<span
					class="size-3 shrink-0 rounded-sm border border-line-strong"
					style:background-color={hexColor ? `#${hexColor}` : undefined}
				></span
				>{label}
			</dt>
			<dd>
				{#if hexColor}
					{@const paletteColor = paletteColorByHex.get(hexColor)}
					<span class="font-mono select-text">#{hexColor}</span>
					{#if paletteColor}
						{#if paletteColor.name}<span>{paletteColor.name}</span>{/if}
						<small class="font-mono select-text"
							>BrickColor {paletteColor.brickColorId}</small>
					{:else if isPaletteLoaded}
						<small>Custom color</small>
					{/if}
				{:else}
					<span class="text-fg-muted">Unavailable</span>
				{/if}
			</dd>
		</div>
	{/each}
</dl>

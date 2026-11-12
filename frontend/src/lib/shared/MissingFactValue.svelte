<script lang="ts" generics="Snapshot extends object, Section">
	import type { SectionLoad } from "./section-load.svelte"
	import Skeleton from "./Skeleton.svelte"

	// MissingFactValue fills a fact that has no value: a placeholder while its
	// section loads, "Unavailable" when the section failed, and the empty label
	// when the section loaded without a value for the fact.
	let {
		sections,
		section,
		emptyLabel = "N/A",
	}: {
		sections: SectionLoad<Snapshot, Section>
		section: Section
		emptyLabel?: string
	} = $props()
</script>

{#if sections.hasSection(section)}<span class="text-fg-muted">{emptyLabel}</span
	>{:else if sections.isSectionUnavailable(section)}<span class="text-fg-muted"
		>Unavailable</span
	>{:else}<Skeleton className="w-28" />{/if}

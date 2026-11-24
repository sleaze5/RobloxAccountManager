<script lang="ts">
	import caretRightIcon from "@phosphor-icons/core/regular/caret-right.svg?raw"
	import circleNotchIcon from "@phosphor-icons/core/regular/circle-notch.svg?raw"
	import funnelSimpleIcon from "@phosphor-icons/core/regular/funnel-simple.svg?raw"
	import packageIcon from "@phosphor-icons/core/regular/package.svg?raw"
	import tagIcon from "@phosphor-icons/core/regular/tag.svg?raw"
	import warningCircleIcon from "@phosphor-icons/core/regular/warning-circle.svg?raw"
	import xIcon from "@phosphor-icons/core/regular/x.svg?raw"
	import { tick } from "svelte"
	import { ModelPropertyKind } from "../backend/bridge"
	import type { ModelProperty } from "../backend/bridge"
	import Icon from "../shared/Icon.svelte"
	import { matchesSearch, searchTerms } from "../shared/search"
	import { instancePath, nodeName } from "./explorer-tree"
	import type { ExplorerNode } from "./explorer-tree"
	import { classIcon } from "./instance-class-icons"
	import PropertyColumnResizer from "./PropertyColumnResizer.svelte"
	import PropertyValue from "./PropertyValue.svelte"

	// InstanceDetails shows the properties of the selected instance in the
	// groups that Roblox Studio shows, or the summary of an asset of a bundle.
	let {
		node,
		imageUrlByAssetId,
		propertyQuery = $bindable(),
		collapsedGroups,
		onToggleGroup,
		propertyNameWidth = $bindable(),
		referencedClassNameOf,
		onSelectInstance,
		onOpenDocument,
		openingAssetId,
		onOpenAsset,
	}: {
		node: ExplorerNode
		imageUrlByAssetId: ReadonlyMap<number, string>
		propertyQuery: string
		collapsedGroups: ReadonlySet<string>
		onToggleGroup: (group: string, isCollapsed: boolean) => void
		propertyNameWidth: number
		// referencedClassNameOf returns the class of the instance with the ID.
		referencedClassNameOf: (instanceId: number) => string
		onSelectInstance: (instanceId: number) => void
		onOpenDocument: (property: ModelProperty) => void
		// openingAssetId is the asset whose details load before they open, or 0.
		openingAssetId: number
		onOpenAsset: (assetId: number) => void
	} = $props()

	const elementIdPrefix = $props.id()
	const exactNumbers = new Intl.NumberFormat("en-US")
	// The X, Y, and Z components of a vector, colored as the axes of the 3D
	// viewers.
	const axes = [
		{ name: "X", colorClass: "text-axis-x" },
		{ name: "Y", colorClass: "text-axis-y" },
		{ name: "Z", colorClass: "text-axis-z" },
	] as const
	// The narrowest name column, and the narrowest value column that a wider
	// name column leaves.
	const minimumNameWidth = 96,
		minimumValueWidth = 160

	let searchInput = $state<HTMLInputElement>(),
		propertiesWidth = $state(0)

	const instance = $derived(node.instance),
		properties = $derived(instance?.properties ?? []),
		terms = $derived(searchTerms(propertyQuery)),
		shownProperties = $derived(
			terms.length > 0
				? properties.filter((property) =>
						matchesSearch(terms, `${property.name} ${property.text}`),
					)
				: properties,
		),
		// The groups keep the order of the properties, which the backend sorts
		// by group as Roblox Studio orders them.
		groups = $derived(
			Object.entries(
				Object.groupBy(shownProperties, (property) => property.group),
			),
		),
		// The images of the assets that the instance names, such as the mesh
		// and the texture of a SpecialMesh.
		assetPreviews = $derived(
			properties.filter(
				(property) =>
					property.kind === ModelPropertyKind.PropertyAsset &&
					imageUrlByAssetId.has(property.assetId),
			),
		)

	async function clearPropertyQuery(): Promise<void> {
		propertyQuery = ""
		await tick()
		searchInput?.focus()
	}
</script>

<!-- componentRows lists the X, Y, and Z of a vector as rows of their own.
Screen readers hear the name of the vector before each axis. -->
{#snippet componentRows(vectorName: string, components: string[], depth: number)}
	{#each components as component, componentIndex (componentIndex)}
		{@const axis = axes[componentIndex]}
		<div>
			<dt style:--property-depth={depth}>
				<span class="sr-only">{`${vectorName} `}</span>
				<span class={["font-semibold", axis?.colorClass]}>{axis?.name}</span>
			</dt>
			<dd>
				<span class="min-w-0 truncate font-mono tabular-nums select-text"
					>{component}</span>
			</dd>
		</div>
	{/each}
{/snippet}

{#if !instance}
	{@const model = node.model}
	<div class="flex min-h-0 flex-1 animate-enter flex-col gap-4 overflow-y-auto p-4">
		<header class="flex items-center gap-3">
			<span
				class="grid size-9 shrink-0 place-items-center rounded-md border border-line bg-raised text-fg-muted">
				<Icon svg={packageIcon} size={16} aria-hidden="true" />
			</span>
			<div class="min-w-0">
				<h3 class="truncate text-sm font-semibold text-fg">
					{nodeName(node) || "Unnamed asset"}
				</h3>
				<p class="font-mono text-2xs text-fg-muted select-text">
					ID: {model.assetId}
				</p>
			</div>
		</header>
		{#if model.unavailableMessage}
			<p class="flex items-center gap-2 notice-warning" role="alert">
				<Icon
					svg={warningCircleIcon}
					size={14}
					class="shrink-0"
					aria-hidden="true" />
				{model.unavailableMessage}
			</p>
		{:else}
			<dl class="facts">
				<div>
					<dt>Instances</dt>
					<dd>
						<span class="font-mono tabular-nums"
							>{exactNumbers.format(model.model.instanceCount)}</span>
					</dd>
				</div>
				<div>
					<dt>File format</dt>
					<dd><span class="font-mono">{model.model.format}</span></dd>
				</div>
			</dl>
		{/if}
	</div>
{:else}
	<header class="flex shrink-0 flex-col gap-2 border-b border-line-subtle p-3">
		<div class="flex items-center gap-3">
			<span
				class="grid size-9 shrink-0 place-items-center rounded-md border border-line bg-raised text-fg">
				{#key instance.className}
					<Icon
						svg={classIcon(instance.className)}
						size={16}
						class="animate-pop"
						aria-hidden="true" />
				{/key}
			</span>
			<div class="min-w-0 flex-1">
				<h3 class="truncate text-sm font-semibold text-fg select-text">
					{instance.name}
				</h3>
				<p
					class="flex min-w-0 items-center gap-1 font-mono text-2xs text-fg-muted">
					<span class="shrink-0">{instance.className}</span>
					<span class="text-fg-faint" aria-hidden="true">·</span>
					<span class="truncate text-fg-faint select-text"
						>{instancePath(node)}</span>
				</p>
			</div>
			<div
				class="search-field h-7 w-44 flex-none lg:w-56"
				data-closes-on-escape={propertyQuery ? "" : undefined}>
				<Icon svg={funnelSimpleIcon} size={14} aria-hidden="true" />
				<input
					bind:this={searchInput}
					type="search"
					placeholder="Filter properties"
					aria-label="Filter properties by name or value"
					autocomplete="off"
					spellcheck="false"
					bind:value={propertyQuery}
					onkeydown={(event) => {
						if (event.key === "Escape" && propertyQuery) {
							event.preventDefault()
							propertyQuery = ""
						}
					}} />
				{#if propertyQuery}
					<button
						type="button"
						aria-label="Clear search"
						onclick={() => void clearPropertyQuery()}>
						<Icon svg={xIcon} size={14} aria-hidden="true" />
					</button>
				{/if}
			</div>
		</div>
		{#if (instance.tags ?? []).length > 0}
			<ul class="flex flex-wrap gap-1" aria-label="Tags">
				{#each instance.tags ?? [] as tag (tag)}
					<li class="tag-pill">
						<Icon svg={tagIcon} size={12} aria-hidden="true" />
						<span>{tag}</span>
					</li>
				{/each}
			</ul>
		{/if}
	</header>

	<div
		class="relative flex min-h-0 flex-1 flex-col"
		style:--property-name-width={`${propertyNameWidth}px`}
		bind:clientWidth={propertiesWidth}>
		{#if groups.length > 0}
			<PropertyColumnResizer
				bind:width={propertyNameWidth}
				minimumWidth={minimumNameWidth}
				maximumWidth={Math.max(
					minimumNameWidth,
					propertiesWidth - minimumValueWidth,
				)} />
		{/if}
		<div
			class="min-h-0 flex-1 scrollbar-stable overflow-y-auto overscroll-contain pb-3">
			{#if assetPreviews.length > 0 && terms.length === 0}
				<ul
					class="flex gap-2 overflow-x-auto px-3 py-3"
					aria-label="Referenced assets">
					{#each assetPreviews as property (`${property.group}:${property.name}`)}
						{@const isOpening = openingAssetId === property.assetId}
						<li class="w-20 shrink-0">
							<button
								class="group flex w-full flex-col gap-1 rounded-lg text-left"
								type="button"
								aria-label={isOpening
									? `Loading ${property.name} details…`
									: `Show ${property.name} details`}
								aria-busy={isOpening}
								data-tooltip="Show asset details"
								disabled={openingAssetId !== 0}
								onclick={() => onOpenAsset(property.assetId)}>
								<span
									class="relative grid size-20 place-items-center overflow-hidden rounded-lg border border-line-subtle bg-raised transition-colors duration-100 group-enabled:group-hover:border-line-strong">
									<img
										class="size-full animate-pop object-contain"
										src={imageUrlByAssetId.get(property.assetId)}
										alt=""
										decoding="async"
										draggable="false"
										referrerpolicy="no-referrer" />
									{#if isOpening}
										<span
											class="absolute inset-0 grid place-items-center bg-scrim text-fg">
											<Icon
												svg={circleNotchIcon}
												class="spinner"
												size={20}
												aria-hidden="true" />
										</span>
									{/if}
								</span>
								<span
									class="truncate text-2xs text-fg-muted transition-colors duration-100 group-enabled:group-hover:text-fg">
									{property.name}
								</span>
							</button>
						</li>
					{/each}
				</ul>
			{/if}
			{#if properties.length === 0}
				<p class="p-3 text-xs text-fg-muted">
					This instance has no properties.
				</p>
			{:else if groups.length === 0}
				<div class="empty-state animate-enter py-6">
					<Icon svg={funnelSimpleIcon} size={28} aria-hidden="true" />
					<h2>No matching properties</h2>
					<p>Filter by the name or the value of a property.</p>
					<button
						class="button"
						type="button"
						onclick={() => void clearPropertyQuery()}>Clear search</button>
				</div>
			{:else}
				{#key node.key}
					<div class="animate-enter">
						{#each groups as [group, groupProperties] (group)}
							{@const isCollapsed =
								terms.length === 0 && collapsedGroups.has(group)}
							{@const groupId = `${elementIdPrefix}-group-${group.replaceAll(" ", "-")}`}
							<section aria-labelledby={`${groupId}-heading`}>
								<h4 id={`${groupId}-heading`} class="sticky top-0 z-10">
									<button
										class="flex h-7 w-full items-center gap-1 border-b border-line-subtle bg-surface pr-3 pl-1 text-left text-xs font-semibold text-fg-muted transition-colors duration-100 hover:text-fg"
										type="button"
										aria-expanded={!isCollapsed}
										aria-controls={groupId}
										onclick={() =>
											onToggleGroup(group, !isCollapsed)}>
										<span
											class="grid size-4 shrink-0 place-items-center">
											<Icon
												svg={caretRightIcon}
												size={12}
												class={[
													"text-fg-faint transition-transform duration-100 ease-out",
													!isCollapsed && "rotate-90",
												]}
												aria-hidden="true" />
										</span>
										<span class="truncate">{group}</span>
										<span
											class="ml-auto font-mono text-2xs font-normal text-fg-faint tabular-nums"
											>{groupProperties?.length ?? 0}</span>
									</button>
								</h4>
								{#if !isCollapsed}
									<dl
										id={groupId}
										class="property-grid animate-enter">
										{#each groupProperties ?? [] as property, index (`${property.name}:${index}`)}
											<!-- A vector names its parts, such as the Position
											of a CFrame, and their X, Y, and Z rows hold the
											numbers, so each number shows once. -->
											<div>
												<dt>
													<span class="truncate"
														>{property.name}</span>
												</dt>
												<dd>
													{#if property.kind !== ModelPropertyKind.PropertyVector}
														<PropertyValue
															{property}
															imageUrl={imageUrlByAssetId.get(
																property.assetId,
															) ?? ""}
															referencedClassName={referencedClassNameOf(
																property.referencedInstanceId,
															)}
															{onSelectInstance}
															{onOpenDocument} />
													{/if}
												</dd>
											</div>
											{#each property.vectorParts ?? [] as part, partIndex (partIndex)}
												{#if part.label}
													<div>
														<dt style:--property-depth={1}>
															<span class="sr-only"
																>{`${property.name} `}</span>
															<span class="truncate"
																>{part.label}</span>
														</dt>
														<dd></dd>
													</div>
													{@render componentRows(
														`${property.name} ${part.label}`,
														part.components ?? [],
														2,
													)}
												{:else}
													{@render componentRows(
														property.name,
														part.components ?? [],
														1,
													)}
												{/if}
											{/each}
										{/each}
									</dl>
								{/if}
							</section>
						{/each}
					</div>
				{/key}
			{/if}
		</div>
	</div>
{/if}

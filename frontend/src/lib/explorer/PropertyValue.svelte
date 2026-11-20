<script lang="ts">
	import arrowBendDownRightIcon from "@phosphor-icons/core/regular/arrow-bend-down-right.svg?raw"
	import fileTextIcon from "@phosphor-icons/core/regular/file-text.svg?raw"
	import { ModelPropertyKind } from "../backend/bridge"
	import type { ModelProperty } from "../backend/bridge"
	import { formatCompactBytes } from "../shared/bytes"
	import Icon from "../shared/Icon.svelte"

	// PropertyValue shows the value of one property in the way that suits its
	// kind, such as a swatch for a color or a link for a reference.
	let {
		property,
		imageUrl,
		onSelectInstance,
		onOpenDocument,
	}: {
		property: ModelProperty
		// imageUrl is the image of the asset that the property names.
		imageUrl: string
		onSelectInstance: (instanceId: number) => void
		onOpenDocument: (property: ModelProperty) => void
	} = $props()

	const lineCount = $derived(
		property.kind === ModelPropertyKind.PropertyDocument
			? property.text.split("\n").length
			: 0,
	)
</script>

{#if property.kind === ModelPropertyKind.PropertyDocument}
	<span class="flex min-w-0 items-center gap-2">
		<button
			class="button shrink-0"
			type="button"
			aria-label={`View ${property.name}`}
			data-view-document={property.name}
			onclick={() => onOpenDocument(property)}>
			<Icon svg={fileTextIcon} size={14} aria-hidden="true" />
			View
		</button>
		<small class="text-2xs text-fg-muted tabular-nums"
			>{lineCount === 1 ? "1 line" : `${lineCount} lines`}</small>
	</span>
{:else if property.kind === ModelPropertyKind.PropertyReference}
	{#if property.referencedInstanceId > 0}
		<button
			class="button-text py-0 font-mono text-fg"
			type="button"
			aria-label={`Select ${property.text || "the referenced instance"}`}
			data-tooltip="Select in the tree"
			onclick={() => onSelectInstance(property.referencedInstanceId)}>
			<Icon
				svg={arrowBendDownRightIcon}
				class="shrink-0 text-fg-muted"
				size={12}
				aria-hidden="true" />
			<span class="truncate">{property.text || "Unnamed"}</span>
		</button>
	{:else}
		<span class="text-fg-muted">None</span>
	{/if}
{:else if property.kind === ModelPropertyKind.PropertyColor}
	<span class="inline-flex items-center gap-2">
		<span
			class="size-3 shrink-0 rounded-sm border border-line-strong bg-(--swatch-color)"
			style:--swatch-color={property.color}
			aria-hidden="true"></span>
		<span class="font-mono tabular-nums select-text">{property.text}</span>
	</span>
{:else if property.kind === ModelPropertyKind.PropertyAsset}
	<span class="inline-flex min-w-0 items-center gap-2">
		{#if imageUrl}
			<img
				class="size-5 shrink-0 rounded-sm border border-line-subtle bg-raised object-contain"
				src={imageUrl}
				alt=""
				decoding="async"
				draggable="false"
				referrerpolicy="no-referrer" />
		{/if}
		<span class="min-w-0 font-mono wrap-anywhere select-text">{property.text}</span>
	</span>
{:else if property.kind === ModelPropertyKind.PropertyBinary}
	<span class="text-fg-muted"
		>Binary data · {formatCompactBytes(Number(property.text))}</span>
{:else if property.kind === ModelPropertyKind.PropertyBoolean}
	<span class={["font-mono", property.text === "true" ? "text-fg" : "text-fg-muted"]}
		>{property.text}</span>
{:else if property.text === ""}
	<span class="text-fg-muted">Empty</span>
{:else}
	<span class="font-mono whitespace-pre-line tabular-nums select-text"
		>{property.text}</span>
{/if}

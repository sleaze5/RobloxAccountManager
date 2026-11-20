<script lang="ts">
	import fileTextIcon from "@phosphor-icons/core/regular/file-text.svg?raw"
	import { ModelPropertyKind } from "../backend/bridge"
	import type { ModelProperty } from "../backend/bridge"
	import { formatCompactBytes } from "../shared/bytes"
	import CheckboxMark from "../shared/CheckboxMark.svelte"
	import Icon from "../shared/Icon.svelte"
	import { classIcon } from "./instance-class-icons"

	// PropertyValue shows the value of one property as an explorer does: a
	// check box for a boolean, a swatch for a color, and a link to the
	// instance that a reference points to. The rows of a vector show its
	// components themselves.
	let {
		property,
		imageUrl,
		referencedClassName,
		onSelectInstance,
		onOpenDocument,
	}: {
		property: ModelProperty
		// imageUrl is the image of the asset that the property names.
		imageUrl: string
		// referencedClassName is the class of the instance that a reference
		// points to.
		referencedClassName: string
		onSelectInstance: (instanceId: number) => void
		onOpenDocument: (property: ModelProperty) => void
	} = $props()

	const lineCount = $derived(
			property.kind === ModelPropertyKind.PropertyDocument
				? property.text.split("\n").length
				: 0,
		),
		// The asset ID inside a content value, such as rbxassetid://123, which
		// shows brighter than the rest of the address.
		assetIdText = $derived(String(property.assetId)),
		assetIdIndex = $derived(
			property.kind === ModelPropertyKind.PropertyAsset
				? property.text.lastIndexOf(assetIdText)
				: -1,
		)
</script>

{#if property.kind === ModelPropertyKind.PropertyDocument}
	<button
		class="button-text py-0 text-fg"
		type="button"
		aria-label={`View ${property.name}`}
		data-view-document={property.name}
		onclick={() => onOpenDocument(property)}>
		<Icon svg={fileTextIcon} size={14} class="shrink-0" aria-hidden="true" />
		View
	</button>
	<span class="font-mono text-2xs text-fg-muted tabular-nums"
		>{lineCount === 1 ? "1 line" : `${lineCount} lines`}</span>
{:else if property.kind === ModelPropertyKind.PropertyBoolean}
	{@const isChecked = property.text === "true"}
	<span
		class={[
			"mt-px grid size-3.5 shrink-0 place-items-center rounded-sm border",
			isChecked ? "border-solid bg-solid text-on-solid" : "border-line-control",
		]}>
		{#if isChecked}
			<CheckboxMark />
		{/if}
	</span>
	<span class="sr-only">{property.text}</span>
{:else if property.kind === ModelPropertyKind.PropertyColor}
	<span
		class="mt-px size-3.5 shrink-0 rounded-sm border border-line-strong bg-(--swatch-color)"
		style:--swatch-color={property.color}
		aria-hidden="true"></span>
	<span class="font-mono tabular-nums select-text">{property.text}</span>
	<span class="font-mono text-fg-muted uppercase select-text">{property.color}</span>
{:else if property.kind === ModelPropertyKind.PropertyReference}
	{#if property.referencedInstanceId > 0}
		<button
			class="group flex min-w-0 items-center gap-1.5 rounded-sm text-left text-fg"
			type="button"
			aria-label={`Select ${property.text || "the referenced instance"}`}
			data-tooltip="Select in the tree"
			onclick={() => onSelectInstance(property.referencedInstanceId)}>
			<Icon
				svg={classIcon(referencedClassName)}
				size={14}
				class="shrink-0 text-fg-muted transition-colors duration-100 group-hover:text-fg"
				aria-hidden="true" />
			<span class="truncate font-mono underline-offset-2 group-hover:underline"
				>{property.text || "Unnamed"}</span>
		</button>
	{:else}
		<span class="text-fg-muted">None</span>
	{/if}
{:else if property.kind === ModelPropertyKind.PropertyAsset}
	{#if imageUrl}
		<img
			class="size-4 shrink-0 rounded-sm border border-line-subtle bg-raised object-contain"
			src={imageUrl}
			alt=""
			decoding="async"
			draggable="false"
			referrerpolicy="no-referrer" />
	{/if}
	{#if assetIdIndex >= 0}
		<span class="min-w-0 truncate font-mono select-text"
			><span class="text-fg-muted">{property.text.slice(0, assetIdIndex)}</span
			>{assetIdText}<span class="text-fg-muted"
				>{property.text.slice(assetIdIndex + assetIdText.length)}</span
			></span>
	{:else}
		<span class="min-w-0 truncate font-mono select-text">{property.text}</span>
	{/if}
{:else if property.kind === ModelPropertyKind.PropertyBinary}
	<span class="text-fg-muted">Binary data</span>
	<span class="font-mono text-fg-muted tabular-nums"
		>{formatCompactBytes(Number(property.text))}</span>
{:else if property.text === ""}
	<span class="text-fg-muted">Empty</span>
{:else}
	<span
		class="min-w-0 font-mono wrap-anywhere whitespace-pre-line tabular-nums select-text"
		>{property.text}</span>
{/if}

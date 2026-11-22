<script lang="ts">
	import caretLeftIcon from "@phosphor-icons/core/regular/caret-left.svg?raw"
	import checkIcon from "@phosphor-icons/core/regular/check.svg?raw"
	import copyIcon from "@phosphor-icons/core/regular/copy.svg?raw"
	import type { ModelProperty } from "../backend/bridge"
	import Icon from "../shared/Icon.svelte"
	import { luauLines } from "./luau-tokens"
	import type { LuauTokenKind } from "./luau-tokens"

	// DocumentViewer shows a long string property, such as the source of a
	// script, with line numbers. Code reads in grayscale steps: keywords
	// brightest, then code, strings, and comments.
	let {
		instanceName,
		property,
		onBack,
		onCopy,
	}: {
		instanceName: string
		property: ModelProperty
		onBack: () => void
		onCopy: (text: string) => Promise<boolean>
	} = $props()

	const tokenClasses: Record<LuauTokenKind, string> = {
		code: "text-fg",
		comment: "text-fg-faint italic",
		string: "text-fg-muted",
		keyword: "font-medium text-fg-bright",
		number: "text-fg-bright",
	}

	let isCopied = $state(false)

	const lines = $derived(luauLines(property.text))

	async function copy(): Promise<void> {
		isCopied = await onCopy(property.text)
	}
</script>

<div class="flex min-h-0 flex-1 animate-enter flex-col">
	<header
		class="flex h-11 shrink-0 items-center gap-2 border-b border-line-subtle px-2">
		<button
			class="button-ghost"
			type="button"
			aria-label="Back to properties"
			data-tooltip="Back to properties"
			data-document-back
			onclick={onBack}>
			<Icon svg={caretLeftIcon} data-nudge="left" size={16} aria-hidden="true" />
		</button>
		<h3 class="min-w-0 flex-1 truncate font-mono text-xs font-medium text-fg">
			{instanceName}.{property.name}
		</h3>
		<span class="shrink-0 font-mono text-2xs text-fg-faint tabular-nums"
			>{lines.length === 1 ? "1 line" : `${lines.length} lines`}</span>
		<button class="button" type="button" onclick={() => void copy()}>
			{#if isCopied}
				<Icon
					svg={checkIcon}
					size={14}
					class="animate-bloom"
					aria-hidden="true" />
				Copied
			{:else}
				<Icon svg={copyIcon} size={14} aria-hidden="true" />
				Copy
			{/if}
		</button>
	</header>
	<div
		class="min-h-0 flex-1 scrollbar-stable overflow-auto overscroll-contain py-2 code-view"
		role="textbox"
		aria-readonly="true"
		aria-multiline="true"
		aria-label={`${property.name} of ${instanceName}`}
		tabindex="0">
		<ol class="min-w-max">
			{#each lines as line, index (index)}
				<li class="flex pr-4">
					<span
						class="sticky left-0 w-12 shrink-0 bg-canvas pr-3 text-right text-fg-faint tabular-nums select-none"
						aria-hidden="true">{index + 1}</span>
					<code class="whitespace-pre select-text"
						>{#each line as token, tokenIndex (tokenIndex)}<span
								class={tokenClasses[token.kind]}>{token.text}</span
							>{/each}</code>
				</li>
			{/each}
		</ol>
	</div>
</div>

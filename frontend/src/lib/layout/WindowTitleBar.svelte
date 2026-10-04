<script lang="ts">
	import GitBranchIcon from "phosphor-svelte/lib/GitBranchIcon"
	import MinusIcon from "phosphor-svelte/lib/MinusIcon"
	import SquareIcon from "phosphor-svelte/lib/SquareIcon"
	import XIcon from "phosphor-svelte/lib/XIcon"
	import { Browser, Window } from "@wailsio/runtime"
	import { repository } from "../../../package.json"

	let { version, pageTitle }: { version: string; pageTitle?: string } = $props()
	const repositoryURL = repository.url.replace(/^git\+/, "").replace(/\.git$/, "")
</script>

<header class="window-titlebar">
	<div class="window-brand" aria-hidden="true">
		<img src="/icon.svg" alt="" />
	</div>
	<div class="window-title">
		{#if pageTitle}
			<span>{pageTitle}</span>
		{:else}
			<span>Roblox Account Manager</span>
			<span class="window-title-separator" aria-hidden="true">-</span>
			<span class="window-title-version">{version}</span>
		{/if}
	</div>
	<div class="window-controls">
		<button
			type="button"
			aria-label="Open repository"
			data-tooltip="Open repository"
			onclick={() => void Browser.OpenURL(repositoryURL)}>
			<GitBranchIcon size={17} weight="light" aria-hidden="true" />
		</button>
		<button
			type="button"
			aria-label="Minimize window"
			data-tooltip="Minimize window"
			onclick={() => void Window.Minimise()}>
			<MinusIcon size={17} weight="light" aria-hidden="true" />
		</button>
		<button
			type="button"
			aria-label="Maximize or restore window"
			data-tooltip="Maximize or restore window"
			onclick={() => void Window.ToggleMaximise()}>
			<SquareIcon size={14} weight="light" aria-hidden="true" />
		</button>
		<button
			class="close-window"
			type="button"
			aria-label="Close window"
			data-tooltip="Close window"
			data-tooltip-side="bottom-end"
			onclick={() => void Window.Close()}>
			<XIcon size={18} weight="light" aria-hidden="true" />
		</button>
	</div>
</header>

<script lang="ts">
	import GitBranch from "@lucide/svelte/icons/git-branch"
	import Minus from "@lucide/svelte/icons/minus"
	import Square from "@lucide/svelte/icons/square"
	import X from "@lucide/svelte/icons/x"
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
			<GitBranch size={15} strokeWidth={1.6} />
		</button>
		<button
			type="button"
			aria-label="Minimize window"
			data-tooltip="Minimize window"
			onclick={() => void Window.Minimise()}>
			<Minus size={15} strokeWidth={1.6} />
		</button>
		<button
			type="button"
			aria-label="Maximize or restore window"
			data-tooltip="Maximize or restore window"
			onclick={() => void Window.ToggleMaximise()}>
			<Square size={12} strokeWidth={1.6} />
		</button>
		<button
			class="close-window"
			type="button"
			aria-label="Close window"
			data-tooltip="Close window"
			data-tooltip-side="bottom-end"
			onclick={() => void Window.Close()}>
			<X size={16} strokeWidth={1.6} />
		</button>
	</div>
</header>

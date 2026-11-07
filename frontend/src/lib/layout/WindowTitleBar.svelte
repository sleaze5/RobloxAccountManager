<script lang="ts">
	import gitBranchLightIcon from "@phosphor-icons/core/light/git-branch-light.svg?raw"
	import minusLightIcon from "@phosphor-icons/core/light/minus-light.svg?raw"
	import squareLightIcon from "@phosphor-icons/core/light/square-light.svg?raw"
	import xLightIcon from "@phosphor-icons/core/light/x-light.svg?raw"
	import Icon from "../shared/Icon.svelte"
	import { Browser, Window } from "@wailsio/runtime"
	import { repository } from "../../../package.json"

	let { version, release }: { version: string; release: boolean } = $props()
	const repositoryURL = repository.url.replace(/^git\+/, "").replace(/\.git$/, "")
	const controlClass =
		"grid h-full w-11 cursor-default place-items-center text-fg-muted transition-colors duration-100 hover:bg-hover hover:text-fg-bright"
</script>

<header
	class="relative flex h-9 shrink-0 items-center border-b border-line-subtle bg-surface select-none"
	style="--wails-draggable: drag"
	data-window-titlebar>
	<div class="grid h-full w-11 place-items-center" aria-hidden="true">
		<img class="size-5" src="/icon.svg" alt="" />
	</div>
	<div
		class="pointer-events-none absolute left-1/2 flex -translate-x-1/2 items-baseline gap-1 text-xs text-fg-muted">
		<span>Roblox Account Manager</span>
		{#if !release}<span class="text-2xs font-medium text-fg-faint">[DEV BUILD]</span
			>{/if}
		<span class="text-fg-faint" aria-hidden="true">-</span>
		<span class="font-mono text-2xs text-fg-faint">{version}</span>
	</div>
	<div class="ml-auto flex h-full" style="--wails-draggable: no-drag">
		<button
			class="button-ghost self-center"
			type="button"
			aria-label="Open repository"
			data-tooltip="Open repository"
			onclick={() => void Browser.OpenURL(repositoryURL)}>
			<Icon svg={gitBranchLightIcon} size={18} aria-hidden="true" />
		</button>
		<div class="mx-2 h-4 w-px self-center bg-line-strong" aria-hidden="true"></div>
		<button
			class={controlClass}
			type="button"
			aria-label="Minimize window"
			data-tooltip="Minimize window"
			onclick={() => void Window.Minimise()}>
			<Icon svg={minusLightIcon} size={18} aria-hidden="true" />
		</button>
		<button
			class={controlClass}
			type="button"
			aria-label="Maximize or restore window"
			data-tooltip="Maximize or restore window"
			onclick={() => void Window.ToggleMaximise()}>
			<Icon svg={squareLightIcon} size={14} aria-hidden="true" />
		</button>
		<button
			class={[controlClass, "hover:bg-danger"]}
			type="button"
			aria-label="Close window"
			data-tooltip="Close window"
			data-tooltip-side="bottom-end"
			onclick={() => void Window.Close()}>
			<Icon svg={xLightIcon} size={18} aria-hidden="true" />
		</button>
	</div>
</header>

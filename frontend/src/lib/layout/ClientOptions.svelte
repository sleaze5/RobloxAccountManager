<script lang="ts">
	import wrenchIcon from "@phosphor-icons/core/regular/wrench.svg?raw"
	import Icon from "../shared/Icon.svelte"
	import type { AccountStore } from "../accounts/account-store.svelte"
	import { robloxClientOptions as options } from "../settings/roblox-client-options.svelte"
	import SettingsStatus from "../settings/SettingsStatus.svelte"
	import { menuIn, menuOut } from "../shared/presence"

	let { store }: { store: AccountStore } = $props()
	let open = $state(false),
		root = $state<HTMLDivElement>(),
		trigger = $state<HTMLButtonElement>()
	const panelID = $props.id(),
		cleaner = $derived(options.cleaner),
		activeOptions = $derived(
			[
				options.multiInstance?.enabled && "Multi-instance",
				options.isCleanBeforeLaunchActive && "Clean before launch",
			].filter((option) => typeof option === "string"),
		),
		triggerLabel = $derived(
			activeOptions.length > 0
				? `Client options: ${activeOptions.join(", ")}`
				: "Client options",
		),
		cleanBeforeLaunchNote = $derived(
			cleaner?.isBlockedByMultiInstance
				? "Unavailable while multi-instance is on."
				: cleaner?.deleteLogsWhenCleaning
					? "Deletes caches, logs, and sign-in data. Keeps Roblox and its settings."
					: "Deletes caches and sign-in data. Keeps logs, Roblox, and its settings.",
		)

	$effect(() => {
		if (store.launching || !store.vault.unlocked) open = false
	})
	$effect(() => {
		if (!open) return
		// The multi-instance status changes while Roblox processes start and stop.
		const interval = setInterval(() => {
			if (options.multiInstance?.enabled) void options.refreshMultiInstance()
		}, 500)
		return () => clearInterval(interval)
	})
</script>

{#snippet optionSwitch(
	title: string,
	note: string,
	isChecked: boolean,
	isDisabled: boolean,
	onToggle: (enabled: boolean) => void,
	status?: { text: string; tone: "neutral" | "success" | "warning" },
)}
	{@const optionID = `${panelID}-${title.toLowerCase().replaceAll(" ", "-")}`}
	<div class="flex items-start justify-between gap-3">
		<span class="flex min-w-0 flex-col gap-0.5">
			<strong id={`${optionID}-title`} class="text-xs font-semibold text-fg"
				>{title}</strong>
			<small id={`${optionID}-note`} class="text-2xs leading-normal text-fg-muted"
				>{note}</small>
			{#if status}
				<SettingsStatus
					id={`${optionID}-status`}
					value={status.text}
					tone={status.tone} />
			{/if}
		</span>
		<label class="switch">
			<input
				type="checkbox"
				role="switch"
				aria-labelledby={`${optionID}-title`}
				aria-describedby={status
					? `${optionID}-note ${optionID}-status`
					: `${optionID}-note`}
				checked={isChecked}
				disabled={isDisabled}
				onchange={(event) => {
					const enabled = event.currentTarget.checked
					event.currentTarget.checked = isChecked
					onToggle(enabled)
				}} />
			<span aria-hidden="true"></span>
		</label>
	</div>
{/snippet}

<svelte:window
	onclick={(event) => {
		if (event.target instanceof Node && !root?.contains(event.target)) open = false
	}}
	onkeydown={(event) => {
		if (open && event.key === "Escape") {
			event.preventDefault()
			open = false
			trigger?.focus()
		}
	}} />

<div class="relative" bind:this={root}>
	<button
		class={["button-ghost-sm", activeOptions.length > 0 && "text-accent-strong"]}
		type="button"
		bind:this={trigger}
		aria-label={triggerLabel}
		aria-expanded={open}
		aria-controls={open ? panelID : undefined}
		data-open={open || undefined}
		data-tooltip={triggerLabel}
		data-tooltip-side="top"
		disabled={store.launching || !store.vault.unlocked}
		onclick={() => {
			open = !open
			if (open) void options.load()
		}}>
		<Icon svg={wrenchIcon} size={14} aria-hidden="true" />
	</button>
	{#if open}
		<section
			class="absolute right-0 bottom-full z-10 mb-2 flex w-80 flex-col gap-3 rounded-xl border border-line-strong bg-overlay p-3 shadow-menu"
			style="--menu-origin: bottom right"
			id={panelID}
			aria-label="Client options"
			in:menuIn
			out:menuOut>
			<strong class="text-xs font-semibold text-fg">Client options</strong>
			{@render optionSwitch(
				"Multi-instance",
				"Launch several games at once while Roblox Account Manager is open.",
				options.multiInstance?.enabled ?? false,
				options.isLoadingMultiInstance ||
					options.isSavingMultiInstance ||
					!options.multiInstance?.supported,
				(enabled) => void options.setMultiInstanceEnabled(enabled),
				options.multiInstanceStatus,
			)}
			{#if cleaner?.supported}
				{@render optionSwitch(
					"Clean before launch",
					cleanBeforeLaunchNote,
					cleaner.cleanBeforeLaunch,
					options.isSavingCleanBeforeLaunch ||
						cleaner.isBlockedByMultiInstance,
					(enabled) => void options.setCleanBeforeLaunch(enabled),
				)}
			{/if}
			{#if options.multiInstanceError || options.multiInstanceStatusError || options.cleanerError}
				<div class="notice-error" role="alert">
					{options.multiInstanceError ||
						options.multiInstanceStatusError ||
						options.cleanerError}
				</div>
			{/if}
		</section>
	{/if}
</div>

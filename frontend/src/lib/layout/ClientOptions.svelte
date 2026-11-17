<script lang="ts">
	import wrenchIcon from "@phosphor-icons/core/regular/wrench.svg?raw"
	import type { AccountStore } from "../accounts/account-store.svelte"
	import { robloxClientOptions as options } from "../settings/roblox-client-options.svelte"
	import SettingsStatus from "../settings/SettingsStatus.svelte"
	import SettingSwitch from "../settings/SettingSwitch.svelte"
	import LaunchPanelPopover from "./LaunchPanelPopover.svelte"

	let { store }: { store: AccountStore } = $props()
	const optionIDPrefix = $props.id(),
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
				? "Off while multi-instance is on."
				: "Clear caches and sign-in data before each launch.",
		)
</script>

{#snippet optionSwitch(
	optionID: string,
	title: string,
	note: string,
	isChecked: boolean,
	isDisabled: boolean,
	onToggle: (enabled: boolean) => void,
	status?: { text: string; tone: "neutral" | "success" | "warning" },
)}
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
		<SettingSwitch
			labelledBy={`${optionID}-title`}
			describedBy={status
				? `${optionID}-note ${optionID}-status`
				: `${optionID}-note`}
			{isChecked}
			{isDisabled}
			{onToggle} />
	</div>
{/snippet}

<LaunchPanelPopover
	title="Client options"
	{triggerLabel}
	triggerIcon={wrenchIcon}
	isTriggerCompact={true}
	isTriggerHighlighted={activeOptions.length > 0}
	isDisabled={store.launching || !store.vault.unlocked}
	onOpen={() => void options.load()}>
	{#snippet children()}
		<div
			class="flex flex-col gap-3"
			{@attach () => options.pollMultiInstanceWhileEnabled()}>
			{@render optionSwitch(
				`${optionIDPrefix}-multi-instance`,
				"Multi-instance",
				"Launch several games at once.",
				options.multiInstance?.enabled ?? false,
				options.isLoadingMultiInstance ||
					options.isSavingMultiInstance ||
					!options.multiInstance?.supported,
				(enabled) => void options.setMultiInstanceEnabled(enabled),
				options.multiInstanceStatus,
			)}
			{#if cleaner?.supported}
				{@render optionSwitch(
					`${optionIDPrefix}-clean-before-launch`,
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
		</div>
	{/snippet}
</LaunchPanelPopover>

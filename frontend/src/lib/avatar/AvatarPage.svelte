<script lang="ts">
	import arrowUUpLeftIcon from "@phosphor-icons/core/regular/arrow-u-up-left.svg?raw"
	import personArmsSpreadIcon from "@phosphor-icons/core/regular/person-arms-spread.svg?raw"
	import { onMount, tick, untrack } from "svelte"
	import AccountPageHeader from "../accounts/AccountPageHeader.svelte"
	import type { AccountStore } from "../accounts/account-store.svelte"
	import type { AvatarOutfit, InventoryItem } from "../backend/bridge"
	import InventoryItemDialog from "../inventory/InventoryItemDialog.svelte"
	import type { InventoryCategoryOption } from "../inventory/inventory-categories"
	import { InventoryState } from "../inventory/inventory-state.svelte"
	import type { NotificationCenter } from "../notifications/notification-center.svelte"
	import Icon from "../shared/Icon.svelte"
	import RefreshIcon from "../shared/RefreshIcon.svelte"
	import SkeletonTile from "../shared/SkeletonTile.svelte"
	import { unsavedChangesGuard } from "../layout/unsaved-changes-guard"
	import AvatarAdjustmentsPanel from "./AvatarAdjustmentsPanel.svelte"
	import AvatarBodyPanel from "./AvatarBodyPanel.svelte"
	import AvatarConfirmation from "./AvatarConfirmation.svelte"
	import AvatarEmotesPanel from "./AvatarEmotesPanel.svelte"
	import AvatarItemPicker from "./AvatarItemPicker.svelte"
	import AvatarOutfitDialog from "./AvatarOutfitDialog.svelte"
	import AvatarOutfitNameDialog from "./AvatarOutfitNameDialog.svelte"
	import AvatarOutfitsPanel from "./AvatarOutfitsPanel.svelte"
	import AvatarStage from "./AvatarStage.svelte"
	import AvatarWearingPanel from "./AvatarWearingPanel.svelte"
	import { avatarItemCategory, emoteCategory } from "./avatar-categories"
	import { emoteSlotCount, isDefaultAdjustment } from "./avatar-draft"
	import { AvatarState } from "./avatar-state.svelte"

	type AvatarTab = "wearing" | "adjustments" | "body" | "emotes" | "outfits"

	let {
		accounts,
		notifications,
		accountId,
		userId,
		username,
		onSearchPlace,
		onBack,
	}: {
		accounts: AccountStore
		notifications: NotificationCenter
		accountId: number
		userId: number
		username: string
		onSearchPlace: (placeId: number) => void
		onBack: () => void
	} = $props()

	const avatar = untrack(
			() => new AvatarState(accountId, userId, accounts, notifications),
		),
		// The item dialog and the item picker read the inventory of the user,
		// which is the account itself whenever the picker opens.
		inventory = untrack(
			() => new InventoryState(accountId, userId, accounts, notifications),
		),
		tabs: { id: AvatarTab; label: string }[] = [
			{ id: "wearing", label: "Worn" },
			{ id: "adjustments", label: "Adjustments" },
			{ id: "body", label: "Body" },
			{ id: "emotes", label: "Emotes" },
			{ id: "outfits", label: "Outfits" },
		]

	let tab = $state<AvatarTab>("wearing"),
		tabPanel = $state<HTMLDivElement | undefined>(undefined),
		detailedItem = $state<{
			item: InventoryItem
			category: InventoryCategoryOption
		} | null>(null),
		// The picker is closed while undefined. A null slot adds items, and a
		// number chooses an emote for that slot.
		pickerEmoteSlot = $state<number | null | undefined>(undefined),
		outfitNaming = $state<{ renamedOutfit: AvatarOutfit | null } | null>(null),
		deletedOutfit = $state<AvatarOutfit | null>(null),
		// The outfit dialog hides while an item of the outfit shows its details,
		// and shows again when the item dialog closes.
		viewedOutfit = $state<AvatarOutfit | null>(null),
		// The accessory whose controls the Adjustments tab opens, or 0.
		adjustedAssetId = $state(0),
		// The accessory that the Adjust shortcut asked the Adjustments tab to
		// focus, or 0.
		focusedAdjustmentAssetId = $state(0),
		isConfirmingDiscard = $state(false),
		// The navigation that closes the page once the user confirms discarding
		// changes, or null when the page stays open.
		pendingLeave = $state<(() => void) | null>(null)

	const draft = $derived(avatar.draft),
		canEdit = $derived(avatar.isEditable && !!avatar.rules && !!draft),
		isChanged = $derived(avatar.isChanged),
		savedAssetIds = $derived(
			new Set((avatar.avatar?.assets ?? []).map((asset) => asset.item.itemId)),
		),
		adjustedCount = $derived(
			(draft?.assets ?? []).filter(
				(asset) => !isDefaultAdjustment(asset.adjustment),
			).length,
		),
		countByTab = $derived<Partial<Record<AvatarTab, string>>>({
			wearing: draft ? String(draft.assets.length) : undefined,
			adjustments: adjustedCount > 0 ? String(adjustedCount) : undefined,
			emotes: draft ? `${draft.emotes.length}/${emoteSlotCount}` : undefined,
		})

	function openItem(item: InventoryItem, category: InventoryCategoryOption): void {
		inventory.clearItemActionError()
		detailedItem = { item, category }
		inventory.loadItemDetails(item, category.id, true)
		void inventory.checkFavorite(item)
	}

	// confirmLeaving holds a navigation while the draft has changes, and asks
	// whether to discard them first. It returns whether it held the navigation.
	function confirmLeaving(leave: () => void): boolean {
		if (!isChanged) return false
		pendingLeave = leave
		isConfirmingDiscard = true
		return true
	}

	// isTextEntry tells whether the target undoes its own typing, such as the
	// custom color field. A number field applies each value at once, so the
	// avatar undo covers it.
	function isTextEntry(target: EventTarget | null): boolean {
		return (
			target instanceof HTMLTextAreaElement ||
			(target instanceof HTMLInputElement &&
				!["range", "checkbox", "radio", "button", "number"].includes(
					target.type,
				))
		)
	}

	// applyFocusedField applies a value that the user typed but has not
	// confirmed. A field applies its value when it loses focus, so focus leaves
	// the field and comes back.
	function applyFocusedField(): void {
		const field = document.activeElement
		if (!(field instanceof HTMLInputElement)) return
		field.blur()
		field.focus()
	}

	// undoFromButton moves the focus to the selected tab after the last step,
	// because the Undo button leaves with the history and would drop the
	// keyboard focus.
	function undoFromButton(): void {
		avatar.undo()
		if (avatar.draftHistory.length === 0) void selectTab(tab, true)
	}

	function requestBack(): void {
		if (!confirmLeaving(onBack)) onBack()
	}

	function confirmDiscard(): void {
		const leave = pendingLeave
		avatar.discardChanges()
		isConfirmingDiscard = false
		pendingLeave = null
		leave?.()
	}

	function cancelDiscard(): void {
		isConfirmingDiscard = false
		pendingLeave = null
	}

	// A save that finishes while the user leaves leaves nothing to discard.
	$effect(() => {
		if (pendingLeave && !isChanged && !avatar.isSaving) untrack(confirmDiscard)
	})

	async function deleteOutfit(outfit: AvatarOutfit): Promise<void> {
		if (await avatar.deleteOutfit(outfit)) deletedOutfit = null
	}

	async function selectTab(nextTab: AvatarTab, shouldFocus: boolean): Promise<void> {
		if (nextTab !== tab) tabPanel?.scrollTo({ top: 0 })
		tab = nextTab
		if (!shouldFocus) return
		await tick()
		document.getElementById(`avatar-tab-${nextTab}`)?.focus()
	}

	function handleTabKeydown(event: KeyboardEvent): void {
		const index = tabs.findIndex((option) => option.id === tab),
			nextIndex =
				event.key === "ArrowRight"
					? (index + 1) % tabs.length
					: event.key === "ArrowLeft"
						? (index - 1 + tabs.length) % tabs.length
						: event.key === "Home"
							? 0
							: event.key === "End"
								? tabs.length - 1
								: -1
		if (nextIndex < 0) return
		event.preventDefault()
		void selectTab(tabs[nextIndex].id, true)
	}

	onMount(() => {
		const releaseUnsavedChangesGuard = unsavedChangesGuard.hold(confirmLeaving)
		void avatar.refresh()
		void avatar.loadRules()
		return () => {
			releaseUnsavedChangesGuard()
			avatar.dispose()
			inventory.dispose()
		}
	})
</script>

<svelte:window
	onkeydown={(event) => {
		if (!(event.ctrlKey || event.metaKey) || !canEdit || event.defaultPrevented) {
			return
		}
		const key = event.key.toLowerCase()
		if (key === "s") {
			event.preventDefault()
			applyFocusedField()
			void avatar.save()
		} else if (key === "z" && !event.shiftKey && !isTextEntry(event.target)) {
			event.preventDefault()
			avatar.undo()
		}
	}} />

<section class="settings-page" aria-labelledby="avatar-title">
	<AccountPageHeader
		titleId="avatar-title"
		title="Avatar"
		{username}
		onBack={requestBack}>
		{#if avatar.isEditable && isChanged}
			<span class="animate-enter text-2xs text-fg-muted" aria-live="polite"
				>Unsaved changes</span>
		{/if}
		{#if avatar.isEditable && avatar.draftHistory.length > 0}
			<button
				class="button-icon animate-enter"
				type="button"
				aria-label="Undo"
				aria-keyshortcuts="Control+Z"
				data-tooltip="Undo (Ctrl+Z)"
				disabled={!avatar.canUndo}
				onclick={undoFromButton}>
				<Icon svg={arrowUUpLeftIcon} size={16} aria-hidden="true" />
			</button>
		{/if}
		{#if avatar.isEditable && isChanged}
			<button
				class="button animate-enter"
				type="button"
				disabled={avatar.isDraftLocked}
				onclick={() => {
					pendingLeave = null
					isConfirmingDiscard = true
				}}>Discard</button>
		{/if}
		{#if avatar.isEditable}
			<button
				class="button-primary"
				type="button"
				aria-busy={avatar.isSaving}
				aria-keyshortcuts="Control+S"
				data-tooltip={isChanged ? "Save avatar (Ctrl+S)" : "No changes to save"}
				disabled={!canEdit || !isChanged || avatar.isDraftLocked}
				onclick={() => void avatar.save()}
				>{avatar.isSaving ? "Saving…" : "Save avatar"}</button>
		{/if}
		<button
			class="button-ghost"
			type="button"
			aria-label={avatar.loading ? "Refreshing avatar…" : "Refresh avatar"}
			data-tooltip={isChanged
				? "Save or discard your changes to refresh"
				: "Refresh avatar"}
			data-tooltip-side="bottom-end"
			aria-busy={avatar.loading}
			disabled={avatar.loading || avatar.isChangingAvatar || isChanged}
			onclick={() => {
				void avatar.refresh()
				if (avatar.outfits) void avatar.loadOutfits()
			}}>
			<RefreshIcon isRefreshing={avatar.loading} />
		</button>
	</AccountPageHeader>

	{#if !avatar.avatar && avatar.error}
		<div class="empty-state flex-1" role="alert">
			<Icon svg={personArmsSpreadIcon} size={28} aria-hidden="true" />
			<h2>Avatar unavailable</h2>
			<p>{avatar.error}</p>
			<button class="button" type="button" onclick={() => void avatar.refresh()}
				>Try again</button>
		</div>
	{:else}
		<div class="flex min-h-0 flex-1">
			<AvatarStage {avatar} {username} />

			<div class="@container flex min-w-0 flex-1 flex-col">
				<div
					class="flex h-9 shrink-0 items-stretch overflow-x-auto border-b border-line-subtle bg-surface"
					role="tablist"
					aria-label="Avatar sections"
					tabindex="-1"
					onkeydown={handleTabKeydown}>
					{#each tabs as option (option.id)}
						<button
							id={`avatar-tab-${option.id}`}
							class="tab"
							type="button"
							role="tab"
							aria-selected={tab === option.id}
							aria-controls="avatar-tab-panel"
							tabindex={tab === option.id ? 0 : -1}
							onclick={() => void selectTab(option.id, false)}>
							{option.label}
							{#if countByTab[option.id]}
								<span
									class="font-mono text-2xs font-normal text-fg-faint"
									>{countByTab[option.id]}</span>
							{/if}
						</button>
					{/each}
				</div>

				<div
					id="avatar-tab-panel"
					bind:this={tabPanel}
					class="min-h-0 flex-1 scrollbar-stable overflow-y-auto"
					role="tabpanel"
					aria-labelledby={`avatar-tab-${tab}`}
					aria-busy={avatar.loading}>
					{#if avatar.isEditable && avatar.rules === null}
						<div
							class="mx-3 mt-3 flex items-center gap-2 notice-error"
							role="alert">
							<span class="min-w-0 flex-1"
								>The avatar limits could not be loaded, so the avatar
								cannot be edited.</span>
							<button
								class="button"
								type="button"
								onclick={() => void avatar.loadRules()}
								>Try again</button>
						</div>
					{/if}
					{#if avatar.isAvatarOutdated}
						<div
							class="mx-3 mt-3 flex items-center gap-2 notice-warning"
							role="status">
							<span class="min-w-0 flex-1"
								>Roblox changed the avatar, but it could not be
								refreshed. Reload it to see what Roblox put on.</span>
							<button
								class="button"
								type="button"
								aria-busy={avatar.loading}
								disabled={avatar.loading || avatar.isChangingAvatar}
								onclick={() => void avatar.refresh()}
								>{avatar.loading ? "Refreshing…" : "Refresh"}</button>
						</div>
					{/if}
					{#if avatar.saveError}
						<p class="mx-3 mt-3 notice-error" role="alert">
							{avatar.saveError}
						</p>
					{/if}
					{#if avatar.error && avatar.avatar}
						<p class="mx-3 mt-3 notice-error" role="alert">
							{avatar.error}
						</p>
					{/if}
					{#if !draft}
						<div
							class="flex flex-col gap-4 p-3"
							role="status"
							aria-busy="true">
							<span class="sr-only">Loading avatar…</span>
							<div
								class="flex min-h-7 items-center text-xs"
								aria-hidden="true">
								<span class="skeleton-line w-56"></span>
							</div>
							<div class="flex flex-col gap-2" aria-hidden="true">
								<span class="text-xs"
									><span class="skeleton-line w-24"></span></span>
								<ul
									class="grid grid-cols-3 gap-2 @lg:grid-cols-4 @2xl:grid-cols-5 @4xl:grid-cols-6 @6xl:grid-cols-8">
									{#each Array.from({ length: 8 }, (_unused, index) => index) as index (index)}
										<SkeletonTile />
									{/each}
								</ul>
							</div>
						</div>
					{:else}
						{#key tab}
							<div class="animate-enter">
								{#if tab === "wearing"}
									<AvatarWearingPanel
										{avatar}
										{canEdit}
										{username}
										onOpenItem={(asset) =>
											openItem(
												asset.item,
												avatarItemCategory(asset.category),
											)}
										onAddItems={() => (pickerEmoteSlot = null)}
										onAdjustItem={(assetId) => {
											adjustedAssetId = focusedAdjustmentAssetId =
												assetId
											void selectTab("adjustments", false)
										}} />
								{:else if tab === "adjustments"}
									<AvatarAdjustmentsPanel
										{avatar}
										{canEdit}
										{username}
										bind:expandedAssetId={adjustedAssetId}
										bind:focusedAssetId={
											focusedAdjustmentAssetId
										} />
								{:else if tab === "body"}
									<AvatarBodyPanel {avatar} {canEdit} />
								{:else if tab === "emotes"}
									<AvatarEmotesPanel
										{avatar}
										{canEdit}
										onOpenEmote={(emote) =>
											openItem(emote.item, emoteCategory)}
										onChooseEmote={(slot) =>
											(pickerEmoteSlot = slot)} />
								{:else}
									<AvatarOutfitsPanel
										{avatar}
										canEdit={avatar.isEditable}
										{username}
										onOpenOutfit={(outfit) =>
											(viewedOutfit = outfit)}
										onCreateOutfit={() => {
											avatar.clearOutfitActionError()
											outfitNaming = { renamedOutfit: null }
										}}
										onRenameOutfit={(outfit) => {
											avatar.clearOutfitActionError()
											outfitNaming = { renamedOutfit: outfit }
										}}
										onDeleteOutfit={(outfit) => {
											avatar.clearOutfitActionError()
											deletedOutfit = outfit
										}} />
								{/if}
							</div>
						{/key}
					{/if}
				</div>
			</div>
		</div>
	{/if}
</section>

{#if detailedItem}
	<!-- The worn items are not an inventory listing, so the dialog checks
	     whether the account owns each item and offers no deletion. -->
	<InventoryItemDialog
		item={detailedItem.item}
		category={detailedItem.category}
		{inventory}
		{username}
		account={accounts.selectedAccount}
		isOwnInventory={false}
		isWorn={savedAssetIds.has(detailedItem.item.itemId)}
		onSearchPlace={(placeId) => {
			detailedItem = null
			viewedOutfit = null
			onSearchPlace(placeId)
		}}
		onClose={() => (detailedItem = null)} />
{/if}

{#if viewedOutfit && !detailedItem}
	<AvatarOutfitDialog
		{avatar}
		outfit={viewedOutfit}
		canWear={avatar.isEditable}
		onOpenItem={(asset) => openItem(asset.item, avatarItemCategory(asset.category))}
		onClose={() => (viewedOutfit = null)} />
{/if}

{#if pickerEmoteSlot !== undefined}
	<AvatarItemPicker
		{avatar}
		{inventory}
		emoteSlot={pickerEmoteSlot}
		onClose={() => (pickerEmoteSlot = undefined)} />
{/if}

{#if outfitNaming}
	<AvatarOutfitNameDialog
		{avatar}
		renamedOutfit={outfitNaming.renamedOutfit}
		onClose={() => (outfitNaming = null)} />
{/if}

{#if deletedOutfit}
	{@const outfit = deletedOutfit}
	<AvatarConfirmation
		title="Delete outfit?"
		description={`${outfit.name || "This outfit"} will be deleted from @${username}'s outfits. This cannot be undone.`}
		actionLabel="Delete outfit"
		progressLabel="Deleting…"
		tone="danger"
		isWorking={avatar.isChangingOutfit}
		error={avatar.outfitActionError}
		onConfirm={() => void deleteOutfit(outfit)}
		onCancel={() => (deletedOutfit = null)} />
{/if}

{#if isConfirmingDiscard}
	<AvatarConfirmation
		title="Discard changes?"
		description="The avatar goes back to how it was last saved. Your unsaved changes are lost."
		actionLabel="Discard changes"
		progressLabel="Saving…"
		tone="warning"
		isWorking={avatar.isSaving}
		error={avatar.saveError}
		onConfirm={confirmDiscard}
		onCancel={cancelDiscard} />
{/if}

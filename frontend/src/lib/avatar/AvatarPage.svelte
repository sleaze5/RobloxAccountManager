<script lang="ts">
	import { notifyErrors } from "../notifications/error-notifications.svelte"
	import warningIcon from "@phosphor-icons/core/regular/warning.svg?raw"
	import arrowUUpLeftIcon from "@phosphor-icons/core/regular/arrow-u-up-left.svg?raw"
	import { onMount, tick, untrack } from "svelte"
	import ProfilePageHeader from "../shared/ProfilePageHeader.svelte"
	import type { AccountStore } from "../accounts/account-store.svelte"
	import type {
		AvatarAsset,
		AvatarOutfit,
		AvatarThumbnailChange,
		InventoryCategory,
		InventoryItem,
	} from "../backend/bridge"
	import InventoryItemDialog from "../inventory/InventoryItemDialog.svelte"
	import type { InventoryCategoryOption } from "../inventory/inventory-categories"
	import {
		categoryItemKey,
		InventoryState,
	} from "../inventory/inventory-state.svelte"
	import type { NotificationCenter } from "../notifications/notification-center.svelte"
	import Icon from "../shared/Icon.svelte"
	import RefreshIcon from "../shared/RefreshIcon.svelte"
	import SkeletonTile from "../shared/SkeletonTile.svelte"
	import { getPageVisibility } from "../layout/page-visibility"
	import { unsavedChangesGuard } from "../layout/unsaved-changes-guard"
	import AvatarAdjustmentsPanel from "./AvatarAdjustmentsPanel.svelte"
	import AvatarBodyPanel from "./AvatarBodyPanel.svelte"
	import ConfirmationDialog from "../shared/ConfirmationDialog.svelte"
	import AvatarEmotesPanel from "./AvatarEmotesPanel.svelte"
	import AvatarItemPicker from "./AvatarItemPicker.svelte"
	import AvatarOutfitDialog from "./AvatarOutfitDialog.svelte"
	import AvatarOutfitNameDialog from "./AvatarOutfitNameDialog.svelte"
	import AvatarOutfitsPanel from "./AvatarOutfitsPanel.svelte"
	import AvatarStage from "./AvatarStage.svelte"
	import AvatarThumbnailsPanel from "./AvatarThumbnailsPanel.svelte"
	import AvatarWearingPanel from "./AvatarWearingPanel.svelte"
	import { avatarItemCategory, emoteCategory } from "./avatar-categories"
	import { emoteSlotCount, isDefaultAdjustment } from "./avatar-draft"
	import { AvatarAnimationLibrary } from "./avatar-animation-library.svelte"
	import { AvatarState } from "./avatar-state.svelte"
	import { AvatarThumbnailsState } from "./avatar-thumbnails-state.svelte"
	import type { ThumbnailFraming } from "../model-viewer/model-subject"

	type AvatarTab =
		| "wearing"
		| "adjustments"
		| "body"
		| "emotes"
		| "thumbnails"
		| "outfits"

	let {
		accounts,
		notifications,
		accountId,
		userId,
		username,
	}: {
		accounts: AccountStore
		notifications: NotificationCenter
		accountId: number
		userId: number
		username: string
	} = $props()
	const elementIdPrefix = $props.id(),
		pageVisibility = getPageVisibility()

	const avatar = untrack(
			() => new AvatarState(accountId, userId, accounts, notifications),
		),
		// The item dialog and the item picker read the inventory of the user,
		// which is the account itself whenever the picker opens.
		inventory = untrack(
			() => new InventoryState(accountId, userId, accounts, notifications),
		),
		thumbnails = untrack(
			() => new AvatarThumbnailsState(accountId, userId, accounts, notifications),
		),
		// The 3D view and the thumbnail poses offer what the user owns.
		animationLibrary = untrack(
			() => new AvatarAnimationLibrary(accountId, userId, accounts),
		),
		tabs: { id: AvatarTab; label: string }[] = [
			{ id: "wearing", label: "Worn" },
			{ id: "adjustments", label: "Adjustments" },
			{ id: "body", label: "Body" },
			{ id: "emotes", label: "Emotes" },
			{ id: "thumbnails", label: "Thumbnails" },
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
		// The profile image that the stage previews on the Thumbnails tab, and
		// how each edited customization frames the avatar.
		previewedThumbnailImage = $state<ThumbnailFraming["image"]>("headshot"),
		framingByImage = $state.raw<
			Partial<Record<ThumbnailFraming["image"], ThumbnailFraming>>
		>({}),
		// The navigation that closes the page once the user confirms discarding
		// changes, or null when the page stays open.
		pendingLeave = $state<(() => void) | null>(null)

	const draft = $derived(avatar.draft),
		stageFraming = $derived(
			tab === "thumbnails" ? framingByImage[previewedThumbnailImage] : undefined,
		),
		canEdit = $derived(avatar.isEditable && !!avatar.rules && !!draft),
		isChanged = $derived(avatar.isChanged),
		isOwnAvatar = $derived(Number(accounts.selectedAccount.userId) === userId),
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

	// isOwnedByAccount tells whether the selected account owns an item that
	// the page shows. The account owns everything that its own avatar wears.
	function isOwnedByAccount(category: InventoryCategory, itemId: number): boolean {
		return (
			isOwnAvatar ||
			inventory.isOwnedByItemKey.get(categoryItemKey(category, itemId)) === true
		)
	}

	function isAssetOwned(asset: AvatarAsset): boolean {
		return isOwnedByAccount(
			avatarItemCategory(asset.category).id,
			asset.item.itemId,
		)
	}

	// The tiles of another user's avatar show which items the account owns.
	$effect(() => {
		if (isOwnAvatar || !draft) return
		const viewedOutfitAssets = viewedOutfit
				? (avatar.outfitDetailsById.get(viewedOutfit.outfitId)?.assets ?? [])
				: [],
			categorizedItems = [
				...[...draft.assets, ...viewedOutfitAssets].map((asset) => ({
					category: avatarItemCategory(asset.category).id,
					item: asset.item,
				})),
				...draft.emotes.map((emote) => ({
					category: emoteCategory.id,
					item: emote.item,
				})),
			]
		untrack(() => void inventory.loadOwnership(categorizedItems))
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

	function leaveIfConfirmed(navigate: () => void): void {
		if (!confirmLeaving(navigate)) navigate()
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

	// saveThumbnail saves a thumbnail customization, then reads the images again
	// once Roblox renders the full-body image, which it renders with the
	// close-up images.
	async function saveThumbnail(change: AvatarThumbnailChange): Promise<void> {
		if (!(await thumbnails.save(change))) return
		await avatar.showRenderedImage()
		void thumbnails.refresh()
	}

	async function deleteOutfit(outfit: AvatarOutfit): Promise<void> {
		if (await avatar.deleteOutfit(outfit)) deletedOutfit = null
	}

	async function selectTab(nextTab: AvatarTab, shouldFocus: boolean): Promise<void> {
		if (nextTab !== tab) tabPanel?.scrollTo({ top: 0 })
		tab = nextTab
		if (!shouldFocus) return
		await tick()
		document.getElementById(`${elementIdPrefix}-avatar-tab-${nextTab}`)?.focus()
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
		void avatar.refresh()
		void avatar.loadRules()
		void thumbnails.refresh()
		return () => {
			avatar.dispose()
			inventory.dispose()
			thumbnails.dispose()
			animationLibrary.dispose()
		}
	})

	// Only the shown page can ask about its changes. Leaving a page asks
	// first, so a hidden page has none.
	$effect(() => {
		if (pageVisibility.isShown) return unsavedChangesGuard.hold(confirmLeaving)
	})

	notifyErrors("Avatar not loaded", () => avatar.error)
	notifyErrors("Avatar not saved", () => avatar.saveError)
	notifyErrors("Avatar limits not loaded", () =>
		avatar.isEditable && avatar.rules === null
			? "Could not load the avatar limits. You cannot edit the avatar. Refresh to try again."
			: "",
	)

	notifyErrors("Outfit not deleted", () => avatar.outfitActionError)
</script>

<svelte:window
	onkeydown={(event) => {
		if (
			!pageVisibility.isShown ||
			!(event.ctrlKey || event.metaKey) ||
			!canEdit ||
			event.defaultPrevented
		) {
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

<section class="settings-page" aria-labelledby={`${elementIdPrefix}-avatar-title`}>
	<ProfilePageHeader
		titleId={`${elementIdPrefix}-avatar-title`}
		title="Avatar"
		subtitle={`@${username}`}
		onLeave={leaveIfConfirmed}>
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
				void thumbnails.refresh()
				if (avatar.outfits) void avatar.loadOutfits()
			}}>
			<RefreshIcon isRefreshing={avatar.loading} />
		</button>
	</ProfilePageHeader>

	{#if !avatar.avatar && avatar.error}
		<div class="empty-state flex-1">
			<Icon svg={warningIcon} size={28} aria-hidden="true" />
			<h2>Avatar unavailable</h2>
			<button class="button" type="button" onclick={() => void avatar.refresh()}
				>Try again</button>
		</div>
	{:else}
		<div class="flex min-h-0 flex-1">
			<AvatarStage
				{avatar}
				{username}
				library={animationLibrary}
				thumbnailAnimation={thumbnails.thumbnails?.fullBody.emote ?? undefined}
				thumbnailFraming={stageFraming}
				bind:previewedThumbnailImage />

			<div class="@container flex min-w-0 flex-1 flex-col">
				<div
					class="tab-bar items-center gap-0.5 overflow-x-auto px-1"
					role="tablist"
					aria-label="Avatar sections"
					tabindex="-1"
					onkeydown={handleTabKeydown}>
					{#each tabs as option (option.id)}
						<button
							id={`${elementIdPrefix}-avatar-tab-${option.id}`}
							class="sub-tab"
							type="button"
							role="tab"
							aria-selected={tab === option.id}
							aria-controls={`${elementIdPrefix}-avatar-tab-panel`}
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
					id={`${elementIdPrefix}-avatar-tab-panel`}
					bind:this={tabPanel}
					class="min-h-0 flex-1 scrollbar-stable overflow-y-auto"
					data-load-more-root
					role="tabpanel"
					aria-labelledby={`${elementIdPrefix}-avatar-tab-${tab}`}
					aria-busy={avatar.loading}>
					<h2 class="sr-only">
						{tabs.find((editorTab) => editorTab.id === tab)?.label}
					</h2>
					{#if avatar.isAvatarOutdated}
						<div
							class="mx-3 mt-3 flex items-center gap-2 notice-warning"
							role="status">
							<span class="min-w-0 flex-1"
								>Roblox changed the avatar, but it did not refresh.
								Refresh to see the change.</span>
							<button
								class="button"
								type="button"
								aria-busy={avatar.loading}
								disabled={avatar.loading || avatar.isChangingAvatar}
								onclick={() => void avatar.refresh()}
								>{avatar.loading ? "Refreshing…" : "Refresh"}</button>
						</div>
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
									class="grid grid-cols-3 gap-2 @2xl:grid-cols-4 @4xl:grid-cols-5 @6xl:grid-cols-7">
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
										isOwned={isAssetOwned}
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
								{:else if tab === "thumbnails"}
									<AvatarThumbnailsPanel
										{avatar}
										{thumbnails}
										library={animationLibrary}
										canEdit={avatar.isEditable}
										onSave={(change) => void saveThumbnail(change)}
										onFramingChange={(framing) =>
											(framingByImage = {
												...framingByImage,
												[framing.image]: framing,
											})}
										onPreview={(image) =>
											(previewedThumbnailImage = image)}
										onOpenPose={(emote, category) =>
											openItem(
												emote,
												avatarItemCategory(category),
											)} />
								{:else if tab === "emotes"}
									<AvatarEmotesPanel
										{avatar}
										{canEdit}
										isOwned={(emote) =>
											isOwnedByAccount(
												emoteCategory.id,
												emote.item.itemId,
											)}
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
		{accounts}
		isOwnInventory={false}
		isWorn={savedAssetIds.has(detailedItem.item.itemId)}
		onClose={() => (detailedItem = null)} />
{/if}

{#if viewedOutfit && !detailedItem}
	<AvatarOutfitDialog
		{avatar}
		outfit={viewedOutfit}
		canWear={avatar.isEditable}
		isOwned={isAssetOwned}
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
	<ConfirmationDialog
		title="Delete outfit?"
		description={`This deletes ${outfit.name || "this outfit"} from @${username}'s outfits. This cannot be undone.`}
		confirmLabel="Delete outfit"
		progressLabel="Deleting…"
		account={accounts.selectedAccount}
		isWorking={avatar.isChangingOutfit}
		onConfirm={() => void deleteOutfit(outfit)}
		onCancel={() => (deletedOutfit = null)} />
{/if}

{#if isConfirmingDiscard}
	<ConfirmationDialog
		title="Discard changes?"
		description="The avatar goes back to how it was last saved. Your unsaved changes are lost."
		confirmLabel="Discard changes"
		progressLabel="Saving…"
		tone="warning"
		isWorking={avatar.isSaving}
		onConfirm={confirmDiscard}
		onCancel={cancelDiscard} />
{/if}

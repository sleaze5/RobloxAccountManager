<script lang="ts">
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
	import { emoteSlotCount } from "./avatar-draft"
	import { AvatarState } from "./avatar-state.svelte"

	type AvatarTab = "wearing" | "body" | "emotes" | "outfits"

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
		inventory = untrack(() => new InventoryState(accountId, userId, accounts)),
		tabs: { id: AvatarTab; label: string }[] = [
			{ id: "wearing", label: "Wearing" },
			{ id: "body", label: "Body" },
			{ id: "emotes", label: "Emotes" },
			{ id: "outfits", label: "Outfits" },
		]

	let tab = $state<AvatarTab>("wearing"),
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
		isConfirmingDiscard = $state(false),
		// Whether the page closes once the user confirms discarding changes.
		isLeavingAfterDiscard = false

	const draft = $derived(avatar.draft),
		canEdit = $derived(avatar.isEditable && !!avatar.rules && !!draft),
		isChanged = $derived(avatar.isChanged),
		savedAssetIds = $derived(
			new Set((avatar.avatar?.assets ?? []).map((asset) => asset.item.itemId)),
		),
		countByTab = $derived<Partial<Record<AvatarTab, string>>>({
			wearing: draft ? String(draft.assets.length) : undefined,
			emotes: draft ? `${draft.emotes.length}/${emoteSlotCount}` : undefined,
		})

	function openItem(item: InventoryItem, category: InventoryCategoryOption): void {
		inventory.clearItemActionError()
		detailedItem = { item, category }
		inventory.loadItemDetails(item, category.id, true)
		void inventory.checkFavorite(item)
	}

	function requestBack(): void {
		if (isChanged && !avatar.isSaving) {
			isLeavingAfterDiscard = true
			isConfirmingDiscard = true
		} else if (!avatar.isSaving) {
			onBack()
		}
	}

	function confirmDiscard(): void {
		avatar.discardChanges()
		isConfirmingDiscard = false
		if (isLeavingAfterDiscard) onBack()
	}

	async function deleteOutfit(outfit: AvatarOutfit): Promise<void> {
		if (await avatar.deleteOutfit(outfit)) deletedOutfit = null
	}

	async function selectTab(nextTab: AvatarTab, shouldFocus: boolean): Promise<void> {
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
		void avatar.refresh()
		void avatar.loadRules()
		return () => {
			avatar.dispose()
			inventory.dispose()
		}
	})
</script>

<svelte:window
	onkeydown={(event) => {
		if (
			(event.ctrlKey || event.metaKey) &&
			event.key.toLowerCase() === "s" &&
			canEdit
		) {
			event.preventDefault()
			void avatar.save()
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
			<button
				class="button animate-enter"
				type="button"
				disabled={avatar.isSaving}
				onclick={() => {
					isLeavingAfterDiscard = false
					isConfirmingDiscard = true
				}}>Discard…</button>
		{/if}
		{#if avatar.isEditable}
			<button
				class="button-primary"
				type="button"
				aria-busy={avatar.isSaving}
				aria-keyshortcuts="Control+S"
				data-tooltip={isChanged ? "Save avatar (Ctrl+S)" : "No changes to save"}
				disabled={!canEdit || !isChanged || avatar.isSaving}
				onclick={() => void avatar.save()}
				>{avatar.isSaving ? "Saving…" : "Save avatar"}</button>
		{/if}
		<button
			class="button-ghost"
			type="button"
			aria-label={avatar.loading ? "Refreshing avatar" : "Refresh avatar"}
			data-tooltip={isChanged
				? "Save or discard your changes to refresh"
				: "Refresh avatar"}
			data-tooltip-side="bottom-end"
			aria-busy={avatar.loading}
			disabled={avatar.loading || avatar.isSaving || isChanged}
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
			<h2>Avatar is unavailable</h2>
			<p>{avatar.error}</p>
			<button class="button" type="button" onclick={() => void avatar.refresh()}
				>Try again</button>
		</div>
	{:else}
		<div class="flex min-h-0 flex-1">
			<AvatarStage {avatar} {username} />

			<main
				class="@container flex min-w-0 flex-1 flex-col"
				aria-labelledby="avatar-title">
				<div
					class="flex h-9 shrink-0 items-stretch border-b border-line-subtle bg-surface"
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
								can't be edited.</span>
							<button
								class="button"
								type="button"
								onclick={() => void avatar.loadRules()}
								>Try again</button>
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
							class="flex flex-col gap-2 p-3"
							role="status"
							aria-busy="true">
							<span class="sr-only">Loading the avatar</span>
							<ul
								class="grid grid-cols-3 gap-2 @lg:grid-cols-4 @2xl:grid-cols-5"
								aria-hidden="true">
								{#each Array.from({ length: 8 }, (_unused, index) => index) as index (index)}
									<li
										class="flex flex-col gap-1 rounded-lg border border-line-subtle bg-surface p-1">
										<span
											class="block aspect-square w-full skeleton rounded-sm"
										></span>
										<span class="flex px-1 pb-0.5 text-xs"
											><span class="skeleton-line w-3/4"></span
											></span>
									</li>
								{/each}
							</ul>
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
										onAddItems={() => (pickerEmoteSlot = null)} />
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
			</main>
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
		accountUsername={accounts.selectedAccount.username}
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
		description={`${outfit.name || "This outfit"} will be deleted from @${username}'s outfits. This can't be undone.`}
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
		progressLabel="Discarding…"
		tone="warning"
		isWorking={false}
		error=""
		onConfirm={confirmDiscard}
		onCancel={() => (isConfirmingDiscard = false)} />
{/if}

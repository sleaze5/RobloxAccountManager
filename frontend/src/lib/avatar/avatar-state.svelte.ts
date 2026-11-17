import { accountBackend } from "../backend/bridge"
import type {
	Avatar,
	AvatarAccessoryAdjustment,
	AvatarAccessoryAdjustmentLimits,
	AvatarOutfit,
	AvatarRules,
	InventoryCategory,
	InventoryItem,
	RenderedModel,
} from "../backend/bridge"
import type { AccountStore } from "../accounts/account-store.svelte"
import { AccountScope } from "../accounts/account-scope"
import type { NotificationCenter } from "../notifications/notification-center.svelte"
import { errorMessage } from "../shared/errors"
import { KeyedLoad } from "../shared/keyed-load.svelte"
import {
	adjustAsset,
	avatarChangeOf,
	clampedAdjustment,
	clearEmoteSlot,
	draftOf,
	isDraftChanged,
	isSameDraft,
	mirroredAdjustment,
	putEmoteInSlot,
	takeOffItem,
	wearItem,
} from "./avatar-draft"
import type { AvatarDraft } from "./avatar-draft"

const maxDraftHistoryLength = 100

interface CopiedAdjustment {
	sourceAssetId: number
	sourceName: string
	adjustment: AvatarAccessoryAdjustment
}

// AvatarState shows how one Roblox user looks, as the account sees it. When
// the user is the account itself, it also edits a draft of the avatar and
// saves it.
export class AvatarState {
	avatar = $state.raw<Avatar | null>(null)
	loading = $state(false)
	error = $state("")
	// What the account may change. undefined while it loads, and null when it
	// failed.
	rules = $state.raw<AvatarRules | null | undefined>(undefined)
	// The BrickColors of the palette by their hex colors, which name the body
	// colors.
	paletteColorByHex = $derived(
		new Map(
			(this.rules?.bodyColorPalette ?? []).map((paletteColor) => [
				paletteColor.hexColor,
				paletteColor,
			]),
		),
	)
	draft = $state.raw<AvatarDraft | null>(null)
	// The drafts before each change, newest last, which undo restores.
	draftHistory = $state.raw<AvatarDraft[]>([])
	// The accessory placement that Copy placement took, which Paste puts on
	// another accessory.
	copiedAdjustment = $state.raw<CopiedAdjustment | null>(null)
	isSaving = $state(false)
	saveError = $state("")
	// Whether Roblox changed the avatar with a save or an outfit but the changed
	// avatar could not be read, so avatar still shows how it looked before. The
	// draft stays locked until a refresh reads the changed avatar, because a
	// save from it would undo the change.
	isAvatarOutdated = $state(false)
	// Whether Roblox still renders the full-body image after a change.
	isRenderingImage = $state(false)
	isRedrawingImage = $state(false)
	// The full-body image of the draft, which Roblox renders without saving
	// the draft. It stays while the preview of a newer draft renders.
	previewImageUrl = $state("")
	isRenderingPreview = $state(false)
	previewError = $state("")
	// The 3D avatar, which loads only when the page shows it. Its key is the
	// full-body image that it matches, because Roblox renders both from the
	// same avatar.
	readonly model: KeyedLoad<RenderedModel>
	outfits = $state.raw<AvatarOutfit[] | null>(null)
	outfitsNextCursor = $state("")
	outfitsLoading = $state(false)
	outfitsLoadingMore = $state(false)
	outfitsError = $state("")
	// The outfit that the account puts on, or 0.
	wearingOutfitId = $state(0)
	isChangingOutfit = $state(false)
	outfitActionError = $state("")
	// How each opened outfit looks, by outfit ID. null means that the request
	// failed.
	outfitDetailsById = $state.raw(new Map<number, Avatar | null>())
	readonly userId: number
	readonly isEditable: boolean
	readonly #scope: AccountScope
	readonly #accounts: AccountStore
	readonly #notifications: NotificationCenter
	// The draft that previewImageUrl shows.
	#previewDraft = $state.raw<AvatarDraft | null>(null)
	// The draft that the running preview request renders.
	#renderingPreviewDraft: AvatarDraft | null = null
	#previewRequest: ReturnType<typeof accountBackend.RenderAvatarPreview> | null = null
	#loadRequest: ReturnType<typeof accountBackend.GetAvatar> | null = null
	#outfitsRequest: ReturnType<typeof accountBackend.GetAvatarOutfits> | null = null
	// The key of the last draft change. A change with the same key continues
	// it, so that dragging a slider is one step in the history.
	#lastChangeKey = ""

	constructor(
		accountId: number,
		userId: number,
		store: AccountStore,
		notifications: NotificationCenter,
	) {
		this.#scope = new AccountScope(accountId, store)
		this.model = new KeyedLoad(this.#scope)
		this.#accounts = store
		this.#notifications = notifications
		this.userId = userId
		this.isEditable = Number(store.selectedAccount.userId) === userId
	}

	get accountId(): number {
		return this.#scope.accountId
	}

	get isChanged(): boolean {
		return (
			!!this.avatar &&
			!!this.draft &&
			!this.isAvatarOutdated &&
			isDraftChanged(this.avatar, this.draft)
		)
	}

	// Whether previewImageUrl shows the current draft.
	get isPreviewCurrent(): boolean {
		return (
			this.isChanged &&
			!!this.previewImageUrl &&
			this.#previewDraft === this.draft
		)
	}

	// Whether a save or an outfit change runs. It shows the changed avatar when
	// it finishes, so a read that overlaps it could show the avatar from before.
	get isChangingAvatar(): boolean {
		return this.isSaving || this.wearingOutfitId !== 0
	}

	// Whether the draft must not change now: a running request replaces it when
	// it finishes, or the avatar changed and could not be read back.
	get isDraftLocked(): boolean {
		return this.loading || this.isChangingAvatar || this.isAvatarOutdated
	}

	// Whether an outfit can replace the avatar now: the draft has no changes,
	// and no other request replaces the avatar.
	get canWearOutfit(): boolean {
		return !this.isChanged && !this.loading && !this.isChangingAvatar
	}

	get maxWornCountByTypeName(): Record<string, number | undefined> {
		return this.rules?.maxWornCountByTypeName ?? {}
	}

	get canUndo(): boolean {
		return this.draftHistory.length > 0 && !this.isDraftLocked
	}

	// adjustmentLimitsOf returns how far the account may adjust an accessory of
	// the type, or undefined when it may not adjust the type.
	adjustmentLimitsOf(typeName: string): AvatarAccessoryAdjustmentLimits | undefined {
		return this.rules?.accessoryAdjustmentLimitsByTypeName?.[typeName]
	}

	async refresh(): Promise<void> {
		if (!this.#scope.isCurrent || this.isChangingAvatar) return
		void this.#loadRequest?.cancel()
		const request = accountBackend.GetAvatar(this.#scope.accountId, this.userId)
		this.#loadRequest = request
		this.loading = true
		this.error = ""
		try {
			const avatar = await this.#scope.track(request)
			if (request !== this.#loadRequest || !this.#scope.isCurrent) return
			if (this.isAvatarOutdated) this.#showChangedAvatar(avatar)
			else this.#showAvatar(avatar)
		} catch (error) {
			if (request !== this.#loadRequest || !this.#scope.isCurrent) return
			this.error = errorMessage(error, "Could not load the avatar. Try again.")
		} finally {
			if (request === this.#loadRequest) {
				this.#loadRequest = null
				this.loading = false
			}
		}
	}

	// loadRules also runs for another user's avatar, because the palette names
	// the BrickColors of the body colors.
	async loadRules(): Promise<void> {
		this.rules = undefined
		let rules: AvatarRules | null
		try {
			rules = await this.#scope.track(
				accountBackend.GetAvatarRules(this.#scope.accountId),
			)
		} catch {
			// The editor shows the rules as unavailable, and the backend logs the
			// failure.
			rules = null
		}
		if (this.#scope.isCurrent) this.rules = rules
	}

	// changeDraft adds a step to the history, unless the change continues the
	// previous change with the same changeKey, such as a slider that moves.
	changeDraft(change: (draft: AvatarDraft) => AvatarDraft, changeKey = ""): void {
		if (!this.draft || this.isDraftLocked) return
		const previousDraft = this.draft,
			nextDraft = change(previousDraft)
		if (nextDraft === previousDraft) return
		if (!changeKey || changeKey !== this.#lastChangeKey) {
			this.draftHistory = [...this.draftHistory, previousDraft]
		}
		this.#lastChangeKey = changeKey
		this.draft = nextDraft
		if (!changeKey) this.#trimDraftHistory()
	}

	// finishDraftChange ends the change that continues with the same key, such
	// as when a slider is released. A change that ends where it began, such as
	// a slider dragged away and back, leaves no step in the history.
	finishDraftChange(): void {
		const previousDraft = this.draftHistory.at(-1)
		if (
			this.#lastChangeKey &&
			previousDraft &&
			this.draft &&
			isSameDraft(previousDraft, this.draft)
		) {
			this.draft = previousDraft
			this.draftHistory = this.draftHistory.slice(0, -1)
		}
		this.#lastChangeKey = ""
		this.#trimDraftHistory()
	}

	// #trimDraftHistory drops the oldest steps beyond the limit. It runs only
	// once a change is finished, so that a slider that ends where it began
	// never costs an older step.
	#trimDraftHistory(): void {
		if (this.draftHistory.length > maxDraftHistoryLength) {
			this.draftHistory = this.draftHistory.slice(-maxDraftHistoryLength)
		}
	}

	undo(): void {
		if (!this.canUndo) return
		this.draft = this.draftHistory.at(-1) ?? this.draft
		this.draftHistory = this.draftHistory.slice(0, -1)
		this.#lastChangeKey = ""
	}

	adjust(
		assetId: number,
		adjustment: AvatarAccessoryAdjustment,
		changeKey = "",
	): void {
		this.changeDraft((draft) => adjustAsset(draft, assetId, adjustment), changeKey)
	}

	copyAdjustment(
		sourceAssetId: number,
		sourceName: string,
		adjustment: AvatarAccessoryAdjustment,
	): void {
		this.copiedAdjustment = { sourceAssetId, sourceName, adjustment }
	}

	// pasteAdjustment puts the copied placement on the accessory, mirrored for
	// the other side when isMirrored is true, and kept within the limits of
	// the accessory type.
	pasteAdjustment(
		assetId: number,
		limits: AvatarAccessoryAdjustmentLimits,
		isMirrored: boolean,
	): void {
		if (!this.copiedAdjustment) return
		const copied = this.copiedAdjustment.adjustment,
			pasted = isMirrored ? mirroredAdjustment(copied) : copied
		this.adjust(assetId, clampedAdjustment(pasted, limits))
	}

	// wear puts the item on the draft. An item that the saved avatar wears
	// comes back with its saved placement and layer, which a save keeps.
	wear(item: InventoryItem, category: InventoryCategory): void {
		const maxWornCount = this.maxWornCountByTypeName[item.typeName] ?? 0,
			savedAsset = this.avatar?.assets?.find(
				(asset) => asset.item.itemId === item.itemId,
			),
			wornAsset = savedAsset ?? { category, item, adjustment: null, layer: null }
		this.changeDraft((draft) => wearItem(draft, wornAsset, maxWornCount))
	}

	takeOff(assetId: number): void {
		this.changeDraft((draft) => takeOffItem(draft, assetId))
	}

	putEmote(slot: number, item: InventoryItem): void {
		this.changeDraft((draft) => putEmoteInSlot(draft, slot, item))
	}

	clearEmote(slot: number): void {
		this.changeDraft((draft) => clearEmoteSlot(draft, slot))
	}

	discardChanges(): void {
		if (this.avatar && !this.isDraftLocked) {
			this.#resetDraft(this.avatar)
			this.#clearPreview()
			this.saveError = ""
		}
	}

	async save(): Promise<void> {
		const draft = this.draft,
			savedAvatar = this.avatar
		if (!draft || !savedAvatar || !this.isChanged || this.isDraftLocked) return
		this.isSaving = true
		this.saveError = ""
		const nameByAssetId = new Map(
			draft.assets.map((asset) => [asset.item.itemId, asset.item.name]),
		)
		try {
			const saved = await this.#scope.track(
				accountBackend.SaveAvatar(
					this.#scope.accountId,
					avatarChangeOf(draft, savedAvatar),
				),
			)
			if (!this.#scope.isCurrent) return
			if (saved.avatar) this.#showChangedAvatar(saved.avatar)
			else this.isAvatarOutdated = true
			this.#showChangeResult({
				id: "avatar-save",
				title: "Avatar saved",
				successMessage: "Roblox is updating the avatar image.",
				isAvatarRead: !!saved.avatar,
				skippedAssetIds: saved.skippedAssetIds ?? [],
				unkeptAdjustmentAssetIds: saved.unkeptAdjustmentAssetIds ?? [],
				nameByAssetId,
			})
		} catch (error) {
			if (this.#scope.isCurrent) {
				this.saveError = errorMessage(
					error,
					"Could not save the avatar. Try again.",
				)
			}
		} finally {
			this.isSaving = false
		}
	}

	async loadOutfits(): Promise<void> {
		await this.#loadOutfits("")
	}

	async loadMoreOutfits(): Promise<void> {
		if (!this.outfitsNextCursor || this.outfitsLoading || this.outfitsLoadingMore) {
			return
		}
		await this.#loadOutfits(this.outfitsNextCursor)
	}

	clearOutfitActionError(): void {
		this.outfitActionError = ""
	}

	// loadOutfitDetails reads how the outfit looks, unless it is already
	// loaded. A failed read loads again when the outfit opens again.
	async loadOutfitDetails(outfit: AvatarOutfit): Promise<void> {
		if (this.outfitDetailsById.get(outfit.outfitId)) return
		if (this.outfitDetailsById.has(outfit.outfitId)) {
			const outfitDetailsById = new Map(this.outfitDetailsById)
			outfitDetailsById.delete(outfit.outfitId)
			this.outfitDetailsById = outfitDetailsById
		}
		let outfitDetails: Avatar | null
		try {
			outfitDetails = await this.#scope.track(
				accountBackend.GetAvatarOutfitDetails(
					this.#scope.accountId,
					outfit.outfitId,
				),
			)
		} catch {
			// The outfit dialog shows the outfit as unavailable, and the backend
			// logs the failure.
			outfitDetails = null
		}
		if (this.#scope.isCurrent) {
			this.outfitDetailsById = new Map(this.outfitDetailsById).set(
				outfit.outfitId,
				outfitDetails,
			)
		}
	}

	// wearOutfit puts the outfit on the saved avatar, and reports whether
	// Roblox put it on. The page allows it only while the draft has no changes,
	// because the outfit replaces the avatar.
	async wearOutfit(outfit: AvatarOutfit): Promise<boolean> {
		if (!this.canWearOutfit) return false
		this.wearingOutfitId = outfit.outfitId
		this.outfitActionError = ""
		try {
			const worn = await this.#scope.track(
				accountBackend.WearAvatarOutfit(this.#scope.accountId, outfit.outfitId),
			)
			if (!this.#scope.isCurrent) return false
			if (worn.avatar) this.#showChangedAvatar(worn.avatar)
			else this.isAvatarOutdated = true
			const outfitAssets =
				this.outfitDetailsById.get(outfit.outfitId)?.assets ?? []
			this.#showChangeResult({
				id: "avatar-outfit-wear",
				title: "Outfit put on",
				successMessage: `The avatar now wears ${outfit.name || "the outfit"}.`,
				isAvatarRead: !!worn.avatar,
				skippedAssetIds: worn.skippedAssetIds ?? [],
				unkeptAdjustmentAssetIds: worn.unkeptAdjustmentAssetIds ?? [],
				nameByAssetId: new Map(
					outfitAssets.map((asset) => [asset.item.itemId, asset.item.name]),
				),
			})
			return true
		} catch (error) {
			if (this.#scope.isCurrent) {
				this.outfitActionError = errorMessage(
					error,
					"Could not put on the outfit. Try again.",
				)
			}
			return false
		} finally {
			this.wearingOutfitId = 0
		}
	}

	// createOutfit saves the saved avatar as an outfit, and reports whether
	// Roblox created it.
	async createOutfit(name: string): Promise<boolean> {
		return this.#changeOutfits(
			() => accountBackend.CreateAvatarOutfit(this.#scope.accountId, name),
			"Could not save the outfit. Try again.",
		)
	}

	async renameOutfit(outfit: AvatarOutfit, name: string): Promise<boolean> {
		return this.#changeOutfits(
			() =>
				accountBackend.RenameAvatarOutfit(
					this.#scope.accountId,
					outfit.outfitId,
					name,
				),
			"Could not rename the outfit. Try again.",
		)
	}

	async deleteOutfit(outfit: AvatarOutfit): Promise<boolean> {
		return this.#changeOutfits(
			() =>
				accountBackend.DeleteAvatarOutfit(
					this.#scope.accountId,
					outfit.outfitId,
				),
			"Could not delete the outfit. Try again.",
		)
	}

	// redrawImage asks Roblox to render the avatar images again, then shows the
	// new image once Roblox renders it.
	async redrawImage(): Promise<void> {
		if (!this.isEditable || this.isRedrawingImage || this.isRenderingImage) return
		this.isRedrawingImage = true
		try {
			await this.#scope.track(
				accountBackend.RedrawAvatarImage(this.#scope.accountId),
			)
			if (this.#scope.isCurrent) {
				void this.#waitForImage(this.avatar?.imageUrl ?? "")
			}
		} catch (error) {
			if (this.#scope.isCurrent) {
				this.#notifications.show({
					id: "avatar-redraw",
					tone: "error",
					title: "Avatar not redrawn",
					message: errorMessage(
						error,
						"Could not redraw the avatar. Try again.",
					),
				})
			}
		} finally {
			this.isRedrawingImage = false
		}
	}

	// renderPreview asks Roblox to render the draft without saving it, unless
	// the preview of the draft is already shown or rendering.
	async renderPreview(): Promise<void> {
		const draft = this.draft,
			savedAvatar = this.avatar
		if (
			!this.#scope.isCurrent ||
			!this.isEditable ||
			!draft ||
			!savedAvatar ||
			!this.isChanged
		) {
			return
		}
		if (this.#isPreviewShownOrRendering(draft)) return
		void this.#previewRequest?.cancel()
		const request = accountBackend.RenderAvatarPreview(
			this.#scope.accountId,
			avatarChangeOf(draft, savedAvatar),
		)
		this.#previewRequest = request
		this.#renderingPreviewDraft = draft
		this.isRenderingPreview = true
		this.previewError = ""
		try {
			const previewImageUrl = await this.#scope.track(request)
			if (request !== this.#previewRequest || !this.#scope.isCurrent) return
			this.previewImageUrl = previewImageUrl
			this.#previewDraft = draft
		} catch (error) {
			if (request !== this.#previewRequest || !this.#scope.isCurrent) return
			this.previewImageUrl = ""
			this.#previewDraft = null
			this.previewError = errorMessage(
				error,
				"Could not preview the changes. Save to see them.",
			)
		} finally {
			if (request === this.#previewRequest) {
				this.#previewRequest = this.#renderingPreviewDraft = null
				this.isRenderingPreview = false
			}
		}
	}

	#isPreviewShownOrRendering(draft: AvatarDraft): boolean {
		return (
			(draft === this.#previewDraft && !!this.previewImageUrl) ||
			(draft === this.#renderingPreviewDraft && this.isRenderingPreview)
		)
	}

	// loadModel reads the 3D avatar that matches the shown full-body image.
	async loadModel(shownImageUrl: string): Promise<void> {
		await this.model.load(
			shownImageUrl,
			() => accountBackend.GetAvatarModel(this.#scope.accountId, this.userId),
			"Could not load the 3D avatar. Try again.",
		)
	}

	dispose(): void {
		this.#loadRequest = this.#outfitsRequest = this.#previewRequest = null
		this.#scope.dispose()
	}

	// #showAvatar replaces the draft, because the shown avatar is the one that
	// a save would change.
	#showAvatar(avatar: Avatar): void {
		const previousImageUrl = this.avatar?.imageUrl ?? ""
		this.avatar = avatar.imageUrl
			? avatar
			: { ...avatar, imageUrl: previousImageUrl }
		this.#resetDraft(avatar)
		this.#clearPreview()
		this.saveError = ""
		this.isAvatarOutdated = false
		if (!avatar.imageUrl && this.isEditable) {
			void this.#waitForImage(previousImageUrl)
		}
	}

	// #showChangeResult reports what Roblox did not keep from a change, in one
	// notification that names both the skipped items and the accessories that
	// Roblox placed differently. Without a reread avatar and nothing to warn
	// about, the page's outdated notice is the only feedback.
	#showChangeResult(result: {
		id: string
		title: string
		successMessage: string
		isAvatarRead: boolean
		skippedAssetIds: number[]
		unkeptAdjustmentAssetIds: number[]
		nameByAssetId: Map<number, string>
	}): void {
		const nameOf = (assetId: number): string =>
				result.nameByAssetId.get(assetId) || `Item ${assetId}`,
			skippedNames = result.skippedAssetIds.map(nameOf),
			unkeptAdjustmentNames = result.unkeptAdjustmentAssetIds.map(nameOf),
			warnings = [
				skippedNames.length > 0 &&
					`Roblox didn't put on ${skippedNames.join(", ")}. The account may not own ${skippedNames.length === 1 ? "it" : "them"}.`,
				unkeptAdjustmentNames.length > 0 &&
					`Roblox placed ${unkeptAdjustmentNames.join(", ")} differently than requested. Check the Adjustments tab.`,
			].filter((warning) => warning !== false)
		if (warnings.length > 0) {
			this.#notifications.show({
				id: result.id,
				tone: "warning",
				title: result.title,
				message: warnings.join(" "),
				durationMs: 10_000,
			})
		} else if (result.isAvatarRead) {
			this.#notifications.show({
				id: result.id,
				tone: "success",
				title: result.title,
				message: result.successMessage,
			})
		}
	}

	#resetDraft(avatar: Avatar): void {
		this.draft = draftOf(avatar)
		this.draftHistory = []
		this.#lastChangeKey = ""
	}

	#clearPreview(): void {
		void this.#previewRequest?.cancel()
		this.#previewRequest = this.#previewDraft = this.#renderingPreviewDraft = null
		this.previewImageUrl = this.previewError = ""
		this.isRenderingPreview = false
	}

	// #showChangedAvatar also shows the new image in the account list. Roblox
	// keeps serving the previous image for a few seconds after a change, so an
	// image that equals the previous one waits for the new one, as a missing
	// image does in #showAvatar.
	#showChangedAvatar(avatar: Avatar): void {
		const previousImageUrl = this.avatar?.imageUrl ?? ""
		this.#showAvatar(avatar)
		if (!avatar.imageUrl) return
		if (avatar.imageUrl === previousImageUrl) {
			void this.#waitForImage(previousImageUrl)
		} else {
			void this.#refreshAccountImage()
		}
	}

	// #waitForImage keeps the previous image while Roblox renders the new one,
	// then shows the new one in the avatar page and the account list.
	async #waitForImage(previousImageUrl: string): Promise<void> {
		if (this.isRenderingImage) return
		this.isRenderingImage = true
		try {
			const imageUrl = await this.#scope.track(
				accountBackend.WaitForAvatarImage(
					this.#scope.accountId,
					previousImageUrl,
				),
			)
			if (!this.#scope.isCurrent || !this.avatar) return
			this.avatar = { ...this.avatar, imageUrl }
			void this.#refreshAccountImage()
		} catch {
			// The page keeps the previous image, and the backend logs the failure.
		} finally {
			this.isRenderingImage = false
		}
	}

	async #refreshAccountImage(): Promise<void> {
		try {
			await this.#accounts.refreshAccountAvatar(this.#scope.accountId)
		} catch {
			// The account list keeps its previous image until the next refresh.
		}
	}

	async #loadOutfits(cursor: string): Promise<void> {
		if (!this.#scope.isCurrent) return
		void this.#outfitsRequest?.cancel()
		const isMore = cursor !== "",
			request = accountBackend.GetAvatarOutfits(
				this.#scope.accountId,
				this.userId,
				cursor,
			)
		this.#outfitsRequest = request
		this.outfitsLoading = !isMore
		this.outfitsLoadingMore = isMore
		this.outfitsError = ""
		try {
			const page = await this.#scope.track(request)
			if (request !== this.#outfitsRequest || !this.#scope.isCurrent) return
			const outfits = page.outfits ?? []
			this.outfits = isMore ? [...(this.outfits ?? []), ...outfits] : outfits
			this.outfitsNextCursor = page.nextCursor
		} catch (error) {
			if (request === this.#outfitsRequest && this.#scope.isCurrent) {
				this.outfitsError = errorMessage(
					error,
					isMore
						? "Could not load more outfits. Try again."
						: "Could not load the outfits. Try again.",
				)
			}
		} finally {
			if (request === this.#outfitsRequest) {
				this.#outfitsRequest = null
				this.outfitsLoading = this.outfitsLoadingMore = false
			}
		}
	}

	// #changeOutfits runs an outfit change and reloads the outfits, because
	// Roblox does not return the changed list.
	async #changeOutfits(
		change: () => ReturnType<typeof accountBackend.CreateAvatarOutfit>,
		fallback: string,
	): Promise<boolean> {
		if (this.isChangingOutfit) return false
		this.isChangingOutfit = true
		this.outfitActionError = ""
		try {
			await this.#scope.track(change())
			if (!this.#scope.isCurrent) return false
			void this.loadOutfits()
			return true
		} catch (error) {
			if (this.#scope.isCurrent) {
				this.outfitActionError = errorMessage(error, fallback)
			}
			return false
		} finally {
			this.isChangingOutfit = false
		}
	}
}

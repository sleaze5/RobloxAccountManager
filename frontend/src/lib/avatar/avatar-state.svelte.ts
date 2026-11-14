import { accountBackend } from "../backend/bridge"
import type {
	Avatar,
	AvatarOutfit,
	AvatarRules,
	InventoryCategory,
	InventoryItem,
} from "../backend/bridge"
import type { AccountStore } from "../accounts/account-store.svelte"
import { AccountScope } from "../accounts/account-scope"
import type { NotificationCenter } from "../notifications/notification-center.svelte"
import { errorMessage } from "../shared/errors"
import {
	avatarChangeOf,
	clearEmoteSlot,
	draftOf,
	isDraftChanged,
	putEmoteInSlot,
	takeOffItem,
	wearItem,
} from "./avatar-draft"
import type { AvatarDraft } from "./avatar-draft"

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
	isSaving = $state(false)
	saveError = $state("")
	// Whether Roblox saved the draft but the saved avatar could not be read, so
	// avatar still shows how it looked before the save. The draft shows what was
	// saved until a refresh reads the saved avatar.
	isAvatarOutdated = $state(false)
	// Whether Roblox still renders the full-body image after a change.
	isRenderingImage = $state(false)
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
	#loadRequest: ReturnType<typeof accountBackend.GetAvatar> | null = null
	#outfitsRequest: ReturnType<typeof accountBackend.GetAvatarOutfits> | null = null

	constructor(
		accountId: number,
		userId: number,
		store: AccountStore,
		notifications: NotificationCenter,
	) {
		this.#scope = new AccountScope(accountId, store)
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

	// Whether the draft must not change now: a running request replaces it when
	// it finishes, or it holds a save that the page could not read back.
	get isDraftLocked(): boolean {
		return (
			this.loading ||
			this.isSaving ||
			this.wearingOutfitId !== 0 ||
			this.isAvatarOutdated
		)
	}

	// Whether an outfit can replace the avatar now: the draft has no changes,
	// and no other request replaces the avatar.
	get canWearOutfit(): boolean {
		return (
			!this.isChanged &&
			!this.loading &&
			!this.isSaving &&
			this.wearingOutfitId === 0
		)
	}

	get maxWornCountByTypeName(): Record<string, number | undefined> {
		return this.rules?.maxWornCountByTypeName ?? {}
	}

	async refresh(): Promise<void> {
		if (!this.#scope.isCurrent) return
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

	changeDraft(change: (draft: AvatarDraft) => AvatarDraft): void {
		if (this.draft && !this.isDraftLocked) this.draft = change(this.draft)
	}

	wear(item: InventoryItem, category: InventoryCategory): void {
		const maxWornCount = this.maxWornCountByTypeName[item.typeName] ?? 0
		this.changeDraft((draft) => wearItem(draft, item, category, maxWornCount))
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
			this.draft = draftOf(this.avatar)
			this.saveError = ""
		}
	}

	async save(): Promise<void> {
		const draft = this.draft
		if (!draft || !this.isChanged || this.isDraftLocked) return
		this.isSaving = true
		this.saveError = ""
		const nameByAssetId = new Map(
			draft.assets.map((asset) => [asset.item.itemId, asset.item.name]),
		)
		try {
			const saved = await this.#scope.track(
				accountBackend.SaveAvatar(this.#scope.accountId, avatarChangeOf(draft)),
			)
			if (!this.#scope.isCurrent) return
			if (!saved.avatar) {
				this.isAvatarOutdated = true
				return
			}
			this.#showChangedAvatar(saved.avatar)
			const skippedNames = (saved.skippedAssetIds ?? []).map(
				(assetId) => nameByAssetId.get(assetId) || `Item ${assetId}`,
			)
			this.#notifications.show(
				skippedNames.length > 0
					? {
							id: "avatar-save",
							tone: "warning",
							title: "Avatar saved",
							message: `Roblox didn't put on ${skippedNames.join(", ")}. The account may not own ${skippedNames.length === 1 ? "it" : "them"}.`,
							durationMs: 10_000,
						}
					: {
							id: "avatar-save",
							tone: "success",
							title: "Avatar saved",
							message: "Roblox is updating the avatar image.",
						},
			)
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
			const avatar = await this.#scope.track(
				accountBackend.WearAvatarOutfit(this.#scope.accountId, outfit.outfitId),
			)
			if (!this.#scope.isCurrent) return false
			this.#showChangedAvatar(avatar)
			this.#notifications.show({
				id: "avatar-outfit-wear",
				tone: "success",
				title: "Outfit put on",
				message: `The avatar now wears ${outfit.name || "the outfit"}.`,
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

	dispose(): void {
		this.#loadRequest = this.#outfitsRequest = null
		this.#scope.dispose()
	}

	// #showAvatar replaces the draft, because the shown avatar is the one that
	// a save would change.
	#showAvatar(avatar: Avatar): void {
		const previousImageUrl = this.avatar?.imageUrl ?? ""
		this.avatar = avatar.imageUrl
			? avatar
			: { ...avatar, imageUrl: previousImageUrl }
		this.draft = draftOf(avatar)
		this.saveError = ""
		this.isAvatarOutdated = false
		if (!avatar.imageUrl && this.isEditable) void this.#waitForImage()
	}

	// #showChangedAvatar also shows the new image in the account list. While
	// Roblox renders the image, #waitForImage updates the list once it is ready.
	#showChangedAvatar(avatar: Avatar): void {
		this.#showAvatar(avatar)
		if (avatar.imageUrl) void this.#refreshAccountImage()
	}

	// #waitForImage keeps the previous image while Roblox renders the new one,
	// then shows the new one in the avatar page and the account list.
	async #waitForImage(): Promise<void> {
		if (this.isRenderingImage) return
		this.isRenderingImage = true
		try {
			const imageUrl = await this.#scope.track(
				accountBackend.WaitForAvatarImage(this.#scope.accountId),
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

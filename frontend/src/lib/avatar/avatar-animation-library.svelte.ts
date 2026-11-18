import { accountBackend, InventoryCategory } from "../backend/bridge"
import type { AccountStore } from "../accounts/account-store.svelte"
import { AccountScope } from "../accounts/account-scope"
import { animationChoiceOfItem } from "../model-viewer/animation-choices"
import type {
	AnimationChoice,
	AnimationChoiceGroup,
	AnimationLibrary,
	OwnedAnimations,
} from "../model-viewer/model-subject"
import { errorMessage } from "../shared/errors"

// The picker searches what it loaded, so it loads every page of a category up
// to this many, which covers nearly every inventory.
const maxOwnedPageCount = 10

// OwnedAnimationList loads every page of one inventory category of the user.
class OwnedAnimationList implements OwnedAnimations {
	choices = $state.raw<AnimationChoice[] | undefined>(undefined)
	isLoading = $state(false)
	error = $state("")
	readonly #scope: AccountScope
	readonly #userId: number
	readonly #category: InventoryCategory
	readonly #group: AnimationChoiceGroup
	readonly #noun: string

	constructor(
		scope: AccountScope,
		userId: number,
		category: InventoryCategory,
		group: AnimationChoiceGroup,
		noun: string,
	) {
		this.#scope = scope
		this.#userId = userId
		this.#category = category
		this.#group = group
		this.#noun = noun
	}

	async load(): Promise<void> {
		if (this.isLoading || this.choices || !this.#scope.isCurrent) return
		this.isLoading = true
		this.error = ""
		try {
			const choices = await this.#loadPages("", maxOwnedPageCount)
			if (choices) this.choices = choices
		} catch (error) {
			if (!this.#scope.isCurrent) return
			this.error = errorMessage(
				error,
				`Could not load the ${this.#noun}. Try again.`,
			)
		} finally {
			this.isLoading = false
		}
	}

	// #loadPages reads the page at the cursor and the pages after it, up to
	// pageCount pages, one after another, because each page names the next.
	// It returns undefined when the scope ended.
	async #loadPages(
		cursor: string,
		pageCount: number,
	): Promise<AnimationChoice[] | undefined> {
		const page = await this.#scope.track(
			accountBackend.GetInventoryPage(
				this.#scope.accountId,
				this.#userId,
				this.#category,
				cursor,
			),
		)
		if (!this.#scope.isCurrent) return undefined
		const choices = (page.items ?? []).flatMap(
			(item) => animationChoiceOfItem(item, this.#group) ?? [],
		)
		if (!page.nextCursor || pageCount <= 1) return choices
		const nextChoices = await this.#loadPages(page.nextCursor, pageCount - 1)
		return nextChoices && [...choices, ...nextChoices]
	}
}

// AvatarAnimationLibrary offers the 3D avatar of one user the emotes and
// animations that the user owns, and the ones that the user names by ID.
export class AvatarAnimationLibrary implements AnimationLibrary {
	readonly ownedEmotes: OwnedAnimationList
	readonly ownedAnimations: OwnedAnimationList
	isFinding = $state(false)
	findError = $state("")
	readonly #scope: AccountScope
	#findRequest: ReturnType<typeof accountBackend.GetAvatarAnimation> | null = null

	constructor(accountId: number, userId: number, store: AccountStore) {
		this.#scope = new AccountScope(accountId, store)
		this.ownedEmotes = new OwnedAnimationList(
			this.#scope,
			userId,
			InventoryCategory.InventoryEmotes,
			"ownedEmote",
			"owned emotes",
		)
		this.ownedAnimations = new OwnedAnimationList(
			this.#scope,
			userId,
			InventoryCategory.InventoryAnimations,
			"ownedAnimation",
			"owned animations",
		)
	}

	// loadOwned loads what the user owns once, and again after it failed.
	loadOwned(): void {
		void this.ownedEmotes.load()
		void this.ownedAnimations.load()
	}

	async findAnimation(reference: string): Promise<AnimationChoice | undefined> {
		if (!this.#scope.isCurrent) return undefined
		void this.#findRequest?.cancel()
		const request = accountBackend.GetAvatarAnimation(
			this.#scope.accountId,
			reference,
		)
		this.#findRequest = request
		this.isFinding = true
		this.findError = ""
		try {
			const item = await this.#scope.track(request)
			if (request !== this.#findRequest || !this.#scope.isCurrent)
				return undefined
			const choice = animationChoiceOfItem(item, "added")
			if (!choice) this.findError = "This item is not an emote or an animation."
			return choice
		} catch (error) {
			if (request !== this.#findRequest || !this.#scope.isCurrent)
				return undefined
			this.findError = errorMessage(
				error,
				"Could not find the emote or animation. Try again.",
			)
			return undefined
		} finally {
			if (request === this.#findRequest) {
				this.#findRequest = null
				this.isFinding = false
			}
		}
	}

	clearFindError(): void {
		this.findError = ""
	}

	dispose(): void {
		this.#scope.dispose()
	}
}

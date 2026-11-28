import type { AccountPage } from "../accounts/account-pages"
import { createContext } from "svelte"
import { menuPointFor } from "../shared/transitions"
import {
	accountSubpageSidebarMinimumWidth,
	inventorySidebarMinimumWidth,
	pageSidebarMinimumWidth,
	settingsSidebarMinimumWidth,
} from "./sidebar-widths"

// AccountContextMenu opens at a point, or beside its anchor, such as the
// Actions button of the selection, so that it never covers the button.
export interface AccountContextMenu {
	accountId: number
	x: number
	y: number
	anchor: HTMLElement | null
	submenuLeft: boolean
	// isFromProfile is true when the More actions button of the profile opens
	// the menu. The menu then leaves out what the profile already shows.
	isFromProfile: boolean
}

export interface ImageContextMenu {
	image: ViewedImage
	x: number
	y: number
}

// ViewedImage is an image that the image viewer can show. label names it, such
// as "@builderman's profile image".
interface ViewedImage {
	imageUrl: string
	label: string
}

type ActiveAccountMenu =
	| "add-account"
	| "empty-add-account"
	| "profile-tags"
	| "tag-filter"
	| null

export class WorkspaceState {
	accountPage = $state<"profile" | AccountPage>("profile")
	// The Chat button of a notification opens Chats with a direct chat to this
	// user. ChatsPage reads it once when it opens.
	chatUserIdToOpen = 0
	accountSettingsPage = $state<
		"details" | "privacy" | "servers" | "sessions" | "robux"
	>("details")
	accountSettingsCategory = $state<string | null>(null)
	accountSettingsQuery = $state("")
	accountSettingsScrollTop = 0
	accountSettingsScrollAnchor: { key: string; offset: number } | null = null
	sidebarCollapsed = $state(false)
	sidebarWidth = $state(pageSidebarMinimumWidth)
	settingsSidebarWidth = $state(settingsSidebarMinimumWidth)
	accountSettingsSidebarWidth = $state(accountSubpageSidebarMinimumWidth)
	inventorySidebarWidth = $state(inventorySidebarMinimumWidth)
	#profileLaunchOpen = $state(true)
	#subpageLaunchOpen = $state(false)
	activeAccountMenu = $state<ActiveAccountMenu>(null)
	accountContextMenu = $state<AccountContextMenu | null>(null)
	imageContextMenu = $state<ImageContextMenu | null>(null)
	viewedImage = $state<ViewedImage | null>(null)

	get launchOpen(): boolean {
		return this.accountPage === "profile"
			? this.#profileLaunchOpen
			: this.#subpageLaunchOpen
	}

	set launchOpen(open: boolean) {
		if (this.accountPage === "profile") this.#profileLaunchOpen = open
		else this.#subpageLaunchOpen = open
	}

	get hasOpenAccountMenu(): boolean {
		return (
			this.activeAccountMenu !== null ||
			this.accountContextMenu !== null ||
			this.imageContextMenu !== null
		)
	}

	closeAccountMenus(): void {
		this.activeAccountMenu = null
		this.accountContextMenu = null
		this.imageContextMenu = null
	}

	toggleAccountMenu(menu: Exclude<ActiveAccountMenu, null>): void {
		this.accountContextMenu = null
		this.imageContextMenu = null
		this.activeAccountMenu = this.activeAccountMenu === menu ? null : menu
	}

	openAccountContextMenu(
		event: MouseEvent | KeyboardEvent,
		accountId: number,
		isFromProfile: boolean,
	): void {
		event.preventDefault()
		event.stopPropagation()
		this.activeAccountMenu = null
		this.imageContextMenu = null
		this.accountContextMenu = {
			accountId,
			anchor: null,
			submenuLeft: false,
			...menuPointFor(event, 0),
			isFromProfile,
		}
	}

	openAccountContextMenuAtAnchor(anchor: HTMLElement, accountId: number): void {
		const bounds = anchor.getBoundingClientRect()
		this.activeAccountMenu = null
		this.imageContextMenu = null
		this.accountContextMenu = {
			accountId,
			anchor,
			submenuLeft: false,
			x: bounds.left,
			y: bounds.bottom,
			isFromProfile: false,
		}
	}

	openImageContextMenu(event: MouseEvent | KeyboardEvent, image: ViewedImage): void {
		event.preventDefault()
		event.stopPropagation()
		this.closeAccountMenus()
		if (!image.imageUrl) return
		this.imageContextMenu = { image, ...menuPointFor(event, 0) }
	}

	handleWindowClick(event: MouseEvent): void {
		const { target } = event
		if (target instanceof Element && target.closest("[data-account-menu]")) {
			return
		}

		this.closeAccountMenus()
	}
}

export const [getWorkspace, setWorkspace] = createContext<WorkspaceState>()

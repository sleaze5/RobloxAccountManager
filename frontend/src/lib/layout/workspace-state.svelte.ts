import { installPointerResize } from "../shared/pointer-resize"

export interface AccountContextMenu {
	accountId: number
	x: number
	y: number
	submenuLeft: boolean
}

export interface ImageContextMenu {
	accountId: number
	imageUrl: string
	x: number
	y: number
}

export type ActiveAccountMenu = "add-account" | "profile-tags" | "tag-filter" | null

export class WorkspaceState {
	accountPage = $state<"profile" | "settings" | "chats">("profile")
	accountSettingsCategory = $state("content-maturity")
	accountSettingsQuery = $state("")
	accountSettingsScrollTop = 0
	accountSettingsScrollAnchor: { key: string; offset: number } | null = null
	sidebarCollapsed = $state(false)
	sidebarWidth = $state(220)
	launchOpen = $state(true)
	launchHeight = $state(180)
	#launchMinimumHeight = 0
	activeAccountMenu = $state<ActiveAccountMenu>(null)
	accountContextMenu = $state<AccountContextMenu | null>(null)
	imageContextMenu = $state<ImageContextMenu | null>(null)

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

	openAccountContextMenu(event: MouseEvent | KeyboardEvent, accountId: number): void {
		event.preventDefault()
		event.stopPropagation()
		this.activeAccountMenu = null
		this.imageContextMenu = null
		const bounds = (event.currentTarget as HTMLElement).getBoundingClientRect(),
			atPointer = event instanceof MouseEvent && event.type === "contextmenu"

		this.accountContextMenu = {
			accountId,
			submenuLeft: false,
			x: atPointer ? event.clientX : bounds.left,
			y: atPointer ? event.clientY : bounds.bottom,
		}
	}

	openImageContextMenu(
		event: MouseEvent | KeyboardEvent,
		accountId: number,
		imageUrl: string,
	): void {
		event.preventDefault()
		event.stopPropagation()
		this.closeAccountMenus()
		if (!imageUrl || accountId === 0) return
		const bounds = (event.currentTarget as HTMLElement).getBoundingClientRect(),
			atPointer = event instanceof MouseEvent && event.type === "contextmenu"
		this.imageContextMenu = {
			accountId,
			imageUrl,
			x: atPointer ? event.clientX : bounds.left,
			y: atPointer ? event.clientY : bounds.bottom,
		}
	}

	handleWindowClick(event: MouseEvent): void {
		const { target } = event
		if (target instanceof Element && target.closest("[data-account-menu]")) {
			return
		}

		this.closeAccountMenus()
	}

	startLaunchResize(event: PointerEvent): void {
		if (!this.launchOpen || !(event.currentTarget instanceof HTMLElement)) {
			return
		}

		const handle = event.currentTarget,
			{ pointerId } = event,
			startY = event.clientY,
			startHeight = this.launchHeight
		handle.setPointerCapture(pointerId)

		const resize = (moveEvent: PointerEvent) => {
			this.resizeLaunch(startHeight + startY - moveEvent.clientY)
		}

		installPointerResize(handle, pointerId, resize)
	}

	resizeLaunch(height: number): void {
		const maximumHeight = Math.max(
			this.#launchMinimumHeight,
			window.innerHeight - 180,
		)
		this.launchHeight = Math.min(
			maximumHeight,
			Math.max(this.#launchMinimumHeight, height),
		)
	}

	setLaunchMinimumHeight(height: number, fitToContent: boolean): void {
		this.#launchMinimumHeight = height
		this.resizeLaunch(fitToContent ? height : this.launchHeight)
	}
}

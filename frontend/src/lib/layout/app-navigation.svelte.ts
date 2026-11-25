import { untrack } from "svelte"
import { SvelteSet } from "svelte/reactivity"
import { unsavedChangesGuard } from "./unsaved-changes-guard"

type ActivePage =
	| "accounts"
	| "social"
	| "groups"
	| "games"
	| "catalog"
	| "logs-explorer"
	| "settings"
type TabPage = Exclude<ActivePage, "settings">

const tabPageOrder: TabPage[] = [
	"accounts",
	"social",
	"groups",
	"games",
	"catalog",
	"logs-explorer",
]

const tabPageLabels: Record<TabPage, string> = {
	"accounts": "Accounts",
	"social": "Social",
	"groups": "Groups",
	"games": "Games",
	"catalog": "Marketplace",
	"logs-explorer": "Logs Explorer",
}

// AppNavigation is the page that the window shows. Settings replaces the tab
// pages and returns to the one that opened it.
export class AppNavigation {
	activePage = $state<ActivePage>("accounts")
	#settingsReturnPage = $state<TabPage>("accounts")
	// The tab pages that the user opened since the vault unlocked. They stay
	// mounted and hidden while another page shows, so that switching back
	// keeps their state.
	readonly #openedTabPages = new SvelteSet<TabPage>()

	constructor(isVaultUnlocked: () => boolean) {
		$effect(() => {
			const page = this.activePage
			if (page !== "settings" && isVaultUnlocked()) {
				untrack(() => this.#openedTabPages.add(page))
			}
		})
	}

	get settingsBackLabel(): string {
		return `Back to ${tabPageLabels[this.#settingsReturnPage]}`
	}

	isTabPageMounted(page: TabPage): boolean {
		return this.activePage === page || this.#openedTabPages.has(page)
	}

	forgetOpenedTabPages(): void {
		this.#openedTabPages.clear()
	}

	// showPage changes the page once a page with unsaved changes lets it close,
	// and then runs onShown.
	showPage(page: ActivePage, onShown?: () => void): void {
		if (page === this.activePage) {
			onShown?.()
			return
		}
		unsavedChangesGuard.leave(() => {
			this.activePage = page
			onShown?.()
		})
	}

	showSettings(): void {
		const returnPage = this.activePage
		this.showPage("settings", () => {
			if (returnPage !== "settings") this.#settingsReturnPage = returnPage
		})
	}

	leaveSettings(): void {
		this.activePage = this.#settingsReturnPage
	}

	showNeighborTabPage(step: 1 | -1): void {
		if (this.activePage === "settings") return
		const index = tabPageOrder.indexOf(this.activePage)
		this.showPage(
			tabPageOrder[(index + step + tabPageOrder.length) % tabPageOrder.length],
		)
	}
}

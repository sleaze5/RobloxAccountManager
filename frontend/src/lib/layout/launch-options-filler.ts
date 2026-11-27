import type { AccountStore } from "../accounts/account-store.svelte"
import { LaunchMethod } from "../backend/bridge"
import { focusAfterRender } from "../shared/focus-after-render"
import type { AppNavigation } from "./app-navigation.svelte"
import type { WorkspaceState } from "./workspace-state.svelte"

// LaunchOptionsFiller fills the launch options from another page, such as a
// server of a game, and shows them in the launch drawer of Accounts with the
// filled field focused.
export class LaunchOptionsFiller {
	readonly #store: AccountStore
	readonly #workspace: WorkspaceState
	readonly #navigation: AppNavigation

	constructor(
		store: AccountStore,
		workspace: WorkspaceState,
		navigation: AppNavigation,
	) {
		this.#store = store
		this.#workspace = workspace
		this.#navigation = navigation
	}

	fillServer(server: { placeId: string; serverId: string }): void {
		const input = this.#store.launchInput
		input.method = LaunchMethod.MethodPlace
		input.placeId = server.placeId
		input.serverId = server.serverId
		this.#showLaunchDrawer("[data-place-input]")
	}

	fillGame(placeId: number, serverId = ""): void {
		this.#store.launchInput.launchData = ""
		this.fillServer({ placeId: String(placeId), serverId })
	}

	// fillFollowUser follows the user into their server, such as the user who
	// sent an invite or a friend whose server their privacy settings hide.
	fillFollowUser(user: { userId: number; username: string }): void {
		const input = this.#store.launchInput
		input.method = LaunchMethod.MethodUser
		input.user = user.username || `id:${user.userId}`
		input.resolveCurrentGame = false
		this.#showLaunchDrawer("[data-launch-user]")
	}

	fillLink(link: string): void {
		const input = this.#store.launchInput
		input.method = LaunchMethod.MethodLink
		input.link = link
		input.launchData = ""
		this.#showLaunchDrawer("[data-launch-link]")
	}

	#showLaunchDrawer(focusSelector: string): void {
		this.#navigation.showPage("accounts", () => {
			this.#workspace.launchOpen = true
			void focusAfterRender(focusSelector)
		})
	}
}

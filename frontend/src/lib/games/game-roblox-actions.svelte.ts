import { accountBackend, GameVote } from "../backend/bridge"
import type { Game, GameAccountStatus } from "../backend/bridge"
import type { AccountScope } from "../accounts/account-scope"
import type { NotificationCenter } from "../notifications/notification-center.svelte"
import { errorMessage } from "../shared/errors"

// GameCountChange is how a Roblox action of the account changes the counts
// that the game details show.
export interface GameCountChange {
	favorites: number
	upVotes: number
	downVotes: number
}

// GameRobloxChange names a change that the account makes to a game on
// Roblox, which the user confirms first.
export type GameRobloxChange =
	| { kind: "favorite"; isFavorite: boolean }
	| { kind: "vote"; vote: GameVote }

// withGameCountChange applies a change that the account made on Roblox to
// the counts of a game, so that they match before the next refresh.
export function withGameCountChange(game: Game, change: GameCountChange): Game {
	const changed = (count: number, difference: number) =>
		Math.max(count + difference, 0)
	return {
		...game,
		favorites: changed(game.favorites, change.favorites),
		upVotes: changed(game.upVotes, change.upVotes),
		downVotes: changed(game.downVotes, change.downVotes),
	}
}

function voteCount(vote: GameVote, counted: GameVote): number {
	return vote === counted ? 1 : 0
}

// GameRobloxActions reads how the selected account relates to one game on
// Roblox and changes it: its favorite and its vote.
export class GameRobloxActions {
	status = $state.raw<GameAccountStatus | null>(null)
	hasFailed = $state(false)
	isSaving = $state(false)
	saveError = $state("")
	readonly #universeId: number
	readonly #scope: AccountScope
	readonly #notifications: NotificationCenter
	readonly #onCountChange: (change: GameCountChange) => void

	constructor(
		scope: AccountScope,
		universeId: number,
		notifications: NotificationCenter,
		onCountChange: (change: GameCountChange) => void,
	) {
		this.#scope = scope
		this.#universeId = universeId
		this.#notifications = notifications
		this.#onCountChange = onCountChange
	}

	isUnavailable(part: "favorite" | "vote" | "manage"): boolean {
		return this.hasFailed || !!this.status?.unavailable?.includes(part)
	}

	async load(): Promise<void> {
		try {
			const status = await this.#scope.track(
				accountBackend.GetGameAccountStatus(
					this.#scope.accountId,
					this.#universeId,
				),
			)
			if (!this.#scope.isCurrent) return
			this.status = status
			this.hasFailed = false
		} catch {
			// The backend logs the failure, and the actions show that they are unavailable.
			if (this.#scope.isCurrent) this.hasFailed = true
		}
	}

	// save makes the change on Roblox and reports whether Roblox saved it.
	async save(change: GameRobloxChange, gameName: string): Promise<boolean> {
		const status = this.status
		if (!status || this.isSaving) return false
		this.isSaving = true
		this.saveError = ""
		try {
			if (change.kind === "favorite") {
				await this.#scope.track(
					accountBackend.SetGameFavorite(
						this.#scope.accountId,
						this.#universeId,
						change.isFavorite,
					),
				)
			} else {
				await this.#scope.track(
					accountBackend.SetGameVote(
						this.#scope.accountId,
						this.#universeId,
						change.vote,
					),
				)
			}
			if (!this.#scope.isCurrent) return false
			this.#apply(status, change, gameName)
			return true
		} catch (error) {
			if (this.#scope.isCurrent) {
				this.saveError = errorMessage(
					error,
					"Roblox did not confirm the change. Refresh to check.",
				)
			}
			return false
		} finally {
			this.isSaving = false
		}
	}

	#apply(
		status: GameAccountStatus,
		change: GameRobloxChange,
		gameName: string,
	): void {
		if (change.kind === "favorite") {
			this.status = { ...status, isFavorite: change.isFavorite }
			this.#onCountChange({
				favorites: change.isFavorite ? 1 : -1,
				upVotes: 0,
				downVotes: 0,
			})
			this.#notifications.show({
				id: `game-favorite-${this.#universeId}`,
				title: change.isFavorite
					? "Added to favorites"
					: "Removed from favorites",
				tone: "success",
				message: `${gameName} on Roblox`,
			})
			return
		}
		const previousVote = status.vote.vote
		this.status = { ...status, vote: { ...status.vote, vote: change.vote } }
		this.#onCountChange({
			favorites: 0,
			upVotes:
				voteCount(change.vote, GameVote.GameVoteLike) -
				voteCount(previousVote, GameVote.GameVoteLike),
			downVotes:
				voteCount(change.vote, GameVote.GameVoteDislike) -
				voteCount(previousVote, GameVote.GameVoteDislike),
		})
		this.#notifications.show({
			id: `game-vote-${this.#universeId}`,
			title:
				change.vote === GameVote.GameVoteLike
					? "Game liked"
					: change.vote === GameVote.GameVoteDislike
						? "Game disliked"
						: "Vote removed",
			tone: "success",
			message: `${gameName} on Roblox`,
		})
	}
}

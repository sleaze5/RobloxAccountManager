import type { Game } from "../backend/bridge"
import type { ProfileStat } from "../shared/ProfileStats.svelte"

const numbers = new Intl.NumberFormat(),
	compactNumbers = new Intl.NumberFormat(undefined, {
		notation: "compact",
		maximumFractionDigits: 1,
	}),
	percent = new Intl.NumberFormat(undefined, {
		style: "percent",
		maximumFractionDigits: 1,
	})

function count(value: number, one: string, many: string): string {
	return `${numbers.format(value)} ${value === 1 ? one : many}`
}

// countStat shows one count of a listed game, which is missing until the game
// loads.
function countStat(
	id: string,
	value: number | undefined,
	one: string,
	many: string,
	isLoading: boolean,
): ProfileStat {
	return {
		id,
		value: value === undefined ? "" : compactNumbers.format(value),
		label: value === 1 ? one : many,
		description: value === undefined ? "" : count(value, one, many),
		isLoading,
	}
}

function ratingStat(
	game: Game | null,
	isWaitingForGame: boolean,
	votesState: { isLoaded: boolean; isPending: boolean },
): ProfileStat {
	const votes = game ? game.upVotes + game.downVotes : 0,
		hasVotes = !!game && votesState.isLoaded
	if (!game || !hasVotes || !votes)
		return {
			id: "rating",
			value: hasVotes ? "0" : "",
			label: hasVotes ? "votes" : "liked",
			description: "No votes",
			isLoading: isWaitingForGame || votesState.isPending,
		}
	return {
		id: "rating",
		value: percent.format(game.upVotes / votes),
		label: "liked",
		description: `${count(game.upVotes, "like", "likes")} and ${count(game.downVotes, "dislike", "dislikes")}`,
	}
}

// gameHeadlineStats lists the players, rating, visits, and favorites of a
// game. Roblox reports them per universe, so a subplace shows its universe's
// values. An unlisted game has no counts, so it has no stats once it loads.
// Until the game loads, every stat holds its place with a placeholder.
export function gameHeadlineStats(
	game: Game | null,
	isWaitingForGame: boolean,
	votesState: { isLoaded: boolean; isPending: boolean },
): ProfileStat[] {
	// The develop API lists an unlisted game without its counts.
	const listedGame = game?.isUnlisted ? undefined : game
	if (!isWaitingForGame && !listedGame) return []
	const playing = listedGame?.playing
	return [
		{
			id: "playing",
			value: playing === undefined ? "" : numbers.format(playing),
			label: "playing",
			description:
				playing === undefined
					? ""
					: count(
							playing,
							"player in the game now",
							"players in the game now",
						),
			isLoading: isWaitingForGame,
			isLive: (playing ?? 0) > 0,
		},
		ratingStat(game, isWaitingForGame, votesState),
		countStat("visits", listedGame?.visits, "visit", "visits", isWaitingForGame),
		countStat(
			"favorites",
			listedGame?.favorites,
			"favorite",
			"favorites",
			isWaitingForGame,
		),
	]
}

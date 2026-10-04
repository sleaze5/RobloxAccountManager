import type { GamePlace } from "../backend/bridge"

export interface PlaceSearchEntry {
	place: GamePlace
	id: string
	names: string[]
}

export function indexPlaces(places: GamePlace[]): PlaceSearchEntry[] {
	return places.map((place) => ({
		place,
		id: String(place.placeId),
		names: [place.nickname, place.name]
			.filter(Boolean)
			.map((name) => ` ${normalize(name)}`),
	}))
}

export function matchPlaces(entries: PlaceSearchEntry[], text: string): GamePlace[] {
	const id = text.trim(),
		words = normalize(text).trim()
	if (!id) return entries.map((entry) => entry.place)
	const ranked: { place: GamePlace; score: number; order: number }[] = []
	entries.forEach((entry, order) => {
		let score = entry.id.startsWith(id) ? 0 : Infinity
		if (words) {
			for (const name of entry.names) {
				score = Math.min(score, nameScore(name, words))
			}
		}
		if (score < Infinity) ranked.push({ place: entry.place, score, order })
	})
	ranked.sort((a, b) => a.score - b.score || a.order - b.order)
	return ranked.map((match) => match.place)
}

function nameScore(name: string, query: string): number {
	if (name.startsWith(query, 1)) return 0
	if (name.includes(` ${query}`)) return 1
	if (name.includes(query)) return 2
	let first = -1,
		at = 0
	for (const char of query) {
		if (char === " ") continue
		at = name.indexOf(char, at + 1)
		if (at < 0) return Infinity
		if (first < 0) first = at
	}
	return 3 + (at - first) / name.length
}

function normalize(text: string): string {
	return text.toLowerCase().replace(/[^\p{L}\p{N}]+/gu, " ")
}

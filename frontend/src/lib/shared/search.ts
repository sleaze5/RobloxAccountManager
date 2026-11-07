export function searchTerms(query: string): string[] {
	return query.trim().toLowerCase().split(/\s+/).filter(Boolean)
}

export function matchesSearch(terms: string[], text: string): boolean {
	const lower = text.toLowerCase()
	return terms.every((term) => lower.includes(term))
}

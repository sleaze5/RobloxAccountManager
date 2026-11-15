interface LoadMoreTrigger {
	// pageKey changes after each loaded page, so that a sentinel that stays in
	// view loads the next page too.
	pageKey: unknown
	loadMore: () => void
}

// loadMoreWhenVisible calls loadMore when the sentinel comes within 160px of
// the bottom of its closest [data-load-more-root] ancestor.
export function loadMoreWhenVisible(
	sentinel: HTMLElement,
	trigger: LoadMoreTrigger,
): { update: (next: LoadMoreTrigger) => void; destroy: () => void } {
	let current = trigger
	const observer = new IntersectionObserver(
		(entries) => {
			if (entries.some((entry) => entry.isIntersecting)) current.loadMore()
		},
		{
			root: sentinel.closest("[data-load-more-root]"),
			rootMargin: "0px 0px 160px 0px",
		},
	)
	observer.observe(sentinel)
	return {
		update(next) {
			current = next
			observer.unobserve(sentinel)
			observer.observe(sentinel)
		},
		destroy: () => observer.disconnect(),
	}
}

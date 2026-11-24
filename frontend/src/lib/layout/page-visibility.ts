import { createContext } from "svelte"

// PageVisibility tells the components of a page whether the page shows. A page
// that the user leaves stays mounted and hidden, so that it keeps its state,
// and its components pause what only a shown page needs, such as shortcuts
// and refresh timers.
interface PageVisibility {
	readonly isShown: boolean
}

export const [getPageVisibility, setPageVisibility] = createContext<PageVisibility>()

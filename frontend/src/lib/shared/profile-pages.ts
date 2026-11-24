import { createContext } from "svelte"
import type { ProfileTab } from "./ProfileTabs.svelte"

// ProfilePages lists the pages of the open profile, so that its overview and
// each of its pages show the same tabs.
export interface ProfilePages {
	label: string
	tabs: ProfileTab[]
	// currentPage is the id of the open page, or "" while the overview is open.
	currentPage: string
	openPage: (page: string) => void
	openOverview: () => void
}

export const [getProfilePages, setProfilePages] = createContext<ProfilePages>()

// openPageLabel names the open page of a profile, or returns "" while the
// overview is open.
export function openPageLabel(pages: ProfilePages): string {
	return pages.tabs.find((tab) => tab.id === pages.currentPage)?.label ?? ""
}

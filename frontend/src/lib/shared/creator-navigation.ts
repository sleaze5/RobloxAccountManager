import { createContext } from "svelte"

// CreatorNavigation opens the creator of an item, a game, or a group on its
// own page: a group in Groups, and a user in Social.
interface CreatorNavigation {
	showGroup: (groupId: number) => void
	showUser: (user: { userId: number; username: string; displayName: string }) => void
}

export const [getCreatorNavigation, setCreatorNavigation] =
	createContext<CreatorNavigation>()

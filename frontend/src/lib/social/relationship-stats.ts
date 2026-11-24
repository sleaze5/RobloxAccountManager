import { SocialList, SocialProfileSection } from "../backend/bridge"
import type { ProfileStat } from "../shared/ProfileStats.svelte"
import type { SocialListKind } from "./social-store.svelte"

const compactNumbers = new Intl.NumberFormat("en-US", {
		notation: "compact",
		maximumFractionDigits: 1,
	}),
	exactNumbers = new Intl.NumberFormat()

// relationshipStats lists the friend, follower, and following counts of a
// user, each of which shows its list. A count that is missing and not loading
// is left out.
export function relationshipStats(
	counts: {
		friends: number | null
		followers: number | null
		following: number | null
	},
	isCountPending: (list: SocialListKind) => boolean,
	onShow?: (list: SocialListKind) => void,
): ProfileStat[] {
	const entries: {
		list: SocialListKind
		count: number | null
		label: string
		noun: string
	}[] = [
		{
			list: SocialList.SocialFriends,
			count: counts.friends,
			label: counts.friends === 1 ? "friend" : "friends",
			noun: counts.friends === 1 ? "friend" : "friends",
		},
		{
			list: SocialList.SocialFollowers,
			count: counts.followers,
			label: counts.followers === 1 ? "follower" : "followers",
			noun: counts.followers === 1 ? "follower" : "followers",
		},
		{
			list: SocialList.SocialFollowing,
			count: counts.following,
			label: "following",
			noun: counts.following === 1 ? "followed user" : "followed users",
		},
	]
	return entries.flatMap((entry): ProfileStat[] => {
		if (entry.count === null) {
			return isCountPending(entry.list)
				? [
						{
							id: entry.list,
							value: "",
							label: entry.label,
							description: entry.noun,
							isLoading: true,
						},
					]
				: []
		}
		return [
			{
				id: entry.list,
				value: compactNumbers.format(entry.count),
				label: entry.label,
				description: `${exactNumbers.format(entry.count)} ${entry.noun}`,
				onShow: onShow && (() => onShow(entry.list)),
			},
		]
	})
}

// relationshipCountSection names the profile section that reads the count of
// a relationship list.
export function relationshipCountSection(list: SocialListKind): SocialProfileSection {
	if (list === SocialList.SocialFollowers)
		return SocialProfileSection.SocialSectionFollowers
	if (list === SocialList.SocialFollowing)
		return SocialProfileSection.SocialSectionFollowing
	return SocialProfileSection.SocialSectionFriends
}

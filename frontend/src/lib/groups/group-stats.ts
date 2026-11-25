import type { GroupSummary } from "../backend/bridge"
import type { ProfileStat } from "../shared/ProfileStats.svelte"

const exactNumbers = new Intl.NumberFormat("en-US"),
	compactNumbers = new Intl.NumberFormat("en-US", {
		notation: "compact",
		maximumFractionDigits: 1,
	})

export function groupStats(group: GroupSummary): ProfileStat[] {
	const noun = group.memberCount === 1 ? "member" : "members"
	return [
		{
			id: "members",
			value: compactNumbers.format(group.memberCount),
			label: noun,
			description: `${exactNumbers.format(group.memberCount)} ${noun}`,
		},
	]
}

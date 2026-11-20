// LabeledBadge is a small pill with an icon and a label, such as a
// relationship or the role of a creator.
export interface LabeledBadge {
	label: string
	icon: string
	// tooltip explains the badge when its label alone does not.
	tooltip?: string
	isStrong: boolean
}

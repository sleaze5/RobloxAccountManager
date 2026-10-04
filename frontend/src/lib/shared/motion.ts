import { MotionPreference } from "../backend/bridge"

const systemReducedMotion = window.matchMedia?.("(prefers-reduced-motion: reduce)")
let preference = MotionPreference.MotionSystem

function apply(): void {
	const reduced =
		preference === MotionPreference.MotionReduced ||
		(preference === MotionPreference.MotionSystem &&
			(systemReducedMotion?.matches ?? false))
	document.documentElement.toggleAttribute("data-reduced-motion", reduced)
}

systemReducedMotion?.addEventListener("change", apply)
apply()

export function setMotionPreference(next: MotionPreference): void {
	preference = next
	apply()
}

export function reducedMotion(): boolean {
	return document.documentElement.hasAttribute("data-reduced-motion")
}

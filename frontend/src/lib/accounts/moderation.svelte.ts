import { ModerationKind } from "../backend/bridge"
import type { Moderation } from "../backend/bridge"
import { formatClock } from "../shared/duration"

const hourMs = 3_600_000,
	dayMs = 24 * hourMs

export interface ModerationStatus {
	label: string
	detail: string
	tone: "danger" | "warning"
}

export function moderationLabel(moderation: Moderation | null): string {
	switch (moderation?.kind) {
		case ModerationKind.ModerationWarned:
			return "Warned"
		case ModerationKind.ModerationBanned:
			return "Banned"
		case ModerationKind.ModerationTerminated:
			return "Terminated"
		case ModerationKind.ModerationLocked:
			return "Locked"
		default:
			return "Restricted"
	}
}

// A locked account has no punishment to accept. Roblox unlocks it after a
// human verification on its website.
export function isLocked(moderation: Moderation | null): boolean {
	return moderation?.kind === ModerationKind.ModerationLocked
}

// Roblox has no reactivation request for a lock or an unknown restriction, so
// the account resolves it on the Roblox website.
export function resolvesInBrowser(moderation: Moderation | null): boolean {
	return !moderation || isLocked(moderation)
}

// Only a warning and a ban can be accepted through the reactivation request.
export function hasReactivation(moderation: Moderation | null): boolean {
	return (
		moderation?.kind === ModerationKind.ModerationWarned ||
		moderation?.kind === ModerationKind.ModerationBanned
	)
}

// Roblox lets a warned account, or a banned account after its ban ends,
// accept the moderation and reactivate. It never reactivates a termination.
export function canReactivate(moderation: Moderation | null, nowMs: number): boolean {
	if (moderation?.kind === ModerationKind.ModerationWarned) return true
	return (
		moderation?.kind === ModerationKind.ModerationBanned &&
		!!moderation.endsAtMs &&
		nowMs >= moderation.endsAtMs
	)
}

export function banLength(moderation: Moderation): string {
	const lengthMs = (moderation.endsAtMs ?? 0) - moderation.beginsAtMs
	if (lengthMs >= dayMs) return `${Math.round(lengthMs / dayMs)}d`
	if (lengthMs >= hourMs) return `${Math.round(lengthMs / hourMs)}h`
	return `${Math.max(1, Math.round(lengthMs / 60_000))}m`
}

export function banRemainingSeconds(moderation: Moderation, nowMs: number): number {
	return Math.max(0, Math.ceil(((moderation.endsAtMs ?? 0) - nowMs) / 1000))
}

export function moderationStatus(
	moderation: Moderation | null,
	nowMs: number,
): ModerationStatus {
	const label = moderationLabel(moderation)
	if (moderation?.kind === ModerationKind.ModerationWarned) {
		return { label, detail: "", tone: "warning" }
	}
	if (resolvesInBrowser(moderation)) {
		return {
			label: `${label}:`,
			detail: isLocked(moderation) ? "verify you're human" : "resolve in browser",
			tone: "warning",
		}
	}
	if (moderation?.kind !== ModerationKind.ModerationBanned || !moderation.endsAtMs) {
		return { label, detail: "", tone: "danger" }
	}
	const remaining = banRemainingSeconds(moderation, nowMs)
	return remaining > 0
		? {
				label: `${label}:`,
				detail: `${formatClock(remaining, "days")} (${banLength(moderation)})`,
				tone: "danger",
			}
		: { label: "Ban ended", detail: `(${banLength(moderation)})`, tone: "warning" }
}

// banEndsAtMs is the end of a running ban countdown, or 0 without one.
export function banEndsAtMs(moderation: Moderation | null): number {
	return moderation?.kind === ModerationKind.ModerationBanned
		? (moderation.endsAtMs ?? 0)
		: 0
}

import { formatTimestamp } from "../shared/timestamp"

// Roblox's web client starts a new timestamp group after this gap.
export const chatTimestampBreakMs = 30_000

function isToday(value: number): boolean {
	return new Date(value).toDateString() === new Date().toDateString()
}

export function chatListTime(value: number): string {
	return value ? formatTimestamp(value, isToday(value) ? "h:mm A" : "MMM D") : ""
}

export function chatMessageTime(value: number): string {
	return formatTimestamp(value, isToday(value) ? "h:mm A" : "MMM D, h:mm A")
}

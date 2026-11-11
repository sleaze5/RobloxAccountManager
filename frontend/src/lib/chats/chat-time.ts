export const chatTimestampBreakMs = 30_000

const time = new Intl.DateTimeFormat("en-US", { hour: "numeric", minute: "2-digit" }),
	day = new Intl.DateTimeFormat("en-US", { month: "short", day: "numeric" }),
	dayTime = new Intl.DateTimeFormat("en-US", {
		month: "short",
		day: "numeric",
		hour: "numeric",
		minute: "2-digit",
	})

function isToday(timestampMs: number): boolean {
	return new Date(timestampMs).toDateString() === new Date().toDateString()
}

export function chatListTime(timestampMs: number): string {
	return timestampMs ? (isToday(timestampMs) ? time : day).format(timestampMs) : ""
}

export function chatMessageTime(timestampMs: number): string {
	return (isToday(timestampMs) ? time : dayTime).format(timestampMs)
}

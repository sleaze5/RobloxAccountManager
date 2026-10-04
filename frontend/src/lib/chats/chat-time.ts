export const chatTimestampBreakMs = 30_000

const time = new Intl.DateTimeFormat("en-US", { hour: "numeric", minute: "2-digit" }),
	day = new Intl.DateTimeFormat("en-US", { month: "short", day: "numeric" }),
	dayTime = new Intl.DateTimeFormat("en-US", {
		month: "short",
		day: "numeric",
		hour: "numeric",
		minute: "2-digit",
	})

function isToday(value: number): boolean {
	return new Date(value).toDateString() === new Date().toDateString()
}

export function chatListTime(value: number): string {
	return value ? (isToday(value) ? time : day).format(value) : ""
}

export function chatMessageTime(value: number): string {
	return (isToday(value) ? time : dayTime).format(value)
}

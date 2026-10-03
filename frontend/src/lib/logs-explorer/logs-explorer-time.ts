export function formatElapsed(milliseconds: number): string {
	const totalSeconds = Math.max(0, Math.floor(milliseconds / 1000)),
		hours = Math.floor(totalSeconds / 3600),
		minutes = Math.floor((totalSeconds % 3600) / 60),
		seconds = totalSeconds % 60
	const parts = [minutes, seconds]
	if (hours > 0) parts.unshift(hours)
	return parts.map((part) => String(part).padStart(2, "0")).join(":")
}

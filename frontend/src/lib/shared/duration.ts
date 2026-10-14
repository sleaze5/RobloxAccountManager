const durationUnits = ["ns", "µs", "ms", "s"] as const

export function formatCompactDuration(milliseconds: number): string {
	if (!Number.isFinite(milliseconds) || milliseconds <= 0) return "0 ns"
	let value = milliseconds * 1_000_000,
		unit = 0
	while (Number(value.toFixed(1)) >= 1000 && unit < durationUnits.length - 1) {
		value /= 1000
		unit++
	}
	return `${Number(value.toFixed(1))} ${durationUnits[unit]}`
}

export function formatClock(
	seconds: number,
	largestUnit: "hours" | "days" = "hours",
): string {
	const total = Math.max(0, Math.floor(seconds)),
		days = largestUnit === "days" ? Math.floor(total / 86_400) : 0,
		hours = Math.floor((total - days * 86_400) / 3600),
		minutes = Math.floor((total % 3600) / 60),
		rest = String(total % 60).padStart(2, "0"),
		clock =
			days || hours
				? `${hours}:${String(minutes).padStart(2, "0")}:${rest}`
				: `${minutes}:${rest}`
	return days ? `${days}d ${clock}` : clock
}

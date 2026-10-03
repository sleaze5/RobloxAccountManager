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

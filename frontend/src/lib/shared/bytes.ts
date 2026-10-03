const byteUnits = [
	"B",
	"kB",
	"MB",
	"GB",
	"TB",
	"PB",
	"EB",
	"ZB",
	"YB",
	"RB",
	"QB",
] as const

export function formatCompactBytes(bytes: number): string {
	if (!Number.isFinite(bytes) || bytes <= 0) return "0 B"
	let value = Math.round(bytes),
		unit = 0
	while (Number(value.toFixed(1)) >= 1000 && unit < byteUnits.length - 1) {
		value /= 1000
		unit++
	}
	return `${unit === 0 ? value : Number(value.toFixed(1))} ${byteUnits[unit]}`
}

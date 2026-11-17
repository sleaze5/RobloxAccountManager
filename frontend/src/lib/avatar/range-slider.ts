// roundedToStep rounds to the step, because range inputs add floating point
// noise such as 0.9500000001.
export function roundedToStep(rawValue: number, step: number): number {
	return Number((Math.round(rawValue / step) * step).toFixed(4))
}

// rangeShareOf is where the value sits on the track, from 0 at the minimum to
// 1 at the maximum.
export function rangeShareOf(value: number, min: number, max: number): number {
	return max > min ? Math.min(Math.max((value - min) / (max - min), 0), 1) : 0
}

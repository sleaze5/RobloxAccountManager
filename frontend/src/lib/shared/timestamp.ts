import { TimestampSegmentKind as Kind } from "../backend/bridge"
import type { TimestampFormat } from "../backend/bridge"

export const defaultTimestampFormats = {
	shown: "$MMM $D $YYYY, $h:$mm:$ss $A",
	hover: "$dddd, $MMMM $DD $YYYY, $hh:$mm:$ss $A ($relative)",
} as const

const monthNames = [
		"January",
		"February",
		"March",
		"April",
		"May",
		"June",
		"July",
		"August",
		"September",
		"October",
		"November",
		"December",
	],
	weekdayNames = [
		"Sunday",
		"Monday",
		"Tuesday",
		"Wednesday",
		"Thursday",
		"Friday",
		"Saturday",
	]

function pad(value: number, length = 2): string {
	return String(value).padStart(length, "0")
}

function hour12(date: Date): number {
	return date.getHours() % 12 || 12
}

const dateParts: Record<
	Exclude<Kind, Kind.$zero | Kind.KindText | Kind.KindRelative>,
	(date: Date) => string
> = {
	[Kind.KindYear]: (date) => pad(date.getFullYear(), 4),
	[Kind.KindYearShort]: (date) => pad(date.getFullYear() % 100),
	[Kind.KindMonthName]: (date) => monthNames[date.getMonth()],
	[Kind.KindMonthNameShort]: (date) => monthNames[date.getMonth()].slice(0, 3),
	[Kind.KindMonthPadded]: (date) => pad(date.getMonth() + 1),
	[Kind.KindMonth]: (date) => String(date.getMonth() + 1),
	[Kind.KindDayPadded]: (date) => pad(date.getDate()),
	[Kind.KindDay]: (date) => String(date.getDate()),
	[Kind.KindWeekday]: (date) => weekdayNames[date.getDay()],
	[Kind.KindWeekdayShort]: (date) => weekdayNames[date.getDay()].slice(0, 3),
	[Kind.KindHour24Padded]: (date) => pad(date.getHours()),
	[Kind.KindHour24]: (date) => String(date.getHours()),
	[Kind.KindHour12Padded]: (date) => pad(hour12(date)),
	[Kind.KindHour12]: (date) => String(hour12(date)),
	[Kind.KindMinutePadded]: (date) => pad(date.getMinutes()),
	[Kind.KindMinute]: (date) => String(date.getMinutes()),
	[Kind.KindSecondPadded]: (date) => pad(date.getSeconds()),
	[Kind.KindSecond]: (date) => String(date.getSeconds()),
	[Kind.KindMillisecond]: (date) => pad(date.getMilliseconds(), 3),
	[Kind.KindMeridiem]: (date) => (date.getHours() < 12 ? "AM" : "PM"),
	[Kind.KindMeridiemLower]: (date) => (date.getHours() < 12 ? "am" : "pm"),
}

// Renders a format that the backend parsed. Returns an empty string until the format loads.
export function formatTimestamp(
	value: number | Date,
	format: TimestampFormat | null,
	now = Date.now(),
): string {
	const date = new Date(value)
	if (!format || Number.isNaN(date.getTime())) {
		return ""
	}
	return (format.segments ?? [])
		.map((segment) => {
			if (segment.kind === Kind.KindText) return segment.text ?? ""
			if (segment.kind === Kind.KindRelative)
				return relativeTime(date.getTime(), now)
			if (segment.kind === Kind.$zero) return ""
			return dateParts[segment.kind](date)
		})
		.join("")
}

const dayMs = 86_400_000

// Rounds to the largest fitting unit, such as "a few seconds", "a minute", or "3 hours".
function relativeDuration(milliseconds: number): string {
	if (Math.round(milliseconds / 1_000) < 45) {
		return "a few seconds"
	}
	const days = milliseconds / dayMs,
		units = [
			{
				value: Math.round(milliseconds / 60_000),
				limit: 45,
				one: "a minute",
				many: "minutes",
			},
			{
				value: Math.round(milliseconds / 3_600_000),
				limit: 22,
				one: "an hour",
				many: "hours",
			},
			{ value: Math.round(days), limit: 26, one: "a day", many: "days" },
			{
				value: Math.round(days / 30.436875),
				limit: 11,
				one: "a month",
				many: "months",
			},
		]
	for (const unit of units) {
		if (unit.value < unit.limit) {
			return unit.value <= 1 ? unit.one : `${unit.value} ${unit.many}`
		}
	}
	const years = Math.round(days / 365.2425)
	return years <= 1 ? "a year" : `${years} years`
}

export function relativeTime(value: number, now = Date.now(), suffix = true): string {
	const difference = value - now,
		duration = relativeDuration(Math.abs(difference))
	if (!suffix) {
		return duration
	}
	return difference > 0 ? `in ${duration}` : `${duration} ago`
}

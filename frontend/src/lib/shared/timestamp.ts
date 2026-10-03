import moment from "moment"

export const timestampFormats = {
	shown: "MMM D YYYY, h:mm:ss A",
	tooltip: "dddd, MMMM DD YYYY, hh:mm:ss A ([relative])",
} as const

const relativePlaceholder = "\uE000"

export function formatTimestamp(
	value: number | Date,
	format: string,
	now = Date.now(),
): string {
	const date = moment(value)
	if (!date.isValid()) {
		return ""
	}

	if (!format.includes("[relative]")) {
		return date.format(format)
	}

	const formatted = date.format(
		format.replaceAll("[relative]", `[${relativePlaceholder}]`),
	)
	return formatted.replaceAll(relativePlaceholder, date.from(moment(now)))
}

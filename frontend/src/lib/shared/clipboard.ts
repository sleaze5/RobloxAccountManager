import type { NotificationCenter } from "../notifications/notification-center.svelte"

export async function copyText(
	notifications: NotificationCenter,
	text: string,
	message: string,
): Promise<boolean> {
	try {
		await navigator.clipboard.writeText(text)
	} catch {
		return false
	}
	notifyCopied(notifications, message)
	return true
}

export function notifyCopied(notifications: NotificationCenter, message: string): void {
	notifications.show({
		id: "clipboard-copy",
		title: "Copied",
		tone: "success",
		message,
	})
}

// copyFailureMessage names what failed to copy. label is a sentence-case name,
// such as "Job ID", and starts lowercase inside the sentence.
export function copyFailureMessage(label: string): string {
	return `Could not copy the ${label.charAt(0).toLowerCase()}${label.slice(1)}. Try again.`
}

// copyLines copies one value per line, such as the user IDs of the selected
// users. label names one value in sentence case, such as "User ID", and noun
// names the items in the plural, such as "users".
export async function copyLines(
	notifications: NotificationCenter,
	values: string[],
	label: string,
	noun: string,
): Promise<boolean> {
	return copyText(
		notifications,
		values.join("\n"),
		values.length === 1
			? `${label} copied to clipboard.`
			: `${label} copied for ${values.length} ${noun}, one per line.`,
	)
}

// copyValue copies one value that label names in sentence case, such as
// "Job ID", and reports a failure as an error notification.
export async function copyValue(
	notifications: NotificationCenter,
	value: string,
	label: string,
): Promise<boolean> {
	if (await copyText(notifications, value, `${label} copied to clipboard.`)) {
		return true
	}
	notifyCopyFailure(notifications, label)
	return false
}

export function notifyCopyFailure(
	notifications: NotificationCenter,
	label: string,
): void {
	notifications.show({
		id: "clipboard-copy-failure",
		title: "Not copied",
		tone: "error",
		message: copyFailureMessage(label),
	})
}

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

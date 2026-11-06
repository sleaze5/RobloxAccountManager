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
	notifications.show({
		id: "clipboard-copy",
		title: "Copied",
		tone: "success",
		message,
	})
	return true
}

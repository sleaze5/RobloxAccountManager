import type { NotificationCenter } from "../notifications/notification-center.svelte"

export async function copyExportFolder(
	directory: string,
	notifications: NotificationCenter,
): Promise<void> {
	try {
		await navigator.clipboard.writeText(directory)
		notifications.show({
			id: "clipboard-copy",
			title: "Copied",
			tone: "success",
			message: "Export folder path copied to clipboard.",
		})
	} catch {
		notifications.show({
			id: "clipboard-copy",
			title: "Path not copied",
			tone: "error",
			message: "The export folder path could not be copied. Try again.",
		})
	}
}

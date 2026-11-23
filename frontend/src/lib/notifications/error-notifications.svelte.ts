import { getNotificationCenter } from "./notification-center.svelte"

// notifyErrors shows each message that readMessage returns as an error
// snackbar while the calling component is mounted. The message is the
// notification ID, so a repeated failure, or one that two components report,
// shows one snackbar.
export function notifyErrors(title: string, readMessage: () => string): void {
	const notifications = getNotificationCenter()
	$effect(() => {
		const message = readMessage()
		if (!message) return
		notifications.show({
			id: `error:${message}`,
			message,
			title,
			tone: "error",
		})
	})
}

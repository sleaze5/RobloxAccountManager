import { errorMessage } from "../shared/errors"
import {
	getNotificationCenter,
	type NotificationCenter,
} from "./notification-center.svelte"

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

// notifyLoadFailure reports a read that the app needs and cannot repeat on its
// own, such as the build details at startup.
export function notifyLoadFailure(
	notifications: NotificationCenter,
	id: string,
	title: string,
	error: unknown,
): void {
	notifications.show({
		id,
		title,
		tone: "error",
		message: errorMessage(error, "Restart the app and try again."),
	})
}

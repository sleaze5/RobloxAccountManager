import { copyText } from "../shared/clipboard"
import { accountBackend } from "../backend/bridge"
import type { NotificationCenter } from "../notifications/notification-center.svelte"
import { errorMessage } from "../shared/errors"

export async function chooseExportFile(
	notifications: NotificationCenter,
): Promise<string> {
	try {
		return await accountBackend.ChooseVaultExport()
	} catch (cause) {
		notifications.show({
			id: "vault-import",
			title: "File not opened",
			tone: "error",
			message: errorMessage(
				cause,
				"The open dialog could not be opened. Try again.",
			),
		})
		return ""
	}
}

export async function copyPath(
	path: string,
	notifications: NotificationCenter,
): Promise<void> {
	if (!(await copyText(notifications, path, "File path copied to clipboard."))) {
		notifications.show({
			id: "clipboard-copy",
			title: "Path not copied",
			tone: "error",
			message: "The file path could not be copied. Try again.",
		})
	}
}

export function fileName(path: string): string {
	return path.split(/[\\/]/).at(-1) ?? path
}

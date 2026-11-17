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
			message: errorMessage(cause, "Could not open the file dialog. Try again."),
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
			title: "Not copied",
			tone: "error",
			message: "Could not copy the file path. Try again.",
		})
	}
}

export function fileName(path: string): string {
	return path.split(/[\\/]/).at(-1) ?? path
}

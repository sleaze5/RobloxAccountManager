import type { RobloxClientState } from "../backend/bridge"

export type LinuxClientOption = {
	value: string
	label: string
	description: string
}

export const automaticLinuxClient: LinuxClientOption = {
	value: "",
	label: "Automatic",
	description:
		"Use the installed client, or the desktop default when both are installed.",
}

export function linuxClientOptions(state: RobloxClientState): LinuxClientOption[] {
	const options = [automaticLinuxClient]
	for (const client of state.clients ?? []) {
		if (!client.installed && client.id !== state.selected) continue
		options.push({
			value: client.id,
			label: client.installed ? client.name : `${client.name} (not installed)`,
			description: "",
		})
	}
	if (state.selected && !options.some((option) => option.value === state.selected)) {
		options.push({
			value: state.selected,
			label: "Selected client (not installed)",
			description: "",
		})
	}
	return options
}

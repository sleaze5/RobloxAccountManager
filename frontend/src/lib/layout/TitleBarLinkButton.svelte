<script lang="ts">
	import Icon from "../shared/Icon.svelte"
	import { copyText } from "../shared/clipboard"
	import { LongPress } from "../shared/long-press"
	import type { NotificationCenter } from "../notifications/notification-center.svelte"
	import { Browser } from "@wailsio/runtime"

	let {
		url,
		label,
		urlName,
		icon,
		notifications,
	}: {
		url: string
		label: string
		urlName: string
		icon: string
		notifications: NotificationCenter
	} = $props()

	// Holding the button is an intentionally hidden shortcut that copies the
	// URL instead of opening it. It has no visible hint on purpose.
	const longPress = new LongPress(() => void copyURL())

	async function copyURL(): Promise<void> {
		if (!(await copyText(notifications, url, `${urlName} copied to clipboard.`))) {
			notifications.show({
				id: "clipboard-copy",
				title: "Not copied",
				tone: "error",
				message: `Could not copy the ${urlName}. Try again.`,
			})
		}
	}
</script>

<button
	class="button-ghost self-center"
	type="button"
	aria-label={label}
	data-tooltip={label}
	onpointerdown={longPress.beginPress}
	onpointerup={longPress.endPress}
	onpointerleave={longPress.endPress}
	onpointercancel={longPress.endPress}
	onkeydown={longPress.forgetLongPress}
	onclick={() => {
		if (longPress.consumeLongPress()) return
		void Browser.OpenURL(url)
	}}>
	<Icon svg={icon} size={18} aria-hidden="true" />
</button>

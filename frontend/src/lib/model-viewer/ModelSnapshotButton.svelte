<script lang="ts">
	import cameraIcon from "@phosphor-icons/core/regular/camera.svg?raw"
	import copyIcon from "@phosphor-icons/core/regular/copy.svg?raw"
	import downloadSimpleIcon from "@phosphor-icons/core/regular/download-simple.svg?raw"
	import { accountBackend } from "../backend/bridge"
	import { getNotificationCenter } from "../notifications/notification-center.svelte"
	import AnchoredActionMenu from "../shared/AnchoredActionMenu.svelte"
	import type { MenuAction } from "../shared/AnchoredActionMenu.svelte"
	import { notifyCopied } from "../shared/clipboard"
	import { errorMessage } from "../shared/errors"
	import Icon from "../shared/Icon.svelte"

	// ModelSnapshotButton takes a snapshot of a 3D view at the moment it is
	// clicked, while an animation plays too, and then offers to copy or save
	// it. label names the subject, such as "@builderman's avatar", and names
	// the saved file. captureFrame returns the frame shown now as a PNG data
	// URL.
	let { label, captureFrame }: { label: string; captureFrame: () => string } =
		$props()

	let button = $state<HTMLButtonElement>(),
		snapshotBase64 = $state(""),
		isSaving = $state(false)

	const notifications = getNotificationCenter(),
		menuId = $props.id(),
		notificationId = "model-snapshot",
		pngDataUrlPrefix = "data:image/png;base64,"

	const menuActions = $derived<MenuAction[]>(
		snapshotBase64
			? [
					{
						label: "Copy image",
						icon: copyIcon,
						onSelect: () => void copySnapshot(takeMenuSnapshot()),
					},
					{
						label: "Save image",
						icon: downloadSimpleIcon,
						onSelect: () => void saveSnapshot(takeMenuSnapshot()),
					},
				]
			: [],
	)

	function toggleMenu(): void {
		if (snapshotBase64) {
			snapshotBase64 = ""
			return
		}
		const frameUrl = captureFrame()
		if (!frameUrl.startsWith(pngDataUrlPrefix)) {
			notifications.show({
				id: notificationId,
				tone: "error",
				title: "Snapshot not taken",
				message: "Could not read the 3D view. Try again.",
			})
			return
		}
		snapshotBase64 = frameUrl.slice(pngDataUrlPrefix.length)
	}

	// takeMenuSnapshot closes the menu and returns the snapshot that it offered.
	function takeMenuSnapshot(): string {
		const offeredSnapshotBase64 = snapshotBase64
		snapshotBase64 = ""
		return offeredSnapshotBase64
	}

	async function copySnapshot(pngBase64: string): Promise<void> {
		try {
			const pngBytes = Uint8Array.from(atob(pngBase64), (character) =>
				character.charCodeAt(0),
			)
			await navigator.clipboard.write([
				new ClipboardItem({
					"image/png": new Blob([pngBytes], { type: "image/png" }),
				}),
			])
		} catch {
			notifications.show({
				id: notificationId,
				tone: "error",
				title: "Snapshot not copied",
				message: "Could not copy the image. Try again.",
			})
			return
		}
		notifyCopied(notifications, "Image copied to clipboard.")
	}

	async function saveSnapshot(pngBase64: string): Promise<void> {
		if (isSaving) return
		isSaving = true
		try {
			const savedPath = await accountBackend.SaveModelSnapshot(label, pngBase64)
			if (!savedPath) return
			notifications.show({
				id: notificationId,
				tone: "success",
				title: "Image saved",
				message: savedPath,
			})
		} catch (error) {
			notifications.show({
				id: notificationId,
				tone: "error",
				title: "Image not saved",
				message: errorMessage(error, "Could not save the image. Try again."),
			})
		} finally {
			isSaving = false
		}
	}
</script>

<button
	bind:this={button}
	class="absolute right-2 bottom-2 z-10 button-icon bg-overlay"
	type="button"
	aria-label="Take snapshot"
	aria-haspopup="menu"
	aria-expanded={!!snapshotBase64}
	aria-controls={snapshotBase64 ? menuId : undefined}
	aria-busy={isSaving}
	data-open={snapshotBase64 ? true : undefined}
	data-tooltip="Take snapshot"
	disabled={isSaving}
	onclick={toggleMenu}>
	<Icon svg={cameraIcon} size={16} aria-hidden="true" />
</button>
{#if button && menuActions.length}
	<AnchoredActionMenu
		id={menuId}
		label="Snapshot actions"
		anchor={button}
		actions={menuActions}
		onClose={() => (snapshotBase64 = "")} />
{/if}

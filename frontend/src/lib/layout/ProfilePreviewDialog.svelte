<script lang="ts">
	import arrowsOutSimpleIcon from "@phosphor-icons/core/regular/arrows-out-simple.svg?raw"
	import atIcon from "@phosphor-icons/core/regular/at.svg?raw"
	import globeHemisphereWestIcon from "@phosphor-icons/core/regular/globe-hemisphere-west.svg?raw"
	import hashIcon from "@phosphor-icons/core/regular/hash.svg?raw"
	import linkIcon from "@phosphor-icons/core/regular/link.svg?raw"
	import xIcon from "@phosphor-icons/core/regular/x.svg?raw"
	import GamePreview from "../games/GamePreview.svelte"
	import GroupPreview from "../groups/GroupPreview.svelte"
	import type { SocialUser } from "../backend/bridge"
	import { getNotificationCenter } from "../notifications/notification-center.svelte"
	import type { MenuAction } from "../shared/AnchoredActionMenu.svelte"
	import { copyValue } from "../shared/clipboard"
	import ConfirmationDialog from "../shared/ConfirmationDialog.svelte"
	import Dialog from "../shared/Dialog.svelte"
	import Icon from "../shared/Icon.svelte"
	import MoreActionsMenu from "../shared/MoreActionsMenu.svelte"
	import type { ProfilePreviewSubject } from "../shared/profile-preview"
	import {
		getRobloxPageOpener,
		openInBrowserUnavailableReason,
		robloxGamePassUrl,
		robloxGameUrl,
		robloxGroupUrl,
		robloxUserProfileUrl,
	} from "../shared/roblox-pages"
	import type {
		ProfilePreviewConfirmation,
		ProfilePreviewFrame,
	} from "../shared/ProfilePreviewLayout.svelte"
	import type { RelationshipActionId } from "../social/relationship"
	import UserPreview from "../social/UserPreview.svelte"
	import { untrack } from "svelte"

	// ProfilePreviewDialog shows the preview of the last subject that was
	// opened, above the page that opened it. A preview that opens from another
	// preview goes back to it.
	let {
		subjects,
		accountId,
		onBack,
		onClose,
		onOpenFull,
		onRelationshipAction,
		onMembershipChanged,
	}: {
		subjects: ProfilePreviewSubject[]
		accountId: number
		onBack: () => void
		onClose: () => void
		onOpenFull: (subject: ProfilePreviewSubject) => void
		// onRelationshipAction runs a relationship action on the user, and
		// returns whether it succeeded.
		onRelationshipAction: (
			person: SocialUser,
			actionId: RelationshipActionId,
		) => Promise<boolean>
		onMembershipChanged: (groupId: number) => void
	} = $props()

	let confirmation = $state<ProfilePreviewConfirmation | null>(null),
		isConfirmationWorking = $state(false)

	// A question belongs to the preview that asked it.
	$effect(() => {
		void subjects.at(-1)
		untrack(() => (confirmation = null))
	})

	async function runConfirmation(): Promise<void> {
		if (!confirmation || isConfirmationWorking) return
		isConfirmationWorking = true
		try {
			if (await confirmation.run()) confirmation = null
		} finally {
			isConfirmationWorking = false
		}
	}

	const robloxPages = getRobloxPageOpener(),
		notifications = getNotificationCenter()

	// RobloxPage is the page of the previewed subject on the Roblox website,
	// with the ID that names it.
	interface RobloxPage {
		pageUrl: string
		linkLabel: string
		id: number
		idLabel: string
	}

	function robloxPageOf(previewed: ProfilePreviewSubject): RobloxPage {
		if (previewed.kind === "User")
			return {
				pageUrl: robloxUserProfileUrl(previewed.userId),
				linkLabel: "Profile link",
				id: previewed.userId,
				idLabel: "User ID",
			}
		if (previewed.kind === "Group")
			return {
				pageUrl: robloxGroupUrl(previewed.groupId),
				linkLabel: "Group link",
				id: previewed.groupId,
				idLabel: "Group ID",
			}
		if (previewed.gamePassId)
			return {
				pageUrl: robloxGamePassUrl(previewed.gamePassId),
				linkLabel: "Game pass link",
				id: previewed.gamePassId,
				idLabel: "Game pass ID",
			}
		return {
			pageUrl: robloxGameUrl(previewed.placeId),
			linkLabel: "Game link",
			id: previewed.placeId,
			idLabel: "Place ID",
		}
	}

	function copyAction(
		label: string,
		icon: string,
		copiedValue: string,
		detail = "",
	): MenuAction {
		return {
			label,
			icon,
			section: "Copy",
			detail,
			accessibleLabel: `Copy ${label.charAt(0).toLowerCase()}${label.slice(1)}`,
			onSelect: () => void copyValue(notifications, copiedValue, label),
		}
	}

	function moreActions(previewed: ProfilePreviewSubject): MenuAction[] {
		const page = robloxPageOf(previewed)
		return [
			{
				label: "Open in browser",
				icon: globeHemisphereWestIcon,
				disabled: !robloxPages.canOpen,
				tooltip: robloxPages.canOpen
					? undefined
					: openInBrowserUnavailableReason,
				onSelect: () => robloxPages.open(page.pageUrl),
			},
			...(previewed.kind === "User" && previewed.username
				? [
						copyAction(
							"Username",
							atIcon,
							previewed.username,
							`@${previewed.username}`,
						),
					]
				: []),
			copyAction(page.idLabel, hashIcon, String(page.id), String(page.id)),
			copyAction(page.linkLabel, linkIcon, page.pageUrl),
		]
	}

	const previewId = $props.id()
	const titleId = `${previewId}-title`,
		pageNameByKind: Record<ProfilePreviewSubject["kind"], string> = {
			User: "Social",
			Group: "Groups",
			Place: "Games",
		},
		actionNounByKind: Record<ProfilePreviewSubject["kind"], string> = {
			User: "User",
			Group: "Group",
			Place: "Game",
		}

	const subject = $derived(subjects.at(-1)),
		frame = $derived<ProfilePreviewFrame>({
			titleId,
			actions,
			onBack: subjects.length > 1 ? onBack : null,
			confirm: (requested) => (confirmation = requested),
		})
</script>

<!-- The preview opens above the page and its dialogs, such as the item
details, so it handles Escape before them. A dialog above the preview, such
as a confirmation, and an open menu handle their own Escape. -->
<svelte:window
	onkeydowncapture={(event) => {
		if (event.key !== "Escape") return
		if (event.target instanceof Element && event.target.closest('[role="menu"]'))
			return
		const topDialog = Array.from(
			document.querySelectorAll('[aria-modal="true"]'),
		).at(-1)
		if (!topDialog?.hasAttribute("data-profile-preview")) return
		event.preventDefault()
		event.stopImmediatePropagation()
		onClose()
	}} />

{#snippet actions()}
	{#if subject}
		<button
			class="button"
			type="button"
			data-open-full-profile
			onclick={() => onOpenFull(subject)}>
			<Icon svg={arrowsOutSimpleIcon} size={16} aria-hidden="true" />
			Open in {pageNameByKind[subject.kind]}
		</button>
		<MoreActionsMenu
			label={`${actionNounByKind[subject.kind]} actions`}
			actions={moreActions(subject)} />
	{/if}
	<button
		class="ml-1 button-close"
		type="button"
		aria-label="Close dialog"
		onclick={onClose}>
		<Icon svg={xIcon} size={14} aria-hidden="true" />
	</button>
{/snippet}

{#if subject}
	<Dialog
		class="max-w-2xl text-left"
		aria-labelledby={titleId}
		data-profile-preview
		onBackdropClick={onClose}>
		{#key subject}
			{#if subject.kind === "User"}
				<UserPreview
					{accountId}
					user={subject}
					{frame}
					{onRelationshipAction} />
			{:else if subject.kind === "Group"}
				<GroupPreview
					{accountId}
					groupId={subject.groupId}
					{frame}
					{onMembershipChanged} />
			{:else}
				<GamePreview placeId={subject.placeId} {frame} />
			{/if}
		{/key}
	</Dialog>
{/if}

{#if subject && confirmation}
	<ConfirmationDialog
		title={confirmation.title}
		description={confirmation.description}
		confirmLabel={confirmation.confirmLabel}
		progressLabel={confirmation.progressLabel}
		tone={confirmation.tone}
		isWorking={isConfirmationWorking}
		onConfirm={() => void runConfirmation()}
		onCancel={() => (confirmation = null)} />
{/if}

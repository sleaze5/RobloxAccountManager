<script lang="ts">
	import xIcon from "@phosphor-icons/core/regular/x.svg?raw"
	import Icon from "../shared/Icon.svelte"
	import Dialog from "../shared/Dialog.svelte"
	import ProfileImage from "../shared/ProfileImage.svelte"
	import type { Account } from "../accounts/account-model"
	import type { SocialUser } from "../backend/bridge"
	import {
		relationshipActionsById,
		relationshipButtonClass,
	} from "../social/relationship"
	import type { RelationshipActionId } from "../social/relationship"
	import { personName } from "../social/social-store.svelte"
	import type { SocialStore } from "../social/social-store.svelte"

	let {
		store,
		account,
		person,
		actionId,
		onClose,
	}: {
		store: SocialStore
		account: Account
		person: SocialUser
		actionId: RelationshipActionId
		onClose: () => void
	} = $props()

	const action = $derived(relationshipActionsById[actionId]),
		pending = $derived(store.isActionPending(person.userId, actionId))

	function close(): void {
		if (pending) return
		store.relationshipError = ""
		onClose()
	}

	async function confirm(): Promise<void> {
		if (await store.runRelationshipAction(person, actionId)) onClose()
	}
</script>

<Dialog
	class="max-w-sm text-left"
	role="alertdialog"
	aria-labelledby="relationship-action-title"
	aria-describedby="relationship-action-description">
	<div class="dialog-heading mb-2">
		<div>
			<h2 id="relationship-action-title" class="wrap-anywhere">
				{action.question(personName(person))}
			</h2>
			<p id="relationship-action-description">{action.consequence}</p>
		</div>
		<button
			type="button"
			aria-label="Close dialog"
			disabled={pending}
			onclick={close}>
			<Icon svg={xIcon} size={14} aria-hidden="true" />
		</button>
	</div>
	<div
		class="mt-1 flex items-center gap-2 rounded-md border border-line bg-fill px-2 py-1.5 text-xs">
		<span
			class="relative grid size-7 shrink-0 place-items-center overflow-hidden rounded-md border border-line-strong bg-raised"
			aria-hidden="true"><ProfileImage url={person.imageUrl} /></span>
		<span class="flex min-w-0 flex-1 flex-col leading-snug">
			<span class="truncate font-medium text-fg">{personName(person)}</span>
			{#if person.username}<span class="truncate text-2xs text-fg-faint"
					>@{person.username}</span
				>{/if}
		</span>
		<span class="shrink-0 text-2xs text-fg-muted">as @{account.username}</span>
	</div>
	{#if store.relationshipError}
		<div class="mt-3 notice-error" role="alert">{store.relationshipError}</div>
	{/if}
	<div class="dialog-actions">
		<button class="button" type="button" disabled={pending} onclick={close}
			>{action.tone === "warning" ? "Keep request" : "Cancel"}</button>
		<button
			class={action.tone === "neutral"
				? "button-primary"
				: relationshipButtonClass(action.tone)}
			type="button"
			aria-busy={pending}
			disabled={pending}
			onclick={() => void confirm()}>
			<Icon svg={action.icon} size={14} aria-hidden="true" />
			{pending ? action.progress : action.label}
		</button>
	</div>
</Dialog>

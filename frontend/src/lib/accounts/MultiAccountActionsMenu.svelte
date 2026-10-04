<script lang="ts">
	import AtIcon from "phosphor-svelte/lib/AtIcon"
	import CaretRightIcon from "phosphor-svelte/lib/CaretRightIcon"
	import HashIcon from "phosphor-svelte/lib/HashIcon"
	import KeyIcon from "phosphor-svelte/lib/KeyIcon"
	import TagIcon from "phosphor-svelte/lib/TagIcon"
	import UserCircleIcon from "phosphor-svelte/lib/UserCircleIcon"
	import { AccountCopyField } from "../backend/bridge"
	import type { WorkspaceState } from "../layout/workspace-state.svelte"
	import type { AccountStore } from "./account-store.svelte"
	import TagPicker from "./TagPicker.svelte"

	let {
		store,
		workspace,
		submenuLeft,
		onManageTags,
	}: {
		store: AccountStore
		workspace: WorkspaceState
		submenuLeft: boolean
		onManageTags: () => void
	} = $props()
	const copyFields = [
		{
			field: AccountCopyField.CopyDisplayName,
			label: "Display names",
			icon: UserCircleIcon,
		},
		{ field: AccountCopyField.CopyUsername, label: "Usernames", icon: AtIcon },
		{ field: AccountCopyField.CopyUserID, label: "User IDs", icon: HashIcon },
		{ field: AccountCopyField.CopyCookie, label: "Cookies", icon: KeyIcon },
	]

	async function copyField(field: AccountCopyField): Promise<void> {
		const accountIds = store.selectedAccounts.map((account) => account.id)
		if (await store.copyAccountFields(accountIds, field)) {
			workspace.closeAccountMenus()
		}
	}
</script>

<div class="tag-filter-heading">
	<strong>{store.selectedAccountIds.length} accounts selected</strong>
</div>
<div class="tag-submenu-host">
	<button class="tag-submenu-trigger" type="button">
		<TagIcon size={16} aria-hidden="true" />
		<span>Edit tags</span>
		<CaretRightIcon size={15} class="submenu-chevron" aria-hidden="true" />
	</button>
	<div
		class:open-left={submenuLeft}
		class="tag-submenu account-menu"
		data-account-menu>
		<TagPicker {store} accounts={store.selectedAccounts} {onManageTags} />
	</div>
</div>
<div role="group" aria-label="Copy selected account details">
	<div class="account-menu-heading" aria-hidden="true">Copy · one per line</div>
	{#each copyFields as item (item.field)}
		<button
			type="button"
			class:warning-action={item.field === AccountCopyField.CopyCookie}
			aria-label={`Copy ${item.label.toLowerCase()}`}
			disabled={store.copying}
			onclick={() => void copyField(item.field)}>
			<item.icon size={16} aria-hidden="true" /><span>{item.label}</span>
		</button>
	{/each}
</div>

<script lang="ts">
	import atIcon from "@phosphor-icons/core/regular/at.svg?raw"
	import hashIcon from "@phosphor-icons/core/regular/hash.svg?raw"
	import keyIcon from "@phosphor-icons/core/regular/key.svg?raw"
	import linkIcon from "@phosphor-icons/core/regular/link.svg?raw"
	import tagIcon from "@phosphor-icons/core/regular/tag.svg?raw"
	import textTIcon from "@phosphor-icons/core/regular/text-t.svg?raw"
	import MenuItem from "../shared/MenuItem.svelte"
	import MenuSection from "../shared/MenuSection.svelte"
	import MenuTitle from "../shared/MenuTitle.svelte"
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
			icon: textTIcon,
		},
		{ field: AccountCopyField.CopyUsername, label: "Usernames", icon: atIcon },
		{ field: AccountCopyField.CopyUserID, label: "User IDs", icon: hashIcon },
	]
	const selectedAccountIds = $derived(
		store.selectedAccounts.map((account) => account.id),
	)

	async function copyField(field: AccountCopyField): Promise<void> {
		if (await store.copyAccountFields(selectedAccountIds, field)) {
			workspace.closeAccountMenus()
		}
	}

	async function copyProfileLinks(): Promise<void> {
		if (await store.copyAccountProfileLinks(selectedAccountIds)) {
			workspace.closeAccountMenus()
		}
	}
</script>

<MenuTitle count={store.selectedAccountIds.length} noun="accounts" />
<div class="relative">
	<MenuItem label="Edit tags" icon={tagIcon} hasSubmenu />
	<div class="submenu" data-side={submenuLeft ? "left" : undefined} data-account-menu>
		<TagPicker {store} accounts={store.selectedAccounts} {onManageTags} />
	</div>
</div>
<MenuSection
	heading="Copy, one per line"
	accessibleLabel="Copy selected account details">
	{#each copyFields as fieldOption (fieldOption.field)}
		<MenuItem
			label={fieldOption.label}
			icon={fieldOption.icon}
			accessibleLabel={`Copy ${fieldOption.label.toLowerCase()}`}
			disabled={store.copying}
			onSelect={() => void copyField(fieldOption.field)} />
	{/each}
	<MenuItem
		label="Profile links"
		icon={linkIcon}
		accessibleLabel="Copy profile links"
		disabled={store.copying}
		onSelect={() => void copyProfileLinks()} />
	<MenuItem
		label="Cookies"
		icon={keyIcon}
		tone="warning"
		accessibleLabel="Copy cookies"
		disabled={store.copying}
		onSelect={() => void copyField(AccountCopyField.CopyCookie)} />
</MenuSection>

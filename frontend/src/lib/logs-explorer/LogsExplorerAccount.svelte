<script lang="ts">
	import starFillIcon from "@phosphor-icons/core/fill/star-fill.svg?raw"
	import Icon from "../shared/Icon.svelte"
	import ProfileImage from "../shared/ProfileImage.svelte"
	import Skeleton from "../shared/Skeleton.svelte"
	import { accountBackend, TagKind } from "../backend/bridge"

	interface LogUser {
		username: string
		displayName: string
		favorite: boolean
		inVault: boolean | null
	}

	let { userId }: { userId: string } = $props()

	let user = $state<LogUser | null>(null),
		isUserUnavailable = $state(false),
		avatarUrl = $state("")

	$effect(() => {
		const id = Number(userId)
		user = null
		isUserUnavailable = false
		avatarUrl = ""
		if (!Number.isSafeInteger(id) || id <= 0) return
		const state = { current: true }
		void load(id, state)
		void loadAvatar(id, state)
		return () => {
			state.current = false
		}
	})

	async function load(id: number, state: { current: boolean }): Promise<void> {
		let inVault: boolean | null = null
		try {
			const account = await accountBackend.FindAccountByRobloxUserID(id)
			if (!state.current) return
			if (account) {
				user = {
					username: account.username,
					displayName: account.displayName,
					favorite:
						account.tags?.some(
							(tag) => tag.kind === TagKind.TagKindFavorite,
						) ?? false,
					inVault: true,
				}
				return
			}
			inVault = false
		} catch {
			// The Roblox lookup below still names the user. Without the vault
			// lookup, the row leaves out whether the user is in the vault.
		}
		try {
			const found = await accountBackend.GetRobloxUser(id)
			if (state.current) {
				user = {
					username: found.username,
					displayName: found.displayName,
					favorite: false,
					inVault,
				}
			}
		} catch {
			if (state.current) isUserUnavailable = true
		}
	}

	async function loadAvatar(id: number, state: { current: boolean }): Promise<void> {
		try {
			const [headshot] = (await accountBackend.GetAvatarHeadshots([id])) ?? []
			if (state.current) avatarUrl = headshot?.imageUrl ?? ""
		} catch {
			// ProfileImage shows the user icon in place of a missing avatar.
		}
	}
</script>

<div class="flex min-w-0 items-center gap-3" aria-busy={!user && !isUserUnavailable}>
	<span
		class="grid size-10 shrink-0 place-items-center overflow-hidden rounded-md border border-line-strong bg-raised text-fg-faint"
		aria-hidden="true">
		<ProfileImage url={avatarUrl} />
	</span>
	<div class="flex min-w-0 flex-col leading-snug">
		{#if user}
			<strong
				class="inline-flex min-w-0 items-center gap-1 text-sm font-semibold text-fg"
				><span class="truncate">{user.displayName || user.username}</span
				>{#if user.favorite}<Icon
						svg={starFillIcon}
						class="shrink-0 animate-pop text-accent-strong"
						size={12}
						aria-label="Favorite" />{/if}</strong>
			<span class="truncate text-xs text-fg-muted"
				>@{user.username}{user.inVault === false
					? " - Not in your vault"
					: ""}</span>
		{:else if isUserUnavailable}
			<span class="text-sm text-fg-muted">Unavailable</span>
			<span class="truncate font-mono text-xs text-fg-muted"
				>User ID {userId}</span>
		{:else}
			<span class="text-sm"
				><Skeleton className="w-28" label="Loading user…" /></span>
			<span class="text-xs"
				><Skeleton className="w-20" label="Loading username…" /></span>
		{/if}
	</div>
</div>
